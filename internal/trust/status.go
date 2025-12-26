package trust

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/ui"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// StatusResult represents the trust status output
type StatusResult struct {
	SchemaVersion string      `json:"schemaVersion"`
	ImageRef      string      `json:"imageRef"`
	Status        string      `json:"status"` // pass, fail, unknown
	ProfileUsed   string      `json:"profileUsed,omitempty"`
	Violations    []Violation `json:"violations"`
	Warnings      []Violation `json:"warnings"`
	SBOMPresent   bool        `json:"sbomPresent"`
	Attestations  []string    `json:"attestations"`
	Timestamp     string      `json:"timestamp"`
}

// Violation represents a policy violation
type Violation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Result   string `json:"result"`
	Message  string `json:"message"`
}

// VerifyState represents persisted verification state
type VerifyState struct {
	ImageRef    string                 `json:"imageRef"`
	Status      string                 `json:"status"`
	Timestamp   string                 `json:"timestamp"`
	ProfileUsed string                 `json:"profileUsed,omitempty"`
	Result      map[string]interface{} `json:"result"`
}

// Status loads and displays the trust status for an image
// v0.3.2: optionally fetch attestations from remote registry when remote=true
func Status(imageRef string, remote, outputJSON bool) (*StatusResult, error) {
	// Load verification state
	state, err := loadVerifyState(imageRef)
	if err != nil {
		// v0.2.7: Always return result with status "unknown" (exit code 2)
		// instead of error (exit code 1) when state not found
		result := &StatusResult{
			SchemaVersion: "v0.2",
			ImageRef:      imageRef,
			Status:        "unknown",
			SBOMPresent:   false, // v0.2.7: Ensure always set
			Violations:    []Violation{},
			Warnings:      []Violation{},
			Attestations:  []string{},
			Timestamp:     "", // v0.2.7: Empty string for unknown state
		}

		if !outputJSON {
			fmt.Fprintf(os.Stderr, "Warning: No verification state found for image: %s\n", imageRef)
			fmt.Fprintf(os.Stderr, "Remediation: Run 'acc verify %s' first\n\n", imageRef)
		}

		return result, nil
	}

	// Resolve digest for per-image attestation lookup
	digest, _ := resolveImageDigest(imageRef)

	// Build result from state (v0.2.7: ensure all fields initialized)
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      state.ImageRef,
		Status:        state.Status,
		ProfileUsed:   state.ProfileUsed,
		Timestamp:     state.Timestamp,
		SBOMPresent:   false, // Default, will be set below
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
	}

	// Extract violations and warnings from result
	if stateResult, ok := state.Result["policyResult"].(map[string]interface{}); ok {
		if violations, ok := stateResult["violations"].([]interface{}); ok {
			for _, v := range violations {
				if vm, ok := v.(map[string]interface{}); ok {
					result.Violations = append(result.Violations, Violation{
						Rule:     getString(vm, "rule"),
						Severity: getString(vm, "severity"),
						Result:   getString(vm, "result"),
						Message:  getString(vm, "message"),
					})
				}
			}
		}
		if warnings, ok := stateResult["warnings"].([]interface{}); ok {
			for _, w := range warnings {
				if wm, ok := w.(map[string]interface{}); ok {
					result.Warnings = append(result.Warnings, Violation{
						Rule:     getString(wm, "rule"),
						Severity: getString(wm, "severity"),
						Result:   getString(wm, "result"),
						Message:  getString(wm, "message"),
					})
				}
			}
		}
	}

	// Check SBOM (v0.2.7: ensure always set as boolean)
	if sbomPresent, ok := state.Result["sbomPresent"].(bool); ok {
		result.SBOMPresent = sbomPresent
	} else {
		result.SBOMPresent = false
	}

	// v0.3.2: Optionally fetch remote attestations before finding local ones
	if remote && digest != "" {
		if err := fetchRemoteAttestations(imageRef, digest, outputJSON); err != nil {
			// Remote fetch failed - log warning but don't fail
			// This preserves local-only workflow when network unavailable
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Warning: Failed to fetch remote attestations: %v\n", err)
			}
		}
	}

	// v0.2.7: Find attestations for this specific image (per-image isolation)
	// v0.3.2: This now includes both local and remote-cached attestations
	result.Attestations = findAttestationsForImage(digest)

	// Output results
	if outputJSON {
		return result, nil
	}

	// Human-readable output
	printHumanStatus(result)
	return result, nil
}

