package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs <service>",
	Short: "Stream logs from a service",
	Long: `Stream Docker logs from a sovereign service.

Example:
  sovereign logs postgres
  sovereign logs caddy -f`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

var logsFollow bool

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart a service or all services",
	Long: `Restart one or all sovereign services.

Example:
  sovereign restart         — restart all services
  sovereign restart postgres — restart just postgres`,
	RunE: runRestart,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update all services to latest versions",
	Long: `Pull the latest Docker images for all sovereign services
and recreate containers with the updated images.`,
	RunE: runUpdate,
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(updateCmd)
}

func composeFile() string {
	return config.ConfigDir() + "/docker-compose.yml"
}

func runLogs(cmd *cobra.Command, args []string) error {
	service := args[0]

	composeArgs := []string{"compose", "-f", composeFile(), "logs", service}
	if logsFollow {
		composeArgs = append(composeArgs, "-f")
	} else {
		composeArgs = append(composeArgs, "--tail", "100")
	}

	proc := exec.Command("docker", composeArgs...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()
}

func runRestart(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Restart")
	fmt.Println("  ────────────────────────────")
	fmt.Println()

	composeArgs := []string{"compose", "-f", composeFile(), "restart"}
	if len(args) > 0 {
		composeArgs = append(composeArgs, args[0])
		fmt.Printf("  Restarting %s...\n", args[0])
	} else {
		fmt.Println("  Restarting all services...")
	}

	proc := exec.Command("docker", composeArgs...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Run(); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Restart complete!")
	fmt.Println()
	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Update")
	fmt.Println("  ───────────────────────────")
	fmt.Println()

	// Pull latest images
	fmt.Println("  [1/3] Pulling latest images...")
	pull := exec.Command("docker", "compose", "-f", composeFile(), "pull")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	// Recreate containers
	fmt.Println()
	fmt.Println("  [2/3] Recreating containers...")
	up := exec.Command("docker", "compose", "-f", composeFile(), "up", "-d", "--remove-orphans")
	up.Stdout = os.Stdout
	up.Stderr = os.Stderr
	if err := up.Run(); err != nil {
		return fmt.Errorf("recreate failed: %w", err)
	}

	// Cleanup old images
	fmt.Println()
	fmt.Println("  [3/3] Cleaning up old images...")
	prune := exec.Command("docker", "image", "prune", "-f")
	out, _ := prune.Output()
	if len(out) > 0 {
		// Show space reclaimed
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "reclaimed") {
				fmt.Printf("         %s\n", strings.TrimSpace(line))
			}
		}
	}

	fmt.Println()
	fmt.Println("  ✓ All services updated!")
	fmt.Println()
	return nil
}
