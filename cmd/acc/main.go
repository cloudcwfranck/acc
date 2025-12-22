package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudcwfranck/acc/internal/attest"
	"github.com/cloudcwfranck/acc/internal/build"
	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/inspect"
	"github.com/cloudcwfranck/acc/internal/policy"
	"github.com/cloudcwfranck/acc/internal/profile"
	"github.com/cloudcwfranck/acc/internal/promote"
	"github.com/cloudcwfranck/acc/internal/push"
	"github.com/cloudcwfranck/acc/internal/runtime"
	"github.com/cloudcwfranck/acc/internal/trust"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/upgrade"
	"github.com/cloudcwfranck/acc/internal/verify"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global flags
var (
	colorFlag   string
	jsonFlag    bool
	quietFlag   bool
	noEmojiFlag bool
	policyPack  string
	configFile  string
)

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError(err.Error()))
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "acc",
		Short: "Secure Workload Accelerator",
		Long: `acc turns source code or OCI references into verified, policy-compliant OCI workloads
that can be built, verified, run, pushed, and promoted with cryptographic and policy gates.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Apply global UI settings
			ui.SetColorMode(colorFlag)
			ui.SetEmojiEnabled(!noEmojiFlag)
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&colorFlag, "color", "auto", "colorize output (auto|always|never)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "suppress non-critical output")
	rootCmd.PersistentFlags().BoolVar(&noEmojiFlag, "no-emoji", false, "disable emoji in output")
	rootCmd.PersistentFlags().StringVar(&policyPack, "policy-pack", "", "path to policy pack")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "path to config file")

	// Add all subcommands
	rootCmd.AddCommand(
		NewInitCmd(),
		NewBuildCmd(),
		NewVerifyCmd(),
		NewRunCmd(),
		NewPushCmd(),
		NewPromoteCmd(),
		NewPolicyCmd(),
		NewAttestCmd(),
		NewInspectCmd(),
		NewTrustCmd(),
		NewConfigCmd(),
		NewLoginCmd(),
		NewVersionCmd(),
		NewUpgradeCmd(),
	)

	return rootCmd
}

func NewInitCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new acc project",
		Long:  "Bootstrap project configuration, generate acc.yaml, and create .acc/ directory with starter policy",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := projectName
			if len(args) > 0 {
				name = args[0]
			}
			return config.Init(name, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&projectName, "name", "n", "", "project name (defaults to directory name)")

	return cmd
}

func NewBuildCmd() *cobra.Command {
	var tag string

	cmd := &cobra.Command{
		Use:   "build [image]",
		Short: "Build OCI image with SBOM generation",
		Long:  "Build OCI image (local or referenced), generate SBOM, and output digest + artifact refs",
		Example: `  # Build with tag flag
  acc build --tag demo-app:ok
  acc build -t demo-app:ok

  # Build with positional argument (backward compatible)
  acc build demo-app:ok`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			// v0.2.3: Accept positional argument for backward compatibility
			finalTag := tag
			if len(args) > 0 {
				if tag != "" {
					// Both positional and --tag provided, --tag takes precedence
					ui.PrintWarning(fmt.Sprintf("Both positional argument %q and --tag %q provided; using --tag", args[0], tag))
				} else {
					finalTag = args[0]
				}
			}

			// Build image
			result, err := build.Build(cfg, finalTag, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "", "image tag (default: from config)")

	return cmd
}

func NewVerifyCmd() *cobra.Command {
	var (
		imageRef    string
		profilePath string
	)

	cmd := &cobra.Command{
		Use:   "verify [image]",
		Short: "Verify SBOM, policy compliance, and attestations",
		Long:  "Verify SBOM exists, evaluate policy, and check signature/attestation presence",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			// v0.2.0: Load profile if specified
			var prof *profile.Profile
			if profilePath != "" {
				prof, err = profile.Load(profilePath)
				if err != nil {
					return fmt.Errorf("failed to load profile: %w", err)
				}
			}

			// Verify
			result, err := verify.Verify(cfg, ref, false, jsonFlag, prof)

			// v0.1.4: Defensive nil check (should never happen after v0.1.4 fixes)
			if result == nil {
				if jsonFlag {
					fmt.Println(`{"status":"fail","error":"internal error: nil result"}`)
				} else {
					fmt.Fprintln(os.Stderr, "Error: verification failed with internal error")
					if err != nil {
						fmt.Fprintln(os.Stderr, err.Error())
					}
				}
				os.Exit(2)
			}

			if err != nil {
				if jsonFlag {
					fmt.Println(result.FormatJSON())
				}
				os.Exit(result.ExitCode())
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			os.Exit(result.ExitCode())
			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to verify")
	cmd.Flags().StringVar(&profilePath, "profile", "", "policy profile name or path (.acc/profiles/<name>.yaml or explicit path)")

	return cmd
}

func NewRunCmd() *cobra.Command {
	var (
		imageRef    string
		user        string
		networkMode string
		readOnly    bool
		caps        []string
	)

	cmd := &cobra.Command{
		Use:   "run [image] [-- command args...]",
		Short: "Verify and run workload locally",
		Long:  "Verify first, then run locally with least privilege defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			// Parse image ref and command args
			ref := imageRef
			cmdArgs := []string{}

			if len(args) > 0 {
				ref = args[0]
				if len(args) > 1 {
					cmdArgs = args[1:]
				}
			}

			if ref == "" {
				return fmt.Errorf("image reference required")
			}

			opts := &runtime.RunOptions{
				ImageRef:     ref,
				Args:         cmdArgs,
				NetworkMode:  networkMode,
				ReadOnly:     readOnly,
				User:         user,
				Capabilities: caps,
			}

			return runtime.Run(cfg, opts, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to run")
	cmd.Flags().StringVarP(&user, "user", "u", "", "run as specified user (default: image default)")
	cmd.Flags().StringVar(&networkMode, "network", "none", "network mode (none|bridge|host)")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "mount root filesystem as read-only")
	cmd.Flags().StringSliceVar(&caps, "cap-add", []string{}, "add Linux capabilities")

	return cmd
}

func NewPushCmd() *cobra.Command {
	var imageRef string

	cmd := &cobra.Command{
		Use:   "push [image]",
		Short: "Verify and push verified artifacts",
		Long:  "Push only verified artifacts - verification gates execution",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			if ref == "" {
				return fmt.Errorf("image reference required\n\nUsage: acc push <image>")
			}

			// Push (with verification gate)
			result, err := push.Push(cfg, ref, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to push")

	return cmd
}

func NewPromoteCmd() *cobra.Command {
	var (
		imageRef  string
		targetEnv string
	)

	cmd := &cobra.Command{
		Use:   "promote [image] --to <env>",
		Short: "Re-verify and promote workload",
		Long:  "Re-verify, apply environment-specific policy, and retag without rebuild",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			if ref == "" {
				return fmt.Errorf("image reference required\n\nUsage: acc promote <image> --to <env>")
			}

			// Promote
			result, err := promote.Promote(cfg, ref, targetEnv, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to promote")
	cmd.Flags().StringVar(&targetEnv, "to", "", "target environment (required)")
	cmd.MarkFlagRequired("to")

	return cmd
}

func NewPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage and test policies",
		Long:  "List policies, test policies, and explain last decision",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add explain subcommand
	explainCmd := &cobra.Command{
		Use:   "explain [last]",
		Short: "Explain last verification decision",
		Long:  "Display developer-friendly explanation of the last verification decision",
		RunE: func(cmd *cobra.Command, args []string) error {
			return policy.Explain(jsonFlag)
		},
	}

	cmd.AddCommand(explainCmd)
	return cmd
}

func NewAttestCmd() *cobra.Command {
	var imageRef string

	cmd := &cobra.Command{
		Use:   "attest [image]",
		Short: "Create attestation for artifact",
		Long:  "Create minimal attestation with build metadata and policy hash",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			if ref == "" {
				return fmt.Errorf("image reference required\n\nUsage: acc attest <image>")
			}

			// Create attestation
			result, err := attest.Attest(cfg, ref, version, commit, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to attest")

	return cmd
}

func NewInspectCmd() *cobra.Command {
	var imageRef string

	cmd := &cobra.Command{
		Use:   "inspect [image]",
		Short: "Inspect artifact trust summary",
		Long:  "Display human-readable trust summary for an artifact",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			if ref == "" {
				return fmt.Errorf("image reference required\n\nUsage: acc inspect <image>")
			}

			// Inspect
			result, err := inspect.Inspect(cfg, ref, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to inspect")

	return cmd
}

func NewTrustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Manage trust status and profiles",
		Long:  "View trust status and manage policy profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add status subcommand
	cmd.AddCommand(NewTrustStatusCmd())
	return cmd
}

func NewTrustStatusCmd() *cobra.Command {
	var imageRef string

	cmd := &cobra.Command{
		Use:   "status [image]",
		Short: "View trust status for an image",
		Long:  "Display verification status, profile used, violations, and attestations for an image",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := imageRef
			if len(args) > 0 {
				ref = args[0]
			}

			if ref == "" {
				return fmt.Errorf("image reference required\n\nUsage: acc trust status <image>")
			}

			// Load trust status
			result, err := trust.Status(ref, jsonFlag)
			if err != nil {
				return err
			}

			if jsonFlag {
				fmt.Println(result.FormatJSON())
			}

			os.Exit(result.ExitCode())
			return nil
		},
	}

	cmd.Flags().StringVarP(&imageRef, "image", "i", "", "image reference to check")

	return cmd
}

func NewConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Get or set configuration values",
		Long:  "Get or set configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate to registries",
		Long:  "Authenticate to registries and identity providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version, commit, and build info",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonFlag {
				fmt.Printf(`{"version":"%s","commit":"%s","date":"%s"}%s`, version, commit, date, "\n")
			} else {
				fmt.Printf("acc version %s\n", version)
				fmt.Printf("commit: %s\n", commit)
				fmt.Printf("built: %s\n", date)
			}
		},
	}
}

func NewUpgradeCmd() *cobra.Command {
	var (
		targetVersion    string
		dryRun           bool
		verifySignature  bool
		cosignKey        string
		verifyProvenance bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade acc to the latest version",
		Long:  "Download and install the latest stable release of acc from GitHub with checksum verification",
		Example: `  # Upgrade to latest version
  acc upgrade

  # Upgrade to specific version
  acc upgrade --version v0.1.6

  # Show what would happen without installing
  acc upgrade --dry-run

  # Upgrade with cosign signature verification
  acc upgrade --verify-signature --cosign-key <path-to-key>

  # Upgrade with SLSA provenance verification
  acc upgrade --verify-provenance

  # Upgrade with both verifications (enterprise mode)
  acc upgrade --verify-signature --verify-provenance`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get upgrade package
			opts := &upgrade.UpgradeOptions{
				Version:          targetVersion,
				DryRun:           dryRun,
				CurrentVersion:   version,
				VerifySignature:  verifySignature,
				CosignKey:        cosignKey,
				VerifyProvenance: verifyProvenance,
				// Read env vars for testing overrides
				APIBase:        os.Getenv("ACC_UPGRADE_API_BASE"),
				DownloadBase:   os.Getenv("ACC_UPGRADE_DOWNLOAD_BASE"),
				DisableInstall: os.Getenv("ACC_UPGRADE_DISABLE_INSTALL") == "1",
			}

			result, err := upgrade.Upgrade(opts)
			if err != nil {
				if jsonFlag {
					fmt.Printf(`{"error":"%s"}%s`, err.Error(), "\n")
				} else {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
				os.Exit(1)
			}

			if jsonFlag {
				data, _ := json.Marshal(result)
				fmt.Println(string(data))
			} else {
				if !result.Updated {
					fmt.Println(result.Message)
				} else {
					fmt.Printf("Current version: %s\n", result.CurrentVersion)
					fmt.Printf("Target version:  %s\n", result.TargetVersion)
					fmt.Printf("Asset:           %s\n", result.AssetName)
					if result.Checksum != "" {
						fmt.Printf("Checksum:        %s\n", result.Checksum[:16]+"...")
					}
					if result.SignatureVerified {
						fmt.Printf("Signature:       ✓ Verified\n")
					}
					if result.ProvenanceVerified {
						fmt.Printf("Provenance:      ✓ Verified\n")
					}
					if result.InstallPath != "" {
						fmt.Printf("Installed to:    %s\n", result.InstallPath)
					}
					fmt.Printf("\n%s\n", result.Message)
				}
			}
		},
	}

	cmd.Flags().StringVar(&targetVersion, "version", "", "target version to install (default: latest)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would happen without downloading/installing")

	// Supply-chain verification flags (opt-in)
	cmd.Flags().BoolVar(&verifySignature, "verify-signature", false, "verify cosign signature (requires cosign in PATH)")
	cmd.Flags().StringVar(&cosignKey, "cosign-key", "", "path/URL to cosign public key (optional, uses keyless if not provided)")
	cmd.Flags().BoolVar(&verifyProvenance, "verify-provenance", false, "verify SLSA provenance")

	return cmd
}
