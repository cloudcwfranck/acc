package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudcwfranck/acc/internal/policy"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// Init initializes a new acc project (AGENTS.md Section 2 - acc init)
// Creates:
// - acc.yaml
// - .acc/ directory
// - .acc/policy/default.rego
func Init(projectName string, outputJSON bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if acc.yaml already exists
	accYamlPath := filepath.Join(cwd, "acc.yaml")
	if _, err := os.Stat(accYamlPath); err == nil {
		return fmt.Errorf("acc.yaml already exists in current directory")
	}

	// If no project name provided, use directory name
	if projectName == "" {
		projectName = filepath.Base(cwd)
	}

	// Create default config
	cfg := DefaultConfig(projectName)

	// Write acc.yaml
	if err := os.WriteFile(accYamlPath, []byte(cfg.ToYAML()), 0644); err != nil {
		return fmt.Errorf("failed to write acc.yaml: %w", err)
	}

	if !outputJSON {
		ui.PrintSuccess(fmt.Sprintf("Created acc.yaml for project '%s'", projectName))
	}

	// Create .acc directory structure
	accDir := filepath.Join(cwd, ".acc")
	dirs := []string{
		accDir,
		filepath.Join(accDir, "policy"),
		filepath.Join(accDir, "profiles"), // v0.2.1: Add profiles directory
		filepath.Join(accDir, "locks"),
		filepath.Join(accDir, "cache"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if !outputJSON {
		ui.PrintSuccess("Created .acc/ directory structure")
	}

	// Write default policy
	policyPath := filepath.Join(accDir, "policy", "default.rego")
	if err := os.WriteFile(policyPath, []byte(policy.DefaultPolicyContent), 0644); err != nil {
		return fmt.Errorf("failed to write default policy: %w", err)
	}

	if !outputJSON {
		ui.PrintSuccess("Created default policy at .acc/policy/default.rego")
		ui.PrintInfo("\nNext steps:")
		fmt.Println("  1. Review and customize acc.yaml")
		fmt.Println("  2. Customize policies in .acc/policy/")
		fmt.Println("  3. Run 'acc build' to build your first workload")
	}

	if outputJSON {
		fmt.Printf(`{"status":"success","project":"%s","files":["acc.yaml",".acc/policy/default.rego"]}%s`, projectName, "\n")
	}

	return nil
}
