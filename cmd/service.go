package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs for a sovereign service",
	Long:  `Stream Docker logs for a specific service. If no service is specified, shows logs for all sovereign services.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLogs,
}

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart a sovereign service",
	Long:  `Restart a specific Docker container. If no service is specified, restarts all sovereign services.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRestart,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Sovereign Stack and all services",
	Long:  `Pull the latest images for all running services and recreate containers.`,
	RunE:  runUpdate,
}

func init() {
	logsCmd.Flags().IntP("tail", "n", 100, "Number of lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")

	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(updateCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	tail, _ := cmd.Flags().GetInt("tail")
	follow, _ := cmd.Flags().GetBool("follow")

	if len(args) == 0 {
		fmt.Println("Showing logs for all sovereign services...")
		cmdArgs := []string{"compose", "-f", getComposeFilePath(), "logs", fmt.Sprintf("--tail=%d", tail)}
		if follow {
			cmdArgs = append(cmdArgs, "-f")
		}
		c := exec.Command("docker", cmdArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	service := args[0]
	cmdArgs := []string{"logs", fmt.Sprintf("--tail=%d", tail)}
	if follow {
		cmdArgs = append(cmdArgs, "-f")
	}
	cmdArgs = append(cmdArgs, "sovereign-"+service)

	c := exec.Command("docker", cmdArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runRestart(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Println("Restarting all sovereign services...")
		c := exec.Command("docker", "compose", "-f", getComposeFilePath(), "restart")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	service := "sovereign-" + args[0]
	fmt.Printf("Restarting %s...\n", service)
	return exec.Command("docker", "restart", service).Run()
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Updating Sovereign Stack...")
	fmt.Println()

	composePath := getComposeFilePath()

	// Pull latest images
	fmt.Println("  Pulling latest images...")
	pullCmd := exec.Command("docker", "compose", "-f", composePath, "pull")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Recreate containers
	fmt.Println("  Recreating containers...")
	upCmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d", "--remove-orphans")
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("failed to recreate containers: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Update complete!")
	return nil
}

func getComposeFilePath() string {
	return config.ConfigDir() + "/docker-compose.yml"
}
