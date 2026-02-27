package apps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/docker"
)

// AppManifest represents a full app definition with compose config
type AppManifest struct {
	Name        string          `yaml:"name"`
	DisplayName string          `yaml:"display_name"`
	Description string          `yaml:"description"`
	Category    string          `yaml:"category"`
	Version     string          `yaml:"version"`
	Website     string          `yaml:"website"`
	Icon        string          `yaml:"icon"`
	Requires    AppRequirements `yaml:"requires"`
	Compose     AppCompose      `yaml:"compose"`
	CaddyRoute  *CaddyRoute     `yaml:"caddy_route,omitempty"`
}

// AppRequirements defines what an app needs
type AppRequirements struct {
	Services  []string `yaml:"services"` // e.g., ["postgres"]
	MinRAMMB  int      `yaml:"min_ram_mb"`
	MinDiskGB int      `yaml:"min_disk_gb"`
}

// AppCompose defines the Docker Compose snippet for an app
type AppCompose struct {
	Image       string   `yaml:"image"`
	Ports       []string `yaml:"ports"`
	Volumes     []string `yaml:"volumes"`
	Environment []string `yaml:"environment"`
	DependsOn   []string `yaml:"depends_on"`
}

// CaddyRoute defines how Caddy should proxy to this app
type CaddyRoute struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
	Port     int    `yaml:"port"`
}

// BuiltinApps is the embedded catalog of apps available at MVP
var BuiltinApps = []AppManifest{
	{
		Name: "nextcloud", DisplayName: "Nextcloud", Description: "File sync, share, and collaboration",
		Category: "productivity", Version: "29", Website: "https://nextcloud.com",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 512, MinDiskGB: 10},
		Compose: AppCompose{
			Image: "nextcloud:29", Ports: []string{"8080:80"},
			Volumes:     []string{"nextcloud_data:/var/www/html"},
			Environment: []string{"POSTGRES_HOST=sovereign-postgres", "POSTGRES_USER=sovereign", "POSTGRES_PASSWORD=sovereign", "POSTGRES_DB=nextcloud"},
		},
		CaddyRoute: &CaddyRoute{Path: "/nextcloud", Port: 8080},
	},
	{
		Name: "jellyfin", DisplayName: "Jellyfin", Description: "Media streaming server",
		Category: "media", Version: "10.9", Website: "https://jellyfin.org",
		Compose:    AppCompose{Image: "jellyfin/jellyfin:10.9", Ports: []string{"8096:8096"}, Volumes: []string{"jellyfin_data:/config", "jellyfin_media:/media"}},
		CaddyRoute: &CaddyRoute{Path: "/jellyfin", Port: 8096},
	},
	{
		Name: "immich", DisplayName: "Immich", Description: "Self-hosted photo & video management",
		Category: "media", Version: "1.99", Website: "https://immich.app",
		Requires:   AppRequirements{Services: []string{"postgres"}, MinRAMMB: 2048, MinDiskGB: 20},
		Compose:    AppCompose{Image: "ghcr.io/immich-app/immich-server:release", Ports: []string{"2283:2283"}, Volumes: []string{"immich_data:/usr/src/app/upload"}},
		CaddyRoute: &CaddyRoute{Path: "/immich", Port: 2283},
	},
	{
		Name: "adguard-home", DisplayName: "AdGuard Home", Description: "Network-wide ad blocking",
		Category: "network", Version: "0.107",
		Compose: AppCompose{Image: "adguard/adguardhome:latest", Ports: []string{"3000:3000", "53:53/udp"}, Volumes: []string{"adguard_data:/opt/adguardhome/work", "adguard_conf:/opt/adguardhome/conf"}},
	},
	{
		Name: "vaultwarden", DisplayName: "Vaultwarden", Description: "Bitwarden-compatible password manager",
		Category: "security", Version: "1.31",
		Compose:    AppCompose{Image: "vaultwarden/server:latest", Ports: []string{"8880:80"}, Volumes: []string{"vaultwarden_data:/data"}},
		CaddyRoute: &CaddyRoute{Path: "/vaultwarden", Port: 8880},
	},
	{
		Name: "gitea", DisplayName: "Gitea", Description: "Lightweight Git hosting",
		Category: "development", Version: "1.22",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 256},
		Compose: AppCompose{Image: "gitea/gitea:1.22", Ports: []string{"3001:3000", "2222:22"}, Volumes: []string{"gitea_data:/data"},
			Environment: []string{"GITEA__database__DB_TYPE=postgres", "GITEA__database__HOST=sovereign-postgres:5432", "GITEA__database__NAME=gitea", "GITEA__database__USER=sovereign", "GITEA__database__PASSWD=sovereign"}},
		CaddyRoute: &CaddyRoute{Path: "/gitea", Port: 3001},
	},
	{
		Name: "n8n", DisplayName: "n8n", Description: "Workflow automation tool",
		Category: "automation", Version: "1.76",
		Compose:    AppCompose{Image: "n8nio/n8n:latest", Ports: []string{"5678:5678"}, Volumes: []string{"n8n_data:/home/node/.n8n"}},
		CaddyRoute: &CaddyRoute{Path: "/n8n", Port: 5678},
	},
	{
		Name: "uptime-kuma", DisplayName: "Uptime Kuma", Description: "Beautiful uptime monitoring",
		Category: "monitoring", Version: "1.23",
		Compose:    AppCompose{Image: "louislam/uptime-kuma:1", Ports: []string{"3002:3001"}, Volumes: []string{"uptime_data:/app/data"}},
		CaddyRoute: &CaddyRoute{Path: "/uptime", Port: 3002},
	},
	{
		Name: "stirling-pdf", DisplayName: "Stirling PDF", Description: "PDF manipulation tools",
		Category: "productivity", Version: "0.34",
		Compose:    AppCompose{Image: "frooodle/s-pdf:latest", Ports: []string{"8181:8080"}, Volumes: []string{"stirling_data:/usr/share/tessdata"}},
		CaddyRoute: &CaddyRoute{Path: "/pdf", Port: 8181},
	},
	{
		Name: "portainer", DisplayName: "Portainer", Description: "Container management UI",
		Category: "system", Version: "2.21",
		Compose:    AppCompose{Image: "portainer/portainer-ce:latest", Ports: []string{"9000:9000"}, Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock", "portainer_data:/data"}},
		CaddyRoute: &CaddyRoute{Path: "/portainer", Port: 9000},
	},
	{
		Name: "syncthing", DisplayName: "Syncthing", Description: "Peer-to-peer file synchronization",
		Category: "productivity", Version: "1.27",
		Compose:    AppCompose{Image: "syncthing/syncthing:latest", Ports: []string{"8384:8384", "22000:22000"}, Volumes: []string{"syncthing_data:/var/syncthing"}},
		CaddyRoute: &CaddyRoute{Path: "/syncthing", Port: 8384},
	},
	{
		Name: "open-webui", DisplayName: "Open WebUI", Description: "ChatGPT-style interface for local AI",
		Category: "ai", Version: "0.5",
		Compose: AppCompose{Image: "ghcr.io/open-webui/open-webui:main", Ports: []string{"3003:8080"}, Volumes: []string{"openwebui_data:/app/backend/data"},
			Environment: []string{"OLLAMA_BASE_URL=http://sovereign-ollama:11434"}},
		CaddyRoute: &CaddyRoute{Path: "/webui", Port: 3003},
	},
}

