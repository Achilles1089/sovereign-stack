package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/config"
	dockerPkg "github.com/Achilles1089/sovereign-stack/internal/docker"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the health of all sovereign services",
	Long:  `Displays the status of all managed services including Docker containers, AI inference, and resource usage.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Status")
	fmt.Println("  ───────────────────────────")
	fmt.Println()

	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)

	if !dockerPkg.IsDockerAvailable() {
		fmt.Println("  ⚠  Docker is not running.")
		fmt.Println("     Start Docker and try again, or run 'sovereign init' to set up.")
		fmt.Println()
		return nil
	}

	services, err := dockerPkg.CheckAllServices()
	if err != nil {
		return fmt.Errorf("failed to check services: %w", err)
	}

	if len(services) == 0 {
		fmt.Println("  No sovereign services are running.")
		fmt.Println("  Run 'sovereign init' to set up your server.")
		fmt.Println()
		return nil
	}

	// Core services
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  SERVICE\tSTATUS\tIMAGE\tPORTS")
	fmt.Fprintln(w, "  ───────\t──────\t─────\t─────")

	for _, s := range services {
		status := "🔴 Down"
		if s.Running {
			status = "🟢 Up"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", s.Name, status, s.Image, s.Ports)
	}
	w.Flush()

	// Resources
	fmt.Println()
	fmt.Println("  Resources:")
	if cfg.Hardware.CPUModel != "" {
		fmt.Printf("    CPU:  %s (%d cores)\n", cfg.Hardware.CPUModel, cfg.Hardware.CPUCores)
		fmt.Printf("    RAM:  %.1f GB\n", float64(cfg.Hardware.RAMTotalMB)/1024)
		fmt.Printf("    Disk: %d GB free / %d GB total\n", cfg.Hardware.DiskFreeGB, cfg.Hardware.DiskTotalGB)
		if cfg.Hardware.GPUType != "none" && cfg.Hardware.GPUType != "" {
			fmt.Printf("    GPU:  %s (%.1f GB)\n", cfg.Hardware.GPUName, float64(cfg.Hardware.GPUMemoryMB)/1024)
		}
	}

	// AI
	if cfg.AI.Enabled {
		fmt.Println()
		fmt.Println("  AI:")
		fmt.Printf("    Model: %s  |  Mode: %s  |  Host: %s\n", cfg.AI.DefaultModel, cfg.AI.OllamaMode, cfg.AI.OllamaHost)
	}

	fmt.Println()

	// Running services count
	running := 0
	for _, s := range services {
		if s.Running {
			running++
		}
	}

	out, _ := exec.Command("docker", "compose", "-f", config.ConfigDir()+"/docker-compose.yml", "ps", "--format", "json").Output()
	_ = out // For future use

	fmt.Printf("  %d/%d services running\n", running, len(services))
	fmt.Println()
	return nil
}
