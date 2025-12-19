package main

import (
	"fmt"
	"os"

	"github.com/cloudcwfranck/acc/internal/build"
	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/runtime"
	"github.com/cloudcwfranck/acc/internal/ui"
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
		NewConfigCmd(),
		NewLoginCmd(),
		NewVersionCmd(),
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
		Use:   "build",
		Short: "Build OCI image with SBOM generation",
		Long:  "Build OCI image (local or referenced), generate SBOM, and output digest + artifact refs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w\n\nHint: Run 'acc init' to create a configuration file", err)
			}

			// Build image
			result, err := build.Build(cfg, tag, jsonFlag)
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
	var imageRef string

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

			// Verify
			result, err := verify.Verify(cfg, ref, false, jsonFlag)
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
	return &cobra.Command{
		Use:   "push",
		Short: "Verify and push verified artifacts",
		Long:  "Verify first, then push only verified artifacts with attestations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewPromoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "promote",
		Short: "Re-verify and promote workload",
		Long:  "Re-verify, apply environment-specific policy, and retag without rebuild",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewPolicyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "policy",
		Short: "Manage and test policies",
		Long:  "List policies, test policies, and explain last decision",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewAttestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attest",
		Short: "Attach attestations to artifacts",
		Long:  "Attach attestations (SLSA, build metadata, env approval)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}

func NewInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect artifact trust summary",
		Long:  "Display human-readable trust summary for an artifact",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
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
