package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile represents a policy profile configuration (v1 schema)
type Profile struct {
	SchemaVersion int                `yaml:"schemaVersion"`
	Name          string             `yaml:"name"`
	Description   string             `yaml:"description"`
	Policies      PolicyConfig       `yaml:"policies,omitempty"`
	Violations    ViolationConfig    `yaml:"violations,omitempty"`
	Warnings      WarningConfig      `yaml:"warnings,omitempty"`
}

// PolicyConfig defines which policies are allowed
type PolicyConfig struct {
	Allow []string `yaml:"allow,omitempty"`
}

// ViolationConfig defines which violations to ignore
type ViolationConfig struct {
	Ignore []string `yaml:"ignore,omitempty"`
}

// WarningConfig defines warning display behavior
type WarningConfig struct {
	Show bool `yaml:"show"`
}

// Load loads a profile from a name or path
// - If path contains "/" or ends with .yaml/.yml, treat as explicit path
// - Otherwise, look in .acc/profiles/<name>.yaml
func Load(nameOrPath string) (*Profile, error) {
	var profilePath string

	// Determine if this is a path or a name
	if strings.Contains(nameOrPath, "/") || strings.HasSuffix(nameOrPath, ".yaml") || strings.HasSuffix(nameOrPath, ".yml") {
		// Explicit path
		profilePath = nameOrPath
	} else {
		// Profile name - look in .acc/profiles/
		profilePath = filepath.Join(".acc", "profiles", nameOrPath+".yaml")
	}

	// Read file
	data, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile not found: %s", profilePath)
		}
		return nil, fmt.Errorf("failed to read profile %s: %w", profilePath, err)
	}

	// Parse YAML with strict mode to reject unknown fields
	var profile Profile
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Reject unknown fields

	if err := decoder.Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile %s: %w", profilePath, err)
	}

	// Validate
	if err := Validate(&profile); err != nil {
		return nil, fmt.Errorf("profile %s validation failed: %w", profilePath, err)
	}

	return &profile, nil
}

// Validate validates a profile structure and values
func Validate(p *Profile) error {
	// Schema version must be 1
	if p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion: %d (expected 1)", p.SchemaVersion)
	}

	// Name is required
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Description is required
	if p.Description == "" {
		return fmt.Errorf("description is required")
	}

	// Validate policies.allow contains valid rule names (non-empty strings)
	for i, rule := range p.Policies.Allow {
		if strings.TrimSpace(rule) == "" {
			return fmt.Errorf("policies.allow[%d]: empty rule name not allowed", i)
		}
	}

	// Validate violations.ignore contains valid severity/rule names
	for i, item := range p.Violations.Ignore {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("violations.ignore[%d]: empty value not allowed", i)
		}
	}

	return nil
}
