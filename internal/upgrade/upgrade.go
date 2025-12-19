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
}

// UpgradeResult contains the result of an upgrade operation
type UpgradeResult struct {
	CurrentVersion string `json:"currentVersion"`
	TargetVersion  string `json:"targetVersion"`
	Updated        bool   `json:"updated"`
	Message        string `json:"message"`
	AssetName      string `json:"assetName,omitempty"`
	Checksum       string `json:"checksum,omitempty"`
	InstallPath    string `json:"installPath,omitempty"`
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
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
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
		return nil, fmt.Errorf("no release asset found for %s/%s (expected: %s)", runtime.GOOS, runtime.GOARCH, assetName)
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

	// Extract binary
	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extract dir: %w", err)
	}

	if err := extractArchive(archivePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find the acc binary in extracted files
	binaryName := "acc"
	if runtime.GOOS == "windows" {
		binaryName = "acc.exe"
	}
	extractedBinary := filepath.Join(extractDir, binaryName)
	if _, err := os.Stat(extractedBinary); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found in archive: %s", binaryName)
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