// FindApp looks up an app by name in the catalog
func FindApp(name string) *AppManifest {
	for i := range BuiltinApps {
		if BuiltinApps[i].Name == name {
			return &BuiltinApps[i]
		}
	}
	return nil
}

// InstallApp installs an app by adding it to the compose file and starting it
func InstallApp(app *AppManifest) error {
	composePath := filepath.Join(config.ConfigDir(), "docker-compose.yml")

	// Load existing compose
	compose, err := docker.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose file: %w", err)
	}

	// Check if already installed
	if _, exists := compose.Services[app.Name]; exists {
		return fmt.Errorf("app '%s' is already installed", app.Name)
	}

	// Create service definition
	service := &docker.ComposeService{
		Image:         app.Compose.Image,
		ContainerName: "sovereign-" + app.Name,
		Restart:       "unless-stopped",
		Ports:         app.Compose.Ports,
		Volumes:       app.Compose.Volumes,
		Environment:   app.Compose.Environment,
		DependsOn:     app.Compose.DependsOn,
	}

	// Add to compose
	docker.AddAppToCompose(compose, app.Name, service)

	// Add volumes
	for _, v := range app.Compose.Volumes {
		volName := strings.Split(v, ":")[0]
		// Only add named volumes, not bind mounts
		if !strings.HasPrefix(volName, "/") && !strings.HasPrefix(volName, ".") {
			compose.Volumes[volName] = nil
		}
	}

	// Write updated compose
	if err := docker.WriteComposeFile(compose, composePath); err != nil {
		return fmt.Errorf("failed to update compose file: %w", err)
	}

	// Start the new service
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d", app.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveApp removes an app from the compose file and stops it
func RemoveApp(appName string) error {
	composePath := filepath.Join(config.ConfigDir(), "docker-compose.yml")

	// Stop the container
	exec.Command("docker", "stop", "sovereign-"+appName).Run()
	exec.Command("docker", "rm", "sovereign-"+appName).Run()

	// Load and modify compose
	compose, err := docker.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose file: %w", err)
	}

	docker.RemoveAppFromCompose(compose, appName)

	return docker.WriteComposeFile(compose, composePath)
}

// InstalledApps returns a list of installed app names
func InstalledApps() ([]string, error) {
	composePath := filepath.Join(config.ConfigDir(), "docker-compose.yml")

	compose, err := docker.LoadComposeFile(composePath)
	if err != nil {
		return nil, err
	}

	var installed []string
	for name, svc := range compose.Services {
		if svc.Labels != nil {
			if _, isApp := svc.Labels["sovereign.app"]; isApp {
				installed = append(installed, name)
			}
		}
	}

	return installed, nil
}
