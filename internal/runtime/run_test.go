package runtime

import (
	"strings"
	"testing"
)

func TestBuildRunCommand(t *testing.T) {
	tests := []struct {
		name     string
		runtime  string
		opts     *RunOptions
		mustHave []string
		mustNot  []string
	}{
		{
			name:    "basic docker run with minimal options",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef: "demo-app:v1",
				Args:     []string{"echo", "test"},
			},
			mustHave: []string{"docker", "run", "--rm", "demo-app:v1", "echo", "test"},
			mustNot:  []string{},
		},
		{
			name:    "docker run with user",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef: "demo-app:v1",
				User:     "1000:1000",
				Args:     []string{"sh"},
			},
			mustHave: []string{"docker", "run", "--rm", "--user", "1000:1000", "demo-app:v1", "sh"},
			mustNot:  []string{},
		},
		{
			name:    "docker run with network none (default)",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef:    "demo-app:v1",
				NetworkMode: "",
				Args:        []string{},
			},
			mustHave: []string{"docker", "run", "--rm", "--network", "none", "demo-app:v1"},
			mustNot:  []string{},
		},
		{
			name:    "docker run with custom network",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef:    "demo-app:v1",
				NetworkMode: "bridge",
				Args:        []string{},
			},
			mustHave: []string{"docker", "run", "--rm", "--network", "bridge", "demo-app:v1"},
			mustNot:  []string{},
		},
		{
			name:    "docker run with read-only filesystem",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef: "demo-app:v1",
				ReadOnly: true,
				Args:     []string{},
			},
			mustHave: []string{"docker", "run", "--rm", "--read-only", "demo-app:v1"},
			mustNot:  []string{},
		},
		{
			name:    "docker run with capabilities",
			runtime: "docker",
			opts: &RunOptions{
				ImageRef:     "demo-app:v1",
				Capabilities: []string{"NET_BIND_SERVICE"},
				Args:         []string{},
			},
			mustHave: []string{"docker", "run", "--rm", "--cap-drop", "ALL", "--cap-add", "NET_BIND_SERVICE", "demo-app:v1"},
			mustNot:  []string{},
		},
		{
			name:    "podman run with security options",
			runtime: "podman",
			opts: &RunOptions{
				ImageRef: "demo-app:v1",
				Args:     []string{},
			},
			mustHave: []string{"podman", "run", "--rm", "--security-opt", "no-new-privileges", "demo-app:v1"},
			mustNot:  []string{},
		},
		{
			name:    "nerdctl run without docker-specific options",
			runtime: "nerdctl",
			opts: &RunOptions{
				ImageRef:     "demo-app:v1",
				Capabilities: []string{"NET_ADMIN"},
				Args:         []string{},
			},
			mustHave: []string{"nerdctl", "run", "--rm", "demo-app:v1"},
			// nerdctl doesn't get cap-drop/cap-add in current implementation
			mustNot: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRunCommand(tt.runtime, tt.opts)
			cmdStr := strings.Join(result, " ")

			// Check that all required strings are present
			for _, mustHave := range tt.mustHave {
				if !contains(result, mustHave) {
					t.Errorf("Expected command to contain %q, but got: %s", mustHave, cmdStr)
				}
			}

			// Check that prohibited strings are not present
			for _, mustNot := range tt.mustNot {
				if contains(result, mustNot) {
					t.Errorf("Expected command to NOT contain %q, but got: %s", mustNot, cmdStr)
				}
			}

			// Verify first argument is the runtime
			if result[0] != tt.runtime {
				t.Errorf("Expected first arg to be %q, got %q", tt.runtime, result[0])
			}

			// Verify second argument is "run"
			if result[1] != "run" {
				t.Errorf("Expected second arg to be 'run', got %q", result[1])
			}
		})
	}
}

// TestBuildRunCommand_TTYBehavior documents the TTY detection behavior
// Note: In CI environments (non-TTY), -i and -t flags should NOT be present
// In interactive terminals (TTY), -i and -t flags should be present
func TestBuildRunCommand_TTYBehavior(t *testing.T) {
	opts := &RunOptions{
		ImageRef: "demo-app:v1",
		Args:     []string{"echo", "test"},
	}

	result := buildRunCommand("docker", opts)
	cmdStr := strings.Join(result, " ")

	// Document the behavior:
	// - In CI (no TTY): should NOT have -i or -t
	// - In terminal (TTY): should have -i and/or -t
	// We can't easily test the actual TTY detection here, but we document it
	t.Logf("Command generated: %s", cmdStr)
	t.Logf("Note: -i flag is added only when stdin is a TTY")
	t.Logf("Note: -t flag is added only when stdout is a TTY")
	t.Logf("This ensures 'acc run' works in both CI and interactive terminals")
}

func TestDetectRuntime(t *testing.T) {
	// This test will vary based on what's installed
	// Just verify it returns something valid or an error
	runtime, err := detectRuntime()
	if err != nil {
		// No runtime found - that's okay for some test environments
		t.Logf("No container runtime found (this is okay in some environments): %v", err)
		return
	}

	validRuntimes := []string{"docker", "podman", "nerdctl"}
	isValid := false
	for _, vr := range validRuntimes {
		if runtime == vr {
			isValid = true
			break
		}
	}

	if !isValid {
		t.Errorf("detectRuntime returned unexpected runtime: %q", runtime)
	}

	t.Logf("Detected container runtime: %s", runtime)
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