// loadVerifyState loads verification state for an image
// First tries digest-scoped state, falls back to global state
func loadVerifyState(imageRef string) (*VerifyState, error) {
	// Try to resolve digest for digest-scoped lookup
	digest, _ := resolveImageDigest(imageRef)
	if digest != "" {
		digestFile := filepath.Join(".acc", "state", "verify", digest+".json")
		if data, err := os.ReadFile(digestFile); err == nil {
			var state VerifyState
			if err := json.Unmarshal(data, &state); err == nil {
				return &state, nil
			}
		}
	}

	// Fall back to global last_verify.json
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state VerifyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// CRITICAL: Verify the state is for the requested image
	// Without this check, different images can share state (cross-image leakage)
	if state.ImageRef != imageRef {
		return nil, fmt.Errorf("state mismatch: found state for %s, requested %s", state.ImageRef, imageRef)
	}

	return &state, nil
}

// resolveImageDigest tries to resolve image digest using container tools
// v0.2.1: Fixed to properly query Docker/Podman/nerdctl for digest
func resolveImageDigest(imageRef string) (string, error) {
	// First, try to extract from imageRef if it contains @sha256:
	if strings.Contains(imageRef, "@sha256:") {
		parts := strings.Split(imageRef, "@sha256:")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	// Try different container tools to get the digest
	tools := []struct {
		name string
		args []string
	}{
		{"docker", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"podman", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"nerdctl", []string{"inspect", "--format={{.Id}}", imageRef}},
	}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool.name); err == nil {
			cmd := exec.Command(tool.name, tool.args...)
			output, err := cmd.Output()
			if err == nil {
				digest := strings.TrimSpace(string(output))
				digest = strings.TrimPrefix(digest, "sha256:")
				if digest != "" {
					return digest, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not resolve digest for %s", imageRef)
}

// findAttestations looks for attestation files (all images)
func findAttestations() []string {
	attestDir := filepath.Join(".acc", "attestations")
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return []string{}
	}

	var attestations []string
	filepath.Walk(attestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			attestations = append(attestations, path)
		}
		return nil
	})

	return attestations
}

// findAttestationsForImage looks for attestation files for a specific image digest
// v0.2.7: Per-image attestation isolation - only returns attestations for this digest
func findAttestationsForImage(digest string) []string {
	if digest == "" {
		// If digest not available, return all attestations as fallback
		return findAttestations()
	}

	// Use first 12 chars of digest to match directory structure
	digestPrefix := digest
	if len(digest) > 12 {
		digestPrefix = digest[:12]
	}

	attestDir := filepath.Join(".acc", "attestations", digestPrefix)
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return []string{}
	}

	var attestations []string
	filepath.Walk(attestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			attestations = append(attestations, path)
		}
		return nil
	})

	return attestations
}

