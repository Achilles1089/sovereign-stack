package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ServiceHealth represents the health of a Docker service
type ServiceHealth struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Ports   string `json:"ports"`
	Image   string `json:"image"`
	IsApp   bool   `json:"is_app"`
	AppName string `json:"app_name"`
}

// CheckAllServices returns the health of all sovereign-managed containers
func CheckAllServices() ([]ServiceHealth, error) {
	out, err := exec.Command("docker", "ps", "-a",
		"--filter", "name=sovereign-",
		"--format", "{{.Names}}|{{.Status}}|{{.Ports}}|{{.Image}}|{{.Label \"sovereign.app\"}}",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query Docker: %w", err)
	}

	var services []ServiceHealth
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 4 {
			continue
		}

		sh := ServiceHealth{
			Name:    parts[0],
			Running: strings.HasPrefix(parts[1], "Up"),
			Status:  parts[1],
			Ports:   parts[2],
			Image:   parts[3],
		}

		if len(parts) >= 5 && parts[4] != "" {
			sh.IsApp = true
			sh.AppName = parts[4]
		}

		// Clean up the name (remove "sovereign-" prefix for display)
		sh.Uptime = extractUptime(parts[1])

		services = append(services, sh)
	}

	return services, nil
}

// IsDockerAvailable checks if Docker daemon is running
func IsDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := exec.CommandContext(ctx, "docker", "info").Run()
	return err == nil
}

// ComposeUp runs docker compose up for the sovereign stack
func ComposeUp(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// ComposeDown stops all sovereign services
func ComposeDown(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", composePath, "down")
	return cmd.Run()
}

// ComposePull pulls latest images
func ComposePull(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", composePath, "pull")
	return cmd.Run()
}

// extractUptime parses Docker status string to get clean uptime
func extractUptime(status string) string {
	if strings.HasPrefix(status, "Up ") {
		return strings.TrimPrefix(status, "Up ")
	}
	return status
}
