package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/apps"
	"github.com/Achilles1089/sovereign-stack/internal/config"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage sovereign apps",
	Long:  `Browse, install, update, and remove apps from the Sovereign Stack marketplace.`,
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available apps",
	RunE:  runAppList,
}

var appInstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install an app",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppInstall,
}

var appRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed app",
	Args:  cobra.ExactArgs(1),
	RunE:  runAppRemove,
}

func init() {
	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appInstallCmd)
	appCmd.AddCommand(appRemoveCmd)
	rootCmd.AddCommand(appCmd)
}

func runAppList(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — App Marketplace")
	fmt.Println("  ─────────────────────────────────────")
	fmt.Println()

	// Check for installed apps
	installed := make(map[string]bool)
	if installedList, err := apps.InstalledApps(); err == nil {
		for _, name := range installedList {
			installed[name] = true
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tDESCRIPTION\tCATEGORY\tSTATUS")
	fmt.Fprintln(w, "  ────\t───────────\t────────\t──────")

	for _, app := range apps.BuiltinApps {
		status := "available"
		if installed[app.Name] {
			status = "✓ installed"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", app.Name, app.Description, app.Category, status)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("  %d apps available. Install with: sovereign app install <name>\n", len(apps.BuiltinApps))
	fmt.Println()
	return nil
}

func runAppInstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	app := apps.FindApp(name)
	if app == nil {
		return fmt.Errorf("app '%s' not found. Run 'sovereign app list' to see available apps", name)
	}

	// Check if initialized
	cfgPath := config.ConfigPath(GetConfigPath())
	_, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("sovereign not initialized. Run 'sovereign init' first")
	}

	fmt.Printf("\n  Installing %s v%s...\n", app.DisplayName, app.Version)
	fmt.Println("  → Pulling Docker image...")
	fmt.Println("  → Generating configuration...")

	if err := apps.InstallApp(app); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Println("  → Starting container...")
	fmt.Printf("  ✓ %s installed successfully!\n", app.DisplayName)

	if app.CaddyRoute != nil {
		fmt.Printf("  → Access at: http://localhost:%d\n", app.CaddyRoute.Port)
	}

	fmt.Println()
	return nil
}

func runAppRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	app := apps.FindApp(name)
	if app == nil {
		return fmt.Errorf("app '%s' not found", name)
	}

	fmt.Printf("\n  Removing %s...\n", app.DisplayName)

	if err := apps.RemoveApp(name); err != nil {
		return fmt.Errorf("removal failed: %w", err)
	}

	fmt.Printf("  ✓ %s removed.\n\n", app.DisplayName)
	return nil
}