// printHumanStatus prints human-readable trust status
func printHumanStatus(result *StatusResult) {
	ui.PrintTrust("Trust Status")
	fmt.Println()

	// Image information
	fmt.Printf("Image:          %s\n", result.ImageRef)
	fmt.Printf("Last Verified:  %s\n", result.Timestamp)
	fmt.Println()

	// Verification status
	statusIcon := "❓"
	switch result.Status {
	case "pass":
		statusIcon = ui.SymbolSuccess
	case "fail":
		statusIcon = ui.SymbolFailure
	case "warn":
		statusIcon = ui.SymbolWarning
	}
	fmt.Printf("Status:         %s %s\n", statusIcon, strings.ToUpper(result.Status))

	// v0.2.0: Show profile if used
	if result.ProfileUsed != "" {
		fmt.Printf("Profile:        %s\n", result.ProfileUsed)
	}
	fmt.Println()

	// Artifacts
	fmt.Println("Artifacts:")
	if result.SBOMPresent {
		ui.PrintSuccess("  SBOM:         present")
	} else {
		ui.PrintWarning("  SBOM:         not found")
	}

	if len(result.Attestations) > 0 {
		ui.PrintSuccess(fmt.Sprintf("  Attestations: %d found", len(result.Attestations)))
	} else {
		ui.PrintWarning("  Attestations: none")
	}
	fmt.Println()

	// v0.2.0: Show violations and warnings separately
	if len(result.Violations) > 0 {
		fmt.Println("Policy Violations:")
		for _, v := range result.Violations {
			ui.PrintError(fmt.Sprintf("  [%s] %s: %s", v.Severity, v.Rule, v.Message))
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Println(fmt.Sprintf("Warnings (%d ignored):", len(result.Warnings)))
		for _, w := range result.Warnings {
			ui.PrintWarning(fmt.Sprintf("  [%s] %s: %s", w.Severity, w.Rule, w.Message))
		}
		fmt.Println()
	}

	if len(result.Violations) == 0 && len(result.Warnings) == 0 && result.Status == "pass" {
		ui.PrintSuccess("No policy violations")
	}
}

// FormatJSON formats status result as JSON
func (sr *StatusResult) FormatJSON() string {
	data, _ := json.MarshalIndent(sr, "", "  ")
	return string(data)
}

// ExitCode returns appropriate exit code
// PRESERVED BEHAVIOR: 0=pass, 1=fail/warn, 2=unknown
func (sr *StatusResult) ExitCode() int {
	if sr.Status == "unknown" {
		return 2
	}
	if sr.Status == "pass" {
		return 0
	}
	// fail or warn
	return 1
}

// getString safely extracts string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// fetchRemoteAttestations fetches attestations from a remote OCI registry and caches them locally
// v0.3.2: Real OCI attestation fetching using oras-go/v2
func fetchRemoteAttestations(imageRef, digest string, outputJSON bool) error {
	ctx := context.Background()

	// 1. Parse image reference to get registry and repository
	registry, repository, _, err := parseImageRef(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	// 2. Create OCI repository client with auth
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", registry, repository))
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	// Configure auth from Docker credentials
	// Try multiple registry key formats that might be in Docker config
	var cred auth.Credential
	var credErr error

	// Try different registry URL formats
	registryFormats := []string{
		registry,                       // e.g., "ghcr.io"
		"https://" + registry,          // e.g., "https://ghcr.io"
		"https://" + registry + "/v2/", // e.g., "https://ghcr.io/v2/"
	}

	for _, regFormat := range registryFormats {
		if c, err := loadDockerCredentials(regFormat); err == nil {
			cred = c
			credErr = nil
			break
		} else {
			credErr = err
		}
	}

	// Set up auth client with credentials if found
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.CredentialFunc(func(ctx context.Context, reg string) (auth.Credential, error) {
			if credErr != nil {
				// No credentials found, return empty (might work for public repos or with other auth methods)
				return auth.Credential{}, nil
			}
			return cred, nil
		}),
	}
	repo.PlainHTTP = false

	// 3. List tags matching our attestation naming pattern
	// Pattern: attestation-<digest-prefix>-*
	digestPrefix := digest
	if len(digest) > 12 {
		digestPrefix = digest[:12]
	}
	attestationPrefix := fmt.Sprintf("attestation-%s-", digestPrefix)

	// List all tags
	var attestationTags []string
	err = repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			if strings.HasPrefix(tag, attestationPrefix) {
				attestationTags = append(attestationTags, tag)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	if len(attestationTags) == 0 {
		// No remote attestations found - not an error
		if !outputJSON {
			ui.PrintWarning("No remote attestations found")
		}
		return nil
	}

	// 4. Pull each attestation and cache it
	fetchedCount := 0
	for _, tag := range attestationTags {
		// Resolve tag to descriptor
		desc, err := repo.Resolve(ctx, tag)
		if err != nil {
			if !outputJSON {
				ui.PrintWarning(fmt.Sprintf("Failed to resolve tag %s: %v", tag, err))
			}
			continue
		}

		// Fetch attestation content
		reader, err := repo.Fetch(ctx, desc)
		if err != nil {
			if !outputJSON {
				ui.PrintWarning(fmt.Sprintf("Failed to fetch attestation %s: %v", tag, err))
			}
			continue
		}

		attestationData, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}

		// 5. Cache attestation locally
		// Path: .acc/attestations/<digest-prefix>/remote/<registry>/<repo>/<hash>.json
		cacheDir := filepath.Join(".acc", "attestations", digestPrefix, "remote", registry, repository)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}

		// Use hash of attestation content as filename for deduplication
		attestationHash := fmt.Sprintf("%x", sha256.Sum256(attestationData))
		cachePath := filepath.Join(cacheDir, attestationHash[:16]+".json")

		// Check if already cached
		if _, err := os.Stat(cachePath); err == nil {
			continue // Already cached
		}

		// Write to cache
		if err := os.WriteFile(cachePath, attestationData, 0644); err != nil {
			return fmt.Errorf("failed to write attestation cache: %w", err)
		}

		fetchedCount++
	}

	if !outputJSON && fetchedCount > 0 {
		ui.PrintSuccess(fmt.Sprintf("Fetched %d remote attestation(s)", fetchedCount))
	}

	return nil
}

