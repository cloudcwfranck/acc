package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the acc configuration (AGENTS.md Section 5.3)
type Config struct {
	Project      ProjectConfig        `mapstructure:"project"`
	Build        BuildConfig          `mapstructure:"build"`
	Registry     RegistryConfig       `mapstructure:"registry"`
	Policy       PolicyConfig         `mapstructure:"policy"`
	Trust        TrustConfig          `mapstructure:"trust"` // v0.3.3: trust requirements
	Signing      SigningConfig        `mapstructure:"signing"`
	SBOM         SBOMConfig           `mapstructure:"sbom"`
	Environments map[string]EnvConfig `mapstructure:"environments"`
}

// EnvConfig represents environment-specific configuration
type EnvConfig struct {
	Policy   *PolicyConfig   `mapstructure:"policy"`
	Registry *RegistryConfig `mapstructure:"registry"`
}

type ProjectConfig struct {
	Name string `mapstructure:"name"`
}

type BuildConfig struct {
	Context    string `mapstructure:"context"`
	DefaultTag string `mapstructure:"defaultTag"`
}

type RegistryConfig struct {
	Default string `mapstructure:"default"`
}

type PolicyConfig struct {
	Mode               string `mapstructure:"mode"`               // enforce|warn
	RequireAttestation bool   `mapstructure:"requireAttestation"` // v0.3.1: require verified attestations for run/push
}

type SigningConfig struct {
	Mode string `mapstructure:"mode"` // keyless|key
}

type SBOMConfig struct {
	Format string `mapstructure:"format"` // spdx|cyclonedx
}

// TrustConfig represents trust/attestation requirements (v0.3.3)
type TrustConfig struct {
	RequireAttestations *AttestationRequirements `mapstructure:"requireAttestations"`
}

// AttestationRequirements defines thresholds for attestation validity
type AttestationRequirements struct {
	Enabled                 bool     `mapstructure:"enabled"`                 // default: false
	MinCount                int      `mapstructure:"minCount"`                // minimum valid attestations required
	Sources                 []string `mapstructure:"sources"`                 // allowed: ["local", "remote"]
	RequireDigestMatch      bool     `mapstructure:"requireDigestMatch"`      // default: true
	RequireValidSchema      bool     `mapstructure:"requireValidSchema"`      // default: true
	RequireResultsHashMatch bool     `mapstructure:"requireResultsHashMatch"` // default: true (v0.3.3)
	Mode                    string   `mapstructure:"mode"`                    // enforce|warn (default: enforce)
}

// Load loads configuration following discovery order (AGENTS.md Section 5.1)
// 1. --config <path>
// 2. ./acc.yaml
// 3. ./.acc/acc.yaml
// 4. $HOME/.acc/config.yaml
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if configPath != "" {
		// Use explicitly provided config path
		v.SetConfigFile(configPath)
	} else {
		// Search in order
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try ./acc.yaml
		if _, err := os.Stat(filepath.Join(cwd, "acc.yaml")); err == nil {
			v.SetConfigFile(filepath.Join(cwd, "acc.yaml"))
		} else if _, err := os.Stat(filepath.Join(cwd, ".acc", "acc.yaml")); err == nil {
			// Try ./.acc/acc.yaml
			v.SetConfigFile(filepath.Join(cwd, ".acc", "acc.yaml"))
		} else {
			// Try $HOME/.acc/config.yaml
			homeDir, err := os.UserHomeDir()
			if err == nil {
				homeConfig := filepath.Join(homeDir, ".acc", "config.yaml")
				if _, err := os.Stat(homeConfig); err == nil {
					v.SetConfigFile(homeConfig)
				}
			}
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration (AGENTS.md Section 5.3)
func (c *Config) Validate() error {
	if c.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}
	if c.Build.Context == "" {
		return fmt.Errorf("build.context is required")
	}
	if c.Build.DefaultTag == "" {
		return fmt.Errorf("build.defaultTag is required")
	}
	if c.Registry.Default == "" {
		return fmt.Errorf("registry.default is required")
	}
	if c.Policy.Mode != "enforce" && c.Policy.Mode != "warn" {
		return fmt.Errorf("policy.mode must be 'enforce' or 'warn'")
	}
	if c.Signing.Mode != "keyless" && c.Signing.Mode != "key" {
		return fmt.Errorf("signing.mode must be 'keyless' or 'key'")
	}
	if c.SBOM.Format != "spdx" && c.SBOM.Format != "cyclonedx" {
		return fmt.Errorf("sbom.format must be 'spdx' or 'cyclonedx'")
	}
	return nil
}

// GetPolicyForEnv returns the policy config for a specific environment
// If environment-specific policy is defined, it overrides the default
func (c *Config) GetPolicyForEnv(env string) PolicyConfig {
	if env != "" && c.Environments != nil {
		if envCfg, ok := c.Environments[env]; ok {
			if envCfg.Policy != nil {
				return *envCfg.Policy
			}
		}
	}
	return c.Policy
}

// GetRegistryForEnv returns the registry config for a specific environment
func (c *Config) GetRegistryForEnv(env string) RegistryConfig {
	if env != "" && c.Environments != nil {
		if envCfg, ok := c.Environments[env]; ok {
			if envCfg.Registry != nil {
				return *envCfg.Registry
			}
		}
	}
	return c.Registry
}

// DefaultConfig returns a default configuration template
func DefaultConfig(projectName string) *Config {
	return &Config{
		Project: ProjectConfig{
			Name: projectName,
		},
		Build: BuildConfig{
			Context:    ".",
			DefaultTag: "latest",
		},
		Registry: RegistryConfig{
			Default: "localhost:5000",
		},
		Policy: PolicyConfig{
			Mode: "enforce",
		},
		Signing: SigningConfig{
			Mode: "keyless",
		},
		SBOM: SBOMConfig{
			Format: "spdx",
		},
	}
}

// ToYAML converts config to YAML string
func (c *Config) ToYAML() string {
	return fmt.Sprintf(`# acc configuration file
project:
  name: %s

build:
  context: %s
  defaultTag: %s

registry:
  default: %s

policy:
  mode: %s
  # requireAttestation: false  # v0.3.1: require verified attestations for run/push

signing:
  mode: %s

sbom:
  format: %s
`, c.Project.Name, c.Build.Context, c.Build.DefaultTag,
		c.Registry.Default, c.Policy.Mode, c.Signing.Mode, c.SBOM.Format)
}
