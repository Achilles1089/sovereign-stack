package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/server"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the web dashboard",
	Long: `Start the Sovereign Stack web dashboard.

The dashboard provides a visual interface to manage your
services, apps, AI models, and backups.`,
	RunE: runDashboard,
}

var dashboardPort string

func init() {
	dashboardCmd.Flags().StringVarP(&dashboardPort, "port", "p", "8080", "Port to serve dashboard on")
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Dashboard")
	fmt.Println("  ──────────────────────────────")
	fmt.Println()

	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)

	addr := "localhost:" + dashboardPort
	srv := server.New(cfg, addr)

	// Check if built dashboard exists
	staticDir := config.ConfigDir() + "/dashboard"
	srv.SetStaticDir(staticDir)

	fmt.Printf("  🌐 Dashboard: http://%s\n", addr)
	fmt.Printf("  📡 API:       http://%s/api/\n", addr)
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println()

	return srv.Start()
}
