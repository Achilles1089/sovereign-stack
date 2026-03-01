package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Achilles1089/sovereign-stack/internal/apps"
	"github.com/Achilles1089/sovereign-stack/internal/backup"
	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/docker"
	"github.com/Achilles1089/sovereign-stack/internal/hardware"
)

// ServerContext holds live system state for context injection
type ServerContext struct {
	Timestamp string                  `json:"timestamp"`
	Platform  string                  `json:"platform"`
	Mode      string                  `json:"mode"`
	Domain    string                  `json:"domain"`
	Hardware  *config.HardwareProfile `json:"hardware"`
	GPUTier   string                  `json:"gpu_tier"`
	Services  []ServiceStatus         `json:"services"`
	Apps      []AppStatus             `json:"apps"`
	AI        AIStatus                `json:"ai"`
	Backup    BackupStatus            `json:"backup"`
}

// ServiceStatus represents a running service
type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// AppStatus represents an installed app
type AppStatus struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Installed bool   `json:"installed"`
}

// AIStatus holds AI inference state
type AIStatus struct {
	Model       string `json:"model"`
	Host        string `json:"host"`
	Mode        string `json:"mode"`
	Recommended string `json:"recommended"`
}

// BackupStatus holds backup state
type BackupStatus struct {
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"`
	HasRepo  bool   `json:"has_repo"`
}

// BuildServerContext gathers live system state
func BuildServerContext(cfg *config.Config) *ServerContext {
	ctx := &ServerContext{
		Timestamp: time.Now().Format(time.RFC3339),
		Platform:  cfg.Platform,
		Mode:      cfg.Mode,
		Domain:    cfg.Domain,
		Hardware:  &cfg.Hardware,
	}

	// GPU tier
	tier := hardware.GetGPUTier(&cfg.Hardware)
	tierNames := map[hardware.GPUTier]string{
		hardware.GPUTierNone: "cpu", hardware.GPUTierBasic: "basic",
		hardware.GPUTierMid: "mid", hardware.GPUTierHigh: "high",
		hardware.GPUTierUltra: "ultra", hardware.GPUTierApex: "apex",
	}
	ctx.GPUTier = tierNames[tier]

	// Services
	services, err := docker.CheckAllServices()
	if err == nil {
		for _, s := range services {
			ctx.Services = append(ctx.Services, ServiceStatus{
				Name:   s.Name,
				Status: s.Status,
			})
		}
	}

	// Apps
	installed, _ := apps.InstalledApps()
	installedMap := make(map[string]bool)
	for _, a := range installed {
		installedMap[a] = true
	}
	for _, app := range apps.BuiltinApps {
		ctx.Apps = append(ctx.Apps, AppStatus{
			Name:      app.Name,
			Category:  app.Category,
			Installed: installedMap[app.Name],
		})
	}

	// AI
	ctx.AI = AIStatus{
		Model:       cfg.AI.DefaultModel,
		Host:        cfg.AI.Host,
		Mode:        "native",
		Recommended: hardware.RecommendedModel(&cfg.Hardware),
	}

	// Backup
	ctx.Backup = BackupStatus{
		Enabled:  cfg.Backup.Enabled,
		Schedule: cfg.Backup.Schedule,
		HasRepo:  backup.IsResticInstalled(),
	}

	return ctx
}

// SystemPrompt generates the system prompt with injected server context
func SystemPrompt(ctx *ServerContext) string {
	contextJSON, _ := json.MarshalIndent(ctx, "", "  ")

	return fmt.Sprintf(`You are the Sovereign Stack AI assistant. You help users manage their self-hosted server.

You have access to live system information. Use it to answer questions accurately.

## Current Server State
%s

## Capabilities
- Answer questions about system resources (CPU, RAM, disk, GPU)
- Report on service status (which are running, which are stopped)
- List available and installed apps from the 30-app marketplace
- Explain AI model capabilities and recommend models for the hardware
- Answer questions about backups and scheduling
- Suggest apps to install based on user needs
- Help troubleshoot service issues

## Interaction Style
- Be concise and direct
- When reporting numbers, format them readably (e.g., "128 GB RAM" not "131072 MB")
- Use emoji sparingly for status (🟢 running, 🔴 stopped, ⚠️ warning)
- If you don't have data about something, say so clearly
- Proactively suggest relevant actions (e.g., "You could install Grafana for monitoring")

## Important Notes
- Platform: %s (%s mode)
- GPU: %s (%s tier, %d MB)
- The server has %d apps in the marketplace, %d categories
`, string(contextJSON),
		ctx.Platform, ctx.Mode,
		ctx.Hardware.GPUName, ctx.GPUTier, ctx.Hardware.GPUMemoryMB,
		len(ctx.Apps), countCategories(ctx.Apps),
	)
}

func countCategories(statuses []AppStatus) int {
	cats := make(map[string]bool)
	for _, a := range statuses {
		cats[a.Category] = true
	}
	return len(cats)
}

// FormatResourceSummary creates a human-readable resource summary
func FormatResourceSummary(hw *config.HardwareProfile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CPU: %s (%d cores)\n", hw.CPUModel, hw.CPUCores))
	sb.WriteString(fmt.Sprintf("RAM: %.1f GB\n", float64(hw.RAMTotalMB)/1024))
	sb.WriteString(fmt.Sprintf("Disk: %d GB total, %d GB free\n", hw.DiskTotalGB, hw.DiskFreeGB))
	sb.WriteString(fmt.Sprintf("GPU: %s (%s, %d MB)\n", hw.GPUName, hw.GPUType, hw.GPUMemoryMB))
	return sb.String()
}