// parseImageRef parses an image reference into registry, repository, and reference
func parseImageRef(imageRef string) (registry, repository, reference string, err error) {
	// Handle image references like:
	// - localhost:5000/repo:tag
	// - ghcr.io/org/repo:tag
	// - ghcr.io/org/repo@sha256:...
	parts := strings.SplitN(imageRef, "/", 2)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid image reference format: %s", imageRef)
	}

	registry = parts[0]
	rest := parts[1]

	// Split repository and reference (tag or digest)
	if strings.Contains(rest, "@") {
		repoParts := strings.SplitN(rest, "@", 2)
		repository = repoParts[0]
		reference = repoParts[1]
	} else if strings.Contains(rest, ":") {
		repoParts := strings.SplitN(rest, ":", 2)
		repository = repoParts[0]
		reference = repoParts[1]
	} else {
		repository = rest
		reference = "latest"
	}

	return registry, repository, reference, nil
}

// loadDockerCredentials loads credentials from ~/.docker/config.json
func loadDockerCredentials(registry string) (auth.Credential, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return auth.Credential{}, err
	}

	configPath := filepath.Join(homeDir, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return auth.Credential{}, err
	}

	var config struct {
		Auths map[string]struct {
			Auth     string `json:"auth"`
			Username string `json:"username,omitempty"`
			Password string `json:"password,omitempty"`
		} `json:"auths"`
		CredsStore  string            `json:"credsStore,omitempty"`
		CredHelpers map[string]string `json:"credHelpers,omitempty"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return auth.Credential{}, err
	}

	// Look for registry auth
	if authEntry, ok := config.Auths[registry]; ok {
		// Try direct username/password first
		if authEntry.Username != "" && authEntry.Password != "" {
			return auth.Credential{
				Username: authEntry.Username,
				Password: authEntry.Password,
			}, nil
		}

		// Try base64-encoded auth
		if authEntry.Auth != "" {
			decoded, err := base64.StdEncoding.DecodeString(authEntry.Auth)
			if err != nil {
				return auth.Credential{}, fmt.Errorf("failed to decode auth: %w", err)
			}

			// Auth is in format "username:password"
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				return auth.Credential{}, fmt.Errorf("invalid auth format")
			}

			return auth.Credential{
				Username: parts[0],
				Password: parts[1],
			}, nil
		}
	}

	// TODO: Support credential helpers via credsStore and credHelpers
	// For now, if we can't find credentials, try to use external credential helpers
	// by executing docker-credential-<helper> commands

	return auth.Credential{}, fmt.Errorf("no credentials found for %s", registry)
}
