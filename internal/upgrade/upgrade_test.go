package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSelectAsset tests asset name selection for different OS/ARCH combinations
func TestSelectAsset(t *testing.T) {
	tests := []struct {
		version string
		goos    string
		goarch  string
		want    string
	}{
		{"v0.1.6", "linux", "amd64", "acc_0.1.6_linux_amd64.tar.gz"},
		{"v0.1.6", "darwin", "arm64", "acc_0.1.6_darwin_arm64.tar.gz"},
		{"v0.1.6", "darwin", "amd64", "acc_0.1.6_darwin_amd64.tar.gz"},
		{"v0.1.6", "windows", "amd64", "acc_0.1.6_windows_amd64.zip"},
		{"0.1.6", "linux", "amd64", "acc_0.1.6_linux_amd64.tar.gz"},
	}

	for _, tt := range tests {
		got := selectAsset(tt.version, tt.goos, tt.goarch)
		if got != tt.want {
			t.Errorf("selectAsset(%q, %q, %q) = %q, want %q",
				tt.version, tt.goos, tt.goarch, got, tt.want)
		}
	}
}

// TestNormalizeVersion tests version normalization
func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0.1.6", "v0.1.6"},
		{"v0.1.6", "v0.1.6"},
		{"latest", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestFetchRelease tests fetching a release from mock GitHub API
func TestFetchRelease(t *testing.T) {
	mockResponse := `{
		"tag_name": "v0.1.6",
		"name": "Release v0.1.6",
		"assets": [
			{
				"name": "acc_0.1.6_linux_amd64.tar.gz",
				"browser_download_url": "https://example.com/acc_0.1.6_linux_amd64.tar.gz"
			},
			{
				"name": "acc_0.1.6_darwin_arm64.tar.gz",
				"browser_download_url": "https://example.com/acc_0.1.6_darwin_arm64.tar.gz"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	release, err := fetchRelease(server.URL)
	if err != nil {
		t.Fatalf("fetchRelease failed: %v", err)
	}

	if release.TagName != "v0.1.6" {
		t.Errorf("TagName = %q, want v0.1.6", release.TagName)
	}

	if len(release.Assets) != 2 {
		t.Errorf("len(Assets) = %d, want 2", len(release.Assets))
	}
}

// TestFetchReleaseNotFound tests 404 response
func TestFetchReleaseNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchRelease(server.URL)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

// TestFetchChecksums tests checksum fetching and parsing
func TestFetchChecksums(t *testing.T) {
	mockChecksums := `abc123def456  acc_0.1.6_linux_amd64.tar.gz
789xyz012uvw  acc_0.1.6_darwin_arm64.tar.gz

# This is a comment
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockChecksums))
	}))
	defer server.Close()

	checksums, err := fetchChecksums(server.URL)
	if err != nil {
		t.Fatalf("fetchChecksums failed: %v", err)
	}

	if len(checksums) != 2 {
		t.Errorf("len(checksums) = %d, want 2", len(checksums))
	}

	if checksums["acc_0.1.6_linux_amd64.tar.gz"] != "abc123def456" {
		t.Errorf("checksum mismatch for linux asset")
	}

	if checksums["acc_0.1.6_darwin_arm64.tar.gz"] != "789xyz012uvw" {
		t.Errorf("checksum mismatch for darwin asset")
	}
}

// TestUpgradeAlreadyLatest tests already up-to-date scenario
func TestUpgradeAlreadyLatest(t *testing.T) {
	mockResponse := `{
		"tag_name": "v0.1.5",
		"name": "Release v0.1.5",
		"assets": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	opts := &UpgradeOptions{
		Version:        "latest",
		CurrentVersion: "v0.1.5",
		APIBase:        server.URL,
		DisableInstall: true,
	}

	result, err := Upgrade(opts)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	if result.Updated {
		t.Error("Expected Updated = false for already-latest")
	}

	if result.TargetVersion != "v0.1.5" {
		t.Errorf("TargetVersion = %q, want v0.1.5", result.TargetVersion)
	}
}

// TestUpgradeDryRun tests dry-run mode
func TestUpgradeDryRun(t *testing.T) {
	mockResponse := `{
		"tag_name": "v0.1.6",
		"name": "Release v0.1.6",
		"assets": [
			{
				"name": "acc_0.1.6_` + runtime.GOOS + `_` + runtime.GOARCH + `.tar.gz",
				"browser_download_url": "https://example.com/acc.tar.gz"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	opts := &UpgradeOptions{
		Version:        "latest",
		CurrentVersion: "v0.1.5",
		APIBase:        server.URL,
		DryRun:         true,
	}

	result, err := Upgrade(opts)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	if result.Updated {
		t.Error("Expected Updated = false for dry-run")
	}

	if result.TargetVersion != "v0.1.6" {
		t.Errorf("TargetVersion = %q, want v0.1.6", result.TargetVersion)
	}

	expectedAsset := fmt.Sprintf("acc_0.1.6_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	if result.AssetName != expectedAsset {
		t.Errorf("AssetName = %q, want %q", result.AssetName, expectedAsset)
	}
}

// TestUpgradeAssetNotFound tests missing asset error
func TestUpgradeAssetNotFound(t *testing.T) {
	mockResponse := `{
		"tag_name": "v0.1.6",
		"name": "Release v0.1.6",
		"assets": []
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	opts := &UpgradeOptions{
		Version:        "latest",
		CurrentVersion: "v0.1.5",
		APIBase:        server.URL,
		DisableInstall: true,
	}

	_, err := Upgrade(opts)
	if err == nil {
		t.Error("Expected error for missing asset, got nil")
	}

	if !containsStr(err.Error(), "no release asset found") {
		t.Errorf("Expected 'no release asset found' error, got: %v", err)
	}
}

// TestComputeSHA256 tests checksum computation
func TestComputeSHA256(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-checksum-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	checksum, err := computeSHA256(testFile)
	if err != nil {
		t.Fatalf("computeSHA256 failed: %v", err)
	}

	// "hello world" SHA256
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if checksum != expected {
		t.Errorf("checksum = %q, want %q", checksum, expected)
	}
}

// TestExtractTarGz tests tar.gz extraction
func TestExtractTarGz(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-extract-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test tar.gz with an "acc" binary
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	if err := createTestTarGz(archivePath, "acc", []byte("fake binary")); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	if err := extractTarGz(archivePath, extractDir); err != nil {
		t.Fatalf("extractTarGz failed: %v", err)
	}

	extractedFile := filepath.Join(extractDir, "acc")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Error("Expected extracted file to exist")
	}

	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(content) != "fake binary" {
		t.Errorf("extracted content = %q, want 'fake binary'", string(content))
	}
}

// Helper: create a test tar.gz archive
func createTestTarGz(path, filename string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	header := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write(content); err != nil {
		return err
	}

	return nil
}

// Helper: check if string contains substring
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsStrRec(s, substr))
}

func containsStrRec(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
