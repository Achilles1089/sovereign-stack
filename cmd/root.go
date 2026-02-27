package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

var (
	verbose    bool
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "sovereign",
	Short: "Sovereign Stack — Own your cloud. Own your AI.",
	Long: `
   ███████╗ ██████╗ ██╗   ██╗███████╗██████╗ ███████╗██╗ ██████╗ ███╗   ██╗
   ██╔════╝██╔═══██╗██║   ██║██╔════╝██╔══██╗██╔════╝██║██╔════╝ ████╗  ██║
   ███████╗██║   ██║██║   ██║█████╗  ██████╔╝█████╗  ██║██║  ███╗██╔██╗ ██║
   ╚════██║██║   ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗██╔══╝  ██║██║   ██║██║╚██╗██║
   ███████║╚██████╔╝ ╚████╔╝ ███████╗██║  ██║███████╗██║╚██████╔╝██║ ╚████║
   ╚══════╝ ╚═════╝   ╚═══╝  ╚══════╝╚═╝  ╚═╝╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═══╝
                            ███████╗████████╗ █████╗  ██████╗██╗  ██╗
                            ██╔════╝╚══██╔══╝██╔══██╗██╔════╝██║ ██╔╝
                            ███████╗   ██║   ███████║██║     █████╔╝
                            ╚════██║   ██║   ██╔══██║██║     ██╔═██╗
                            ███████║   ██║   ██║  ██║╚██████╗██║  ██╗
                            ╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝

  One command to own your cloud. One command to own your AI.
  https://github.com/Achilles1089/sovereign-stack

  Run 'sovereign init' to set up your personal sovereign server.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Config file path (default: ~/.sovereign/config.yaml)")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version of Sovereign Stack",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Sovereign Stack %s\n", Version)
		},
	})
}

// GetVerbose returns the verbose flag value for use by subcommands
func GetVerbose() bool {
	return verbose
}

// GetConfigPath returns the config path for use by subcommands
func GetConfigPath() string {
	return configPath
}
