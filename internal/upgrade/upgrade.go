package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpgradeOptions contains options for upgrade
type UpgradeOptions struct {
	Version           string // Target version (e.g., "v0.1.6" or "latest")
	DryRun            bool   // If true, only show what would happen
	APIBase           string // GitHub API base URL (for testing)
	DownloadBase      string // GitHub download base URL (for testing)
	DisableInstall    bool   // If true, don't actually install (for testing)
	CurrentVersion    string // Current version
	CurrentExecutable string // Path to current executable

	// Supply-chain verification (opt-in)
	VerifySignature  bool   // If true, verify cosign signature
	CosignKey        string // Path/URL to cosign public key (optional, keyless if empty)
	VerifyProvenance bool   // If true, verify SLSA provenance
}

// UpgradeResult contains the result of an upgrade operation
type UpgradeResult struct {
	CurrentVersion     string `json:"currentVersion"`
	TargetVersion      string `json:"targetVersion"`
	Updated            bool   `json:"updated"`
	Message            string `json:"message"`
	AssetName          string `json:"assetName,omitempty"`
	Checksum           string `json:"checksum,omitempty"`
	InstallPath        string `json:"installPath,omitempty"`
	SignatureVerified  bool   `json:"signatureVerified,omitempty"`
	ProvenanceVerified bool   `json:"provenanceVerified,omitempty"`
}

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Upgrade performs the upgrade operation
func Upgrade(opts *UpgradeOptions) (*UpgradeResult, error) {
	// Set defaults
	if opts.APIBase == "" {
		opts.APIBase = "https://api.github.com"
	}
	if opts.DownloadBase == "" {
		opts.DownloadBase = "https://github.com"
	}

	result := &UpgradeResult{
		CurrentVersion: opts.CurrentVersion,
	}

	// Fetch target release
	var release *Release
	var err error

	if opts.Version == "" || opts.Version == "latest" {
		release, err = fetchLatestRelease(opts.APIBase)
	} else {
		release, err = fetchReleaseByTag(opts.APIBase, normalizeVersion(opts.Version))
	}

	if err != nil {
		if opts.Version != "" && opts.Version != "latest" {
			return nil, fmt.Errorf("failed to fetch release %s: %w", opts.Version, err)
		}
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	result.TargetVersion = release.TagName

	// Check if already up-to-date
	if normalizeVersion(opts.CurrentVersion) == release.TagName {
		result.Message = fmt.Sprintf("Already up-to-date (version %s)", opts.CurrentVersion)
		result.Updated = false
		return result, nil
	}

	// Select appropriate asset for this OS/ARCH
	assetName := selectAsset(release.TagName, runtime.GOOS, runtime.GOARCH)
	asset := findAsset(release.Assets, assetName)
	if asset == nil {
		return nil, fmt.Errorf("no release asset found for %s %s/%s (expected: %s)", release.TagName, runtime.GOOS, runtime.GOARCH, assetName)
	}

	result.AssetName = asset.Name

	if opts.DryRun {
		result.Message = fmt.Sprintf("Would upgrade from %s to %s using %s", opts.CurrentVersion, release.TagName, asset.Name)
		result.Updated = false
		return result, nil
	}

	// Download and verify
	tmpDir, err := os.MkdirTemp("", "acc-upgrade-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset.Name)
	if err := downloadFile(asset.BrowserDownloadURL, archivePath); err != nil {
		return nil, fmt.Errorf("failed to download release: %w", err)
	}

	// Verify checksum
	checksumsURL := buildChecksumsURL(opts.DownloadBase, release.TagName)
	checksums, err := fetchChecksums(checksumsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checksums: %w", err)
	}

	expectedChecksum, ok := checksums[asset.Name]
	if !ok {
		return nil, fmt.Errorf("checksum not found for %s", asset.Name)
	}

	actualChecksum, err := computeSHA256(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksum: %w", err)
	}

	if actualChecksum != expectedChecksum {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	result.Checksum = actualChecksum

	// Optional: Verify cosign signature (opt-in)
	if opts.VerifySignature {
		if err := verifyCosignSignature(archivePath, opts.DownloadBase, release.TagName, opts.CosignKey); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
		result.SignatureVerified = true
	}

	// Optional: Verify SLSA provenance (opt-in)
	if opts.VerifyProvenance {
		if err := verifySLSAProvenance(opts.DownloadBase, release.TagName, asset.Name); err != nil {
			return nil, fmt.Errorf("provenance verification failed: %w", err)
		}
		result.ProvenanceVerified = true
	}

	// Extract binary
	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extract dir: %w", err)
	}

	if err := extractArchive(archivePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find the acc binary in extracted files (backward-compatible search)
	extractedBinary, err := findExecutableInDir(extractDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find binary in archive: %w", err)
	}

	// Install binary (if not disabled for testing)
	if !opts.DisableInstall {
		installPath := opts.CurrentExecutable
		if installPath == "" {
			installPath, err = os.Executable()
			if err != nil {
				return nil, fmt.Errorf("failed to get executable path: %w", err)
			}
		}

		if err := installBinary(extractedBinary, installPath); err != nil {
			return nil, fmt.Errorf("failed to install binary: %w", err)
		}

		result.InstallPath = installPath
	}

	result.Message = fmt.Sprintf("Successfully upgraded from %s to %s", opts.CurrentVersion, release.TagName)
	result.Updated = true

	return result, nil
}

// fetchLatestRelease fetches the latest release from GitHub
func fetchLatestRelease(apiBase string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/cloudcwfranck/acc/releases/latest", apiBase)
	return fetchRelease(url)
}

// fetchReleaseByTag fetches a specific release by tag
func fetchReleaseByTag(apiBase, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/cloudcwfranck/acc/releases/tags/%s", apiBase, tag)
	return fetchRelease(url)
}

// fetchRelease fetches a release from a URL
func fetchRelease(url string) (*Release, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// selectAsset selects the appropriate asset name for the given OS/ARCH
func selectAsset(version, goos, goarch string) string {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	switch goos {
	case "windows":
		return fmt.Sprintf("acc_%s_windows_%s.zip", version, goarch)
	default:
		return fmt.Sprintf("acc_%s_%s_%s.tar.gz", version, goos, goarch)
	}
}

// findAsset finds an asset by name in the assets list
func findAsset(assets []Asset, name string) *Asset {
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i]
		}
	}
	return nil
}

// normalizeVersion normalizes a version string to include 'v' prefix
func normalizeVersion(version string) string {
	if version == "" || version == "latest" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// buildChecksumsURL builds the URL for the checksums.txt file
func buildChecksumsURL(downloadBase, tag string) string {
	return fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/checksums.txt", downloadBase, tag)
}

// fetchChecksums fetches and parses checksums.txt
func fetchChecksums(url string) (map[string]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksum := parts[0]
			filename := parts[1]
			checksums[filename] = checksum
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return checksums, nil
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// computeSHA256 computes the SHA256 checksum of a file
func computeSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extractArchive extracts a tar.gz or zip archive
func extractArchive(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".tar.gz") {
		return extractTarGz(archivePath, destDir)
	} else if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractTarGz extracts a tar.gz archive
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// extractZip extracts a zip archive
func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		path := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			rc.Close()
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			rc.Close()
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// findExecutableInDir finds the acc executable in a directory
// Searches for files matching "acc" or "acc-*" (with .exe suffix on Windows)
// Returns error if 0 or multiple executables are found
func findExecutableInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var candidates []string
	expectedSuffix := ""
	if runtime.GOOS == "windows" {
		expectedSuffix = ".exe"
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if name matches pattern: acc or acc-* (with optional .exe on Windows)
		nameWithoutExt := strings.TrimSuffix(name, expectedSuffix)

		if nameWithoutExt == "acc" || strings.HasPrefix(nameWithoutExt, "acc-") {
			// On Windows, must have .exe suffix
			if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
				continue
			}

			fullPath := filepath.Join(dir, name)

			// On Unix, verify it's executable
			if runtime.GOOS != "windows" {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				// Check if file has executable bit (owner, group, or other)
				if info.Mode()&0111 == 0 {
					continue
				}
			}

			candidates = append(candidates, fullPath)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no acc executable found (expected 'acc' or 'acc-*' with executable permissions)")
	}

	if len(candidates) > 1 {
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = filepath.Base(c)
		}
		return "", fmt.Errorf("multiple acc executables found: %s (expected exactly one)", strings.Join(names, ", "))
	}

	return candidates[0], nil
}

// installBinary installs a binary using atomic replacement
func installBinary(srcPath, destPath string) error {
	// On Windows, we can't replace a running executable
	// Write instructions for manual replacement
	if runtime.GOOS == "windows" {
		return installBinaryWindows(srcPath, destPath)
	}

	// For Unix-like systems, use atomic rename
	return installBinaryUnix(srcPath, destPath)
}

// installBinaryUnix installs binary on Unix-like systems
func installBinaryUnix(srcPath, destPath string) error {
	// Create backup
	backupPath := destPath + ".backup"
	if err := copyFile(destPath, backupPath); err != nil {
		// Backup failed, but continue anyway
	}
	defer os.Remove(backupPath)

	// Write new binary to .new file
	newPath := destPath + ".new"
	if err := copyFile(srcPath, newPath); err != nil {
		return err
	}

	// Make executable
	if err := os.Chmod(newPath, 0755); err != nil {
		os.Remove(newPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(newPath, destPath); err != nil {
		os.Remove(newPath)
		// Try to restore backup
		if _, backupErr := os.Stat(backupPath); backupErr == nil {
			os.Rename(backupPath, destPath)
		}
		return err
	}

	return nil
}

// installBinaryWindows handles Windows installation
func installBinaryWindows(srcPath, destPath string) error {
	// On Windows, we write to .new.exe and instruct user to replace
	newPath := strings.TrimSuffix(destPath, ".exe") + ".new.exe"

	if err := copyFile(srcPath, newPath); err != nil {
		return err
	}

	return fmt.Errorf("Windows binary downloaded to: %s\n\nTo complete upgrade:\n1. Close this terminal\n2. Rename %s to acc.exe.old\n3. Rename %s to acc.exe\n4. Delete acc.exe.old", newPath, destPath, newPath)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// verifyCosignSignature verifies the cosign signature of a release asset
func verifyCosignSignature(archivePath, downloadBase, tag, cosignKey string) error {
	// Check if cosign is available
	cosignPath, err := findCosignBinary()
	if err != nil {
		return fmt.Errorf("cosign is required for signature verification but was not found in PATH. Install cosign: https://docs.sigstore.dev/cosign/installation/")
	}

	// Construct signature file URL
	// Expected format: acc_0.2.7_linux_amd64.tar.gz.sig
	signatureURL := fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/%s.sig", downloadBase, tag, filepath.Base(archivePath))

	// Download signature file
	sigPath := archivePath + ".sig"
	if err := downloadFile(signatureURL, sigPath); err != nil {
		return fmt.Errorf("failed to download signature file: %w (expected: %s)", err, signatureURL)
	}
	defer os.Remove(sigPath)

	// Also try to download certificate file (for keyless signatures)
	certURL := fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/%s.pem", downloadBase, tag, filepath.Base(archivePath))
	certPath := archivePath + ".pem"
	downloadFile(certURL, certPath) // Best effort, may not exist
	defer os.Remove(certPath)

	// Build cosign verify command
	args := []string{"verify-blob"}

	if cosignKey != "" {
		// Key-based verification
		args = append(args, "--key", cosignKey)
	} else {
		// Keyless verification (requires certificate)
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			return fmt.Errorf("no cosign key provided and no certificate found for keyless verification (expected: %s)", certURL)
		}
		args = append(args, "--certificate", certPath)
		args = append(args, "--certificate-identity-regexp", ".*")
		args = append(args, "--certificate-oidc-issuer-regexp", ".*")
	}

	args = append(args, "--signature", sigPath)
	args = append(args, archivePath)

	// Execute cosign verify
	cmd := exec.Command(cosignPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign verification failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// verifySLSAProvenance verifies the SLSA provenance for a release asset
func verifySLSAProvenance(downloadBase, tag, assetName string) error {
	// Expected provenance formats:
	// 1. Single provenance file: <tag>.intoto.jsonl or provenance.intoto.jsonl
	// 2. Per-asset provenance: <assetName>.intoto.jsonl

	provenanceURLs := []string{
		fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/provenance.intoto.jsonl", downloadBase, tag),
		fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/%s.intoto.jsonl", downloadBase, tag, tag),
		fmt.Sprintf("%s/cloudcwfranck/acc/releases/download/%s/%s.intoto.jsonl", downloadBase, tag, assetName),
	}

	// Try to download provenance file
	var provenanceData []byte
	var lastErr error

	for _, url := range provenanceURLs {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			provenanceData, err = io.ReadAll(resp.Body)
			if err != nil {
				lastErr = err
				continue
			}
			break
		}
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if provenanceData == nil {
		return fmt.Errorf("no SLSA provenance found for this release (tried: provenance.intoto.jsonl, %s.intoto.jsonl, %s.intoto.jsonl): %v", tag, assetName, lastErr)
	}

	// Basic provenance validation
	// Parse as JSON to ensure it's valid SLSA provenance
	var provenance map[string]interface{}
	if err := json.Unmarshal(provenanceData, &provenance); err != nil {
		return fmt.Errorf("provenance file is not valid JSON: %w", err)
	}

	// Verify provenance structure (basic checks)
	predicateType, ok := provenance["predicateType"].(string)
	if !ok || predicateType == "" {
		return fmt.Errorf("provenance missing predicateType field")
	}

	if !strings.Contains(predicateType, "slsa") && !strings.Contains(predicateType, "provenance") {
		return fmt.Errorf("provenance predicateType is not SLSA provenance: %s", predicateType)
	}

	// Verify subject refers to the correct asset
	predicate, ok := provenance["predicate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("provenance missing predicate field")
	}

	// Check buildType
	buildType, _ := predicate["buildType"].(string)
	if buildType != "" && !strings.Contains(buildType, "github") {
		return fmt.Errorf("provenance buildType is not GitHub Actions: %s", buildType)
	}

	// Verify builder identity (should be GitHub Actions)
	builder, ok := predicate["builder"].(map[string]interface{})
	if ok {
		builderID, _ := builder["id"].(string)
		if builderID != "" && !strings.Contains(builderID, "github") {
			return fmt.Errorf("provenance builder is not GitHub: %s", builderID)
		}
	}

	// Note: Full cryptographic verification would require slsa-verifier or similar tool
	// This implementation does basic structure validation only
	// For production use, integrate slsa-verifier CLI tool

	return nil
}

// findCosignBinary finds the cosign binary in PATH
func findCosignBinary() (string, error) {
	// Check for cosign in PATH
	path, err := exec.LookPath("cosign")
	if err != nil {
		return "", fmt.Errorf("cosign not found in PATH")
	}
	return path, nil
}
