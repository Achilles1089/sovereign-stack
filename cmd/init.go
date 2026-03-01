package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/docker"
	"github.com/Achilles1089/sovereign-stack/internal/hardware"
	"github.com/Achilles1089/sovereign-stack/internal/platform"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your sovereign server",
	Long: `Run the Sovereign Stack setup wizard.

Detects your hardware, installs Docker and AI inference,
and spins up your personal cloud — all in one command.

Behavior varies by platform:
  Linux:  Full server mode — Docker Engine + llama-server
  macOS:  Personal mode — Docker Desktop + llama-server (Metal GPU)
  WSL2:   WSL2 mode — Docker Desktop + llama-server`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  \u26a1 Sovereign Stack \u2014 Init Wizard")
	fmt.Println("  \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")
	fmt.Println()

	// Step 1: Platform detection
	fmt.Println("  [1/8] Detecting platform...")
	pinfo := platform.Detect()
	fmt.Printf("         Platform: %s\n", pinfo)
	fmt.Printf("         Mode:     %s\n", pinfo.Mode)
	fmt.Println()

	// Step 2: Hardware scan
	fmt.Println("  [2/8] Scanning hardware...")
	hw, err := hardware.Detect()
	if err != nil {
		return fmt.Errorf("hardware detection failed: %w", err)
	}
	fmt.Println(indentText(hardware.Summary(hw), "         "))
	fmt.Println()

	// Step 3: Pre-flight checks
	fmt.Println("  [3/8] Running pre-flight checks...")
	if err := preflightChecks(pinfo, hw); err != nil {
		return err
	}
	fmt.Println("         \u2713 All checks passed")
	fmt.Println()

	// Step 4: AI model recommendation
	fmt.Println("  [4/8] Selecting AI model...")
	model := hardware.RecommendedModel(hw)
	fmt.Printf("         Recommended: %s\n", hardware.RecommendedModelDescription(hw))
	fmt.Println()

	// Step 5: Generate config
	fmt.Println("  [5/8] Generating configuration...")
	cfg := config.DefaultConfig()
	cfg.Platform = string(pinfo.Platform)
	cfg.Mode = string(pinfo.Mode)
	cfg.Hardware = *hw
	cfg.AI.DefaultModel = model

	// llama-server runs natively on all platforms
	cfg.AI.Host = "localhost:8085"

	if pinfo.Mode == platform.ModeServer {
		cfg.Domain = "localhost"
		fmt.Println("         Domain: localhost (configure later with your real domain)")
	} else {
		cfg.Domain = "localhost"
		fmt.Println("         Domain: localhost")
	}

	cfgPath := config.ConfigPath(GetConfigPath())
	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("         Config saved to: %s\n", cfgPath)
	fmt.Println()

	// Step 6: Generate Docker Compose + Caddyfile (always write, even without Docker)
	fmt.Println("  [6/8] Generating Docker Compose...")
	compose := docker.GenerateCoreCompose(cfg)
	composePath := config.ConfigDir() + "/docker-compose.yml"
	if err := docker.WriteComposeFile(compose, composePath); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	fmt.Printf("         Compose: %s\n", composePath)

	if err := docker.WriteCaddyfile(cfg); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}
	fmt.Printf("         Caddy:   %s/Caddyfile\n", config.ConfigDir())
	fmt.Println()

	// Step 7: Docker check
	fmt.Println("  [7/8] Checking Docker...")
	dockerReady := true
	if err := checkDocker(pinfo); err != nil {
		dockerReady = false
		fmt.Println("         \u26a0  Docker not available \u2014 services won't start yet")
		fmt.Println("         Install Docker, then run: docker compose -f " + composePath + " up -d")
	} else {
		fmt.Println("         \u2713 Docker is ready")
	}
	fmt.Println()

	// Step 8: Summary
	fmt.Println("  [8/8] Setup complete!")
	if !dockerReady {
		fmt.Println("         (Docker services pending \u2014 install Docker to start them)")
	}
	fmt.Println()
	printSummary(cfg, pinfo)

	return nil
}

func preflightChecks(pinfo *platform.Info, hw *config.HardwareProfile) error {
	if hw.DiskFreeGB < 20 {
		return fmt.Errorf("insufficient disk space: %d GB free (minimum 20 GB required)", hw.DiskFreeGB)
	}
	if hw.RAMTotalMB < 2048 {
		fmt.Println("         \u26a0 Low RAM detected (<2 GB). Performance may be limited.")
	}
	if pinfo.Platform == platform.PlatformLinux && !pinfo.IsRoot {
		return fmt.Errorf("sovereign init requires root privileges on Linux. Run with: sudo sovereign init")
	}
	return nil
}

func checkDocker(pinfo *platform.Info) error {
	_, err := exec.Command("docker", "version").Output()
	if err != nil {
		if pinfo.NeedsDockerDesktop() {
			fmt.Println("         \u2717 Docker Desktop not found.")
			fmt.Println("         Please install Docker Desktop from: https://docker.com/products/docker-desktop")
			return fmt.Errorf("Docker Desktop is required but not installed")
		}
		fmt.Println("         Docker not found.")
		fmt.Println("         Run: curl -fsSL https://get.docker.com | sh")
		return fmt.Errorf("Docker Engine is required but not installed")
	}
	return nil
}

func printSummary(cfg *config.Config, pinfo *platform.Info) {
	fmt.Println("  \u250c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510")
	fmt.Println("  \u2502         \ud83c\udff0 Sovereign Stack is Ready         \u2502")
	fmt.Println("  \u251c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2524")
	fmt.Printf("  \u2502  Platform:  %-31s \u2502\n", pinfo)
	fmt.Printf("  \u2502  AI Model:  %-31s \u2502\n", cfg.AI.DefaultModel)
	fmt.Printf("  \u2502  Engine:    %-31s \u2502\n", "llama-server")
	fmt.Printf("  \u2502  Config:    %-31s \u2502\n", "~/.sovereign/config.yaml")
	fmt.Println("  \u251c\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2524")
	fmt.Println("  \u2502  Next steps:                                \u2502")
	fmt.Println("  \u2502    sovereign status    \u2014 Check services     \u2502")
	fmt.Println("  \u2502    sovereign app list  \u2014 Browse apps        \u2502")
	fmt.Println("  \u2502    sovereign ai chat   \u2014 Chat with AI       \u2502")
	fmt.Println("  \u2502    sovereign ai pull   \u2014 Download AI model  \u2502")
	fmt.Println("  \u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518")
	fmt.Println()
}

func indentText(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
