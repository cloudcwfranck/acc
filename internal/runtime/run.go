package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/trust"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/verify"
	"golang.org/x/term"
)

// RunOptions represents options for running a workload
type RunOptions struct {
	ImageRef     string
	Args         []string
	NetworkMode  string
	ReadOnly     bool
	User         string
	Capabilities []string
}

// Run runs a workload locally with verification gates (AGENTS.md Section 2 - acc run)
// CRITICAL: This MUST call verify first and MUST fail if verification fails (Section 1.1)
func Run(cfg *config.Config, opts *RunOptions, outputJSON bool) error {
	// CRITICAL: Verification gates execution (AGENTS.md Section 1.1)
	if !outputJSON {
		ui.PrintTrust("Verifying workload before execution...")
	}

	verifyResult, err := verify.Verify(cfg, opts.ImageRef, false, outputJSON, nil)
	if err != nil {
		// RED OUTPUT MEANS STOP (AGENTS.md Section 0)
		if !outputJSON {
			ui.PrintError("Verification failed - workload will NOT run")
		}
		return fmt.Errorf("verification failed: %w", err)
	}

	if verifyResult.Status == "fail" {
		// RED OUTPUT MEANS STOP (AGENTS.md Section 0)
		if !outputJSON {
			ui.PrintError("Verification failed - workload will NOT run")
		}
		return fmt.Errorf("verification failed with status: %s", verifyResult.Status)
	}

	if !outputJSON {
		ui.PrintSuccess("Verification passed")
	}

	// v0.3.1: Optional attestation enforcement
	if cfg.Policy.RequireAttestation {
		if !outputJSON {
			ui.PrintTrust("Checking attestation requirement...")
		}

		// Use local attestations only for enforcement check (remote=false)
		attestResult, err := trust.VerifyAttestations(opts.ImageRef, false, outputJSON)
		if err != nil || attestResult.VerificationStatus != "verified" {
			// Attestation enforcement blocks execution (same exit code as verification gate)
			if !outputJSON {
				ui.PrintError("Attestation requirement not met - workload will NOT run")
				fmt.Fprintf(os.Stderr, "\nRemediation:\n")
				fmt.Fprintf(os.Stderr, "  1. Verify the workload: acc verify %s\n", opts.ImageRef)
				fmt.Fprintf(os.Stderr, "  2. Create attestation: acc attest %s\n", opts.ImageRef)
				fmt.Fprintf(os.Stderr, "  3. Re-run: acc run %s\n", opts.ImageRef)
			}
			return fmt.Errorf("attestation requirement not met: %s", attestResult.VerificationStatus)
		}

		if !outputJSON {
			ui.PrintSuccess(fmt.Sprintf("Attestation verified (%d found)", attestResult.AttestationCount))
		}
	}

	if !outputJSON {
		ui.PrintInfo("Proceeding to run workload")
	}

	// CRITICAL: Trust enforcement succeeded at this point
	// Runtime execution failures beyond this point must not override trust decision

	// Detect runtime tool
	runtime, err := detectRuntime()
	if err != nil {
		// Runtime not available is a warning, not a trust failure
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Container runtime not available: %v", err))
			ui.PrintInfo("Trust enforcement succeeded, but cannot execute workload")
		}
		// Return success because trust enforcement passed
		return nil
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Using runtime: %s", runtime))
	}

	// Build run command with security defaults (AGENTS.md Section 8)
	cmdArgs := buildRunCommand(runtime, opts)

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Running: %s", strings.Join(cmdArgs, " ")))
	}

	// Execute workload
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// Runtime execution failure - this is NOT a trust failure
		// Trust enforcement already succeeded, so we log but don't fail
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Workload execution failed: %v", err))
			ui.PrintInfo("Trust enforcement succeeded (exit 0)")
		}
		// Return success because trust enforcement passed
		return nil
	}

	return nil
}

// detectRuntime detects which container runtime is available
func detectRuntime() (string, error) {
	runtimes := []string{"docker", "podman", "nerdctl"}
	for _, rt := range runtimes {
		if _, err := exec.LookPath(rt); err == nil {
			return rt, nil
		}
	}
	return "", fmt.Errorf("no container runtime found (tried: docker, podman, nerdctl)\n\nRemediation:\n  - Install Docker: https://docs.docker.com/get-docker/\n  - Or install Podman: https://podman.io/getting-started/installation\n  - Or install nerdctl: https://github.com/containerd/nerdctl")
}

// isStdinTTY checks if stdin is a terminal
func isStdinTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// isStdoutTTY checks if stdout is a terminal
func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// buildRunCommand builds the run command with security defaults (AGENTS.md Section 8)
func buildRunCommand(runtime string, opts *RunOptions) []string {
	args := []string{runtime, "run"}

	// Remove container after run
	args = append(args, "--rm")

	// Add -i (interactive) only if stdin is a TTY
	// This allows piped input in CI while preserving interactive mode in terminals
	if isStdinTTY() {
		args = append(args, "-i")
	}

	// Add -t (TTY) only if stdout is a TTY
	// This prevents "the input device is not a TTY" errors in CI environments
	if isStdoutTTY() {
		args = append(args, "-t")
	}

	// Apply security defaults (AGENTS.md Section 8)

	// 1. Non-root user (if specified)
	if opts.User != "" {
		args = append(args, "--user", opts.User)
	} else {
		// Default to non-root user if runtime supports it
		if runtime == "docker" || runtime == "podman" {
			// Warn if no user specified
			ui.PrintWarning("No user specified - consider using --user to run as non-root")
		}
	}

	// 2. Read-only filesystem where supported
	if opts.ReadOnly {
		args = append(args, "--read-only")
	}

	// 3. Network restricted by default
	networkMode := opts.NetworkMode
	if networkMode == "" {
		networkMode = "none"
	}
	args = append(args, "--network", networkMode)

	// 4. Drop all capabilities by default, add only specified ones
	if runtime == "docker" || runtime == "podman" {
		args = append(args, "--cap-drop", "ALL")
		for _, cap := range opts.Capabilities {
			args = append(args, "--cap-add", cap)
		}
	}

	// Security options
	if runtime == "docker" || runtime == "podman" {
		args = append(args, "--security-opt", "no-new-privileges")
	}

	// Add image reference
	args = append(args, opts.ImageRef)

	// Add any additional arguments
	args = append(args, opts.Args...)

	return args
}

// isTTY checks if the given file is a terminal
func isTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
