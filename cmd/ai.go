package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	aiPkg "github.com/Achilles1089/sovereign-stack/internal/ai"
	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/hardware"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Manage AI inference",
	Long:  `Manage local AI models, check GPU status, and chat with your AI.`,
}

var aiStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show AI inference status",
	RunE:  runAIStatus,
}

var aiPullCmd = &cobra.Command{
	Use:   "pull <model>",
	Short: "Download an AI model",
	Args:  cobra.ExactArgs(1),
	RunE:  runAIPull,
}

var aiChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with your local AI",
	Long:  `Start an interactive chat session with your local AI model via llama-server.`,
	RunE:  runAIChat,
}

var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List installed AI models",
	RunE:  runAIModels,
}

var aiCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Show available AI models for your hardware",
	RunE:  runAICatalog,
}

func init() {
	aiCmd.AddCommand(aiStatusCmd)
	aiCmd.AddCommand(aiPullCmd)
	aiCmd.AddCommand(aiChatCmd)
	aiCmd.AddCommand(aiModelsCmd)
	aiCmd.AddCommand(aiCatalogCmd)
	rootCmd.AddCommand(aiCmd)
}

func getLlamaClient() *aiPkg.Client {
	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)
	host := cfg.AI.Host
	if host == "" {
		host = "localhost:8085"
	}
	return aiPkg.NewClient(host)
}

func runAIStatus(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — AI Status")
	fmt.Println("  ──────────────────────────────")
	fmt.Println()

	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)

	hw := &cfg.Hardware
	if hw.GPUType != "" && hw.GPUType != "none" {
		fmt.Printf("  GPU:           %s (%d MB)\n", hw.GPUName, hw.GPUMemoryMB)
	} else {
		fmt.Println("  GPU:           None (CPU inference)")
	}
	fmt.Printf("  Recommended:   %s\n", hardware.RecommendedModelDescription(hw))
	fmt.Printf("  Default Model: %s\n", cfg.AI.DefaultModel)
	fmt.Println()

	client := getLlamaClient()
	fmt.Printf("  Server Host:   %s\n", client.Host)
	fmt.Printf("  Engine:        llama-server\n")

	if client.IsRunning() {
		fmt.Println("  llama-server:  🟢 Running")

		models, err := client.ListModels()
		if err == nil {
			fmt.Printf("  Models:        %d installed\n", len(models))
			for _, m := range models {
				sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
				fmt.Printf("                 • %s (%.1f GB)\n", m.Name, sizeGB)
			}
		}
	} else {
		fmt.Println("  llama-server:  🔴 Not reachable")
		fmt.Println()
		fmt.Println("  Start with: llama-server -m <model.gguf> --host 0.0.0.0 --port 8085")
	}

	fmt.Println()
	return nil
}

func runAIPull(cmd *cobra.Command, args []string) error {
	model := args[0]
	client := getLlamaClient()

	fmt.Printf("\n  Pulling model: %s\n", model)
	fmt.Printf("  From: %s\n\n", client.Host)

	err := client.PullModel(model, func(status string, completed, total int64) {
		if total > 0 {
			pct := float64(completed) / float64(total) * 100
			fmt.Printf("\r  %-30s %.1f%%", status, pct)
		} else {
			fmt.Printf("\r  %s", status)
		}
	})

	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	fmt.Printf("\n  ✓ Model %s pulled successfully!\n\n", model)
	return nil
}

func runAIChat(cmd *cobra.Command, args []string) error {
	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)
	client := getLlamaClient()
	model := cfg.AI.DefaultModel
	if model == "" {
		model = "rwkv7-2.9B"
	}

	if !client.IsRunning() {
		return fmt.Errorf("llama-server is not running. Start it first, then try again")
	}

	fmt.Println()
	fmt.Printf("  ⚡ Sovereign AI Chat — Model: %s\n", model)
	fmt.Println("  Type 'exit' or Ctrl+C to quit")
	fmt.Println("  ──────────────────────────────")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var history []aiPkg.ChatMessage

	for {
		fmt.Print("  You > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("  Goodbye! 👋")
			break
		}

		history = append(history, aiPkg.ChatMessage{Role: "user", Content: input})

		fmt.Print("  AI  > ")

		var response strings.Builder
		err := client.Chat(model, history, func(content string, done bool) {
			fmt.Print(content)
			response.WriteString(content)
		})

		if err != nil {
			fmt.Printf("\n  Error: %v\n", err)
			// Remove failed message from history
			history = history[:len(history)-1]
		} else {
			history = append(history, aiPkg.ChatMessage{Role: "assistant", Content: response.String()})
		}

		fmt.Println()
		fmt.Println()
	}

	return nil
}

func runAIModels(cmd *cobra.Command, args []string) error {
	client := getLlamaClient()

	models, err := client.ListModels()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("  Installed AI Models:")
	fmt.Println("  ────────────────────")
	for _, m := range models {
		sizeGB := float64(m.Size) / (1024 * 1024 * 1024)
		fmt.Printf("  • %-30s  %.1f GB\n", m.Name, sizeGB)
	}
	if len(models) == 0 {
		fmt.Println("  No models installed. Pull one with: sovereign ai pull rwkv7-2.9B")
	}
	fmt.Println()
	return nil
}

func runAICatalog(cmd *cobra.Command, args []string) error {
	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)
	tier := hardware.GetGPUTier(&cfg.Hardware)

	tierName := map[hardware.GPUTier]string{
		hardware.GPUTierNone:  "cpu",
		hardware.GPUTierBasic: "basic",
		hardware.GPUTierMid:   "mid",
		hardware.GPUTierHigh:  "high",
		hardware.GPUTierUltra: "ultra",
		hardware.GPUTierApex:  "apex",
	}[tier]

	models := aiPkg.GetModelsForTier(tierName)

	fmt.Println()
	fmt.Printf("  AI Models Available for Your Hardware (Tier: %s)\n", tierName)
	fmt.Println("  ──────────────────────────────────────────────────")
	fmt.Println()

	for _, m := range models {
		fmt.Printf("  %-25s  %.1f GB  %s\n", m.Name, m.SizeGB, m.Description)
	}

	fmt.Println()
	fmt.Println("  Pull with: sovereign ai pull <model-name>")
	fmt.Println()
	return nil
}
