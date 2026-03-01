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
			Environment: []string{"LLAMA_SERVER_URL=http://localhost:8085"}},
		CaddyRoute: &CaddyRoute{Path: "/webui", Port: 3003},
	},

	// Phase 2 additions (18 apps)
	{Name: "home-assistant", DisplayName: "Home Assistant", Description: "Open-source home automation platform",
		Category: "smart-home", Version: "2024.12", Website: "https://home-assistant.io",
		Requires:   AppRequirements{MinRAMMB: 1024, MinDiskGB: 5},
		Compose:    AppCompose{Image: "ghcr.io/home-assistant/home-assistant:stable", Ports: []string{"8123:8123"}, Volumes: []string{"homeassistant_data:/config"}},
		CaddyRoute: &CaddyRoute{Path: "/homeassistant", Port: 8123},
	},
	{Name: "paperless-ngx", DisplayName: "Paperless-ngx", Description: "Document management with OCR",
		Category: "productivity", Version: "2.14", Website: "https://docs.paperless-ngx.com",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 1024, MinDiskGB: 10},
		Compose: AppCompose{Image: "ghcr.io/paperless-ngx/paperless-ngx:latest", Ports: []string{"8010:8000"}, Volumes: []string{"paperless_data:/usr/src/paperless/data", "paperless_media:/usr/src/paperless/media"},
			Environment: []string{"PAPERLESS_DBHOST=sovereign-postgres", "PAPERLESS_DBUSER=sovereign", "PAPERLESS_DBPASS=sovereign", "PAPERLESS_DBNAME=paperless"}},
		CaddyRoute: &CaddyRoute{Path: "/paperless", Port: 8010},
	},
	{Name: "bookstack", DisplayName: "BookStack", Description: "Self-hosted wiki and documentation",
		Category: "productivity", Version: "24.12", Website: "https://bookstackapp.com",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 256},
		Compose: AppCompose{Image: "lscr.io/linuxserver/bookstack:latest", Ports: []string{"6875:80"}, Volumes: []string{"bookstack_data:/config"},
			Environment: []string{"DB_HOST=sovereign-postgres", "DB_USER=sovereign", "DB_PASS=sovereign", "DB_DATABASE=bookstack"}},
		CaddyRoute: &CaddyRoute{Path: "/bookstack", Port: 6875},
	},
	{Name: "grafana", DisplayName: "Grafana", Description: "Metrics visualization and dashboards",
		Category: "monitoring", Version: "11.4", Website: "https://grafana.com",
		Compose:    AppCompose{Image: "grafana/grafana-oss:latest", Ports: []string{"3004:3000"}, Volumes: []string{"grafana_data:/var/lib/grafana"}},
		CaddyRoute: &CaddyRoute{Path: "/grafana", Port: 3004},
	},
	{Name: "prometheus", DisplayName: "Prometheus", Description: "Time-series monitoring and alerting",
		Category: "monitoring", Version: "2.55", Website: "https://prometheus.io",
		Compose:    AppCompose{Image: "prom/prometheus:latest", Ports: []string{"9090:9090"}, Volumes: []string{"prometheus_data:/prometheus"}},
		CaddyRoute: &CaddyRoute{Path: "/prometheus", Port: 9090},
	},
	{Name: "wikijs", DisplayName: "Wiki.js", Description: "Modern, powerful wiki engine",
		Category: "productivity", Version: "2.5", Website: "https://js.wiki",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 512},
		Compose: AppCompose{Image: "ghcr.io/requarks/wiki:2", Ports: []string{"3005:3000"}, Volumes: []string{"wikijs_data:/wiki/data"},
			Environment: []string{"DB_TYPE=postgres", "DB_HOST=sovereign-postgres", "DB_PORT=5432", "DB_USER=sovereign", "DB_PASS=sovereign", "DB_NAME=wikijs"}},
		CaddyRoute: &CaddyRoute{Path: "/wiki", Port: 3005},
	},
	{Name: "plausible", DisplayName: "Plausible", Description: "Privacy-friendly web analytics",
		Category: "analytics", Version: "2.1", Website: "https://plausible.io",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 512},
		Compose: AppCompose{Image: "ghcr.io/plausible/community-edition:v2.1", Ports: []string{"8011:8000"}, Volumes: []string{"plausible_data:/var/lib/plausible"},
			Environment: []string{"DATABASE_URL=postgres://sovereign:sovereign@sovereign-postgres:5432/plausible", "BASE_URL=http://localhost:8011", "SECRET_KEY_BASE=sovereign-secret-key-change-in-production"}},
		CaddyRoute: &CaddyRoute{Path: "/plausible", Port: 8011},
	},
	{Name: "mealie", DisplayName: "Mealie", Description: "Recipe management and meal planning",
		Category: "lifestyle", Version: "2.4", Website: "https://mealie.io",
		Compose: AppCompose{Image: "ghcr.io/mealie-recipes/mealie:latest", Ports: []string{"9925:9000"}, Volumes: []string{"mealie_data:/app/data"},
			Environment: []string{"ALLOW_SIGNUP=false", "TZ=UTC"}},
		CaddyRoute: &CaddyRoute{Path: "/mealie", Port: 9925},
	},
	{Name: "firefly", DisplayName: "Firefly III", Description: "Personal finance manager",
		Category: "finance", Version: "6.1", Website: "https://firefly-iii.org",
		Requires: AppRequirements{Services: []string{"postgres"}, MinRAMMB: 512},
		Compose: AppCompose{Image: "fireflyiii/core:latest", Ports: []string{"8012:8080"}, Volumes: []string{"firefly_data:/var/www/html/storage/upload"},
			Environment: []string{"DB_HOST=sovereign-postgres", "DB_PORT=5432", "DB_CONNECTION=pgsql", "DB_DATABASE=firefly", "DB_USERNAME=sovereign", "DB_PASSWORD=sovereign", "APP_KEY=sovereign-change-this-32-char-key!!"}},
		CaddyRoute: &CaddyRoute{Path: "/firefly", Port: 8012},
	},
	{Name: "photoprism", DisplayName: "PhotoPrism", Description: "AI-powered photo management",
		Category: "media", Version: "240915", Website: "https://photoprism.app",
		Requires: AppRequirements{MinRAMMB: 2048, MinDiskGB: 20},
		Compose: AppCompose{Image: "photoprism/photoprism:latest", Ports: []string{"2342:2342"}, Volumes: []string{"photoprism_originals:/photoprism/originals", "photoprism_storage:/photoprism/storage"},
			Environment: []string{"PHOTOPRISM_ADMIN_PASSWORD=sovereign", "PHOTOPRISM_SITE_URL=http://localhost:2342/"}},
		CaddyRoute: &CaddyRoute{Path: "/photoprism", Port: 2342},
	},
	{Name: "audiobookshelf", DisplayName: "Audiobookshelf", Description: "Self-hosted audiobook and podcast server",
		Category: "media", Version: "2.17", Website: "https://www.audiobookshelf.org",
		Compose:    AppCompose{Image: "ghcr.io/advplyr/audiobookshelf:latest", Ports: []string{"13378:80"}, Volumes: []string{"audiobookshelf_data:/config", "audiobookshelf_meta:/metadata"}},
		CaddyRoute: &CaddyRoute{Path: "/audiobookshelf", Port: 13378},
	},
	{Name: "kavita", DisplayName: "Kavita", Description: "Digital library for comics, manga, and books",
		Category: "media", Version: "0.8", Website: "https://kavitareader.com",
		Compose:    AppCompose{Image: "jvmilazz0/kavita:latest", Ports: []string{"5000:5000"}, Volumes: []string{"kavita_data:/kavita/config"}},
		CaddyRoute: &CaddyRoute{Path: "/kavita", Port: 5000},
	},
	{Name: "navidrome", DisplayName: "Navidrome", Description: "Modern music server and streamer",
		Category: "media", Version: "0.53", Website: "https://navidrome.org",
		Compose:    AppCompose{Image: "deluan/navidrome:latest", Ports: []string{"4533:4533"}, Volumes: []string{"navidrome_data:/data", "navidrome_music:/music:ro"}},
		CaddyRoute: &CaddyRoute{Path: "/navidrome", Port: 4533},
	},
	{Name: "minio", DisplayName: "MinIO", Description: "S3-compatible object storage",
		Category: "system", Version: "2024", Website: "https://min.io",
		Compose: AppCompose{Image: "minio/minio:latest", Ports: []string{"9100:9000", "9101:9001"}, Volumes: []string{"minio_data:/data"},
			Environment: []string{"MINIO_ROOT_USER=sovereign", "MINIO_ROOT_PASSWORD=sovereign123"}},
		CaddyRoute: &CaddyRoute{Path: "/minio", Port: 9101},
	},
	{Name: "changedetection", DisplayName: "Changedetection.io", Description: "Website change monitoring",
		Category: "monitoring", Version: "0.46", Website: "https://changedetection.io",
		Compose:    AppCompose{Image: "ghcr.io/dgtlmoon/changedetection.io:latest", Ports: []string{"5555:5000"}, Volumes: []string{"changedetection_data:/datastore"}},
		CaddyRoute: &CaddyRoute{Path: "/changedetection", Port: 5555},
	},
	{Name: "it-tools", DisplayName: "IT-Tools", Description: "Collection of developer tools (JSON, Base64, hashing, etc.)",
		Category: "development", Version: "2024.10",
		Compose:    AppCompose{Image: "corentinth/it-tools:latest", Ports: []string{"8013:80"}},
		CaddyRoute: &CaddyRoute{Path: "/it-tools", Port: 8013},
	},
	{Name: "speedtest-tracker", DisplayName: "Speedtest Tracker", Description: "Internet speed monitoring over time",
		Category: "monitoring", Version: "0.20",
		Compose:    AppCompose{Image: "lscr.io/linuxserver/speedtest-tracker:latest", Ports: []string{"8765:80"}, Volumes: []string{"speedtest_data:/config"}},
		CaddyRoute: &CaddyRoute{Path: "/speedtest", Port: 8765},
	},
	{Name: "homarr", DisplayName: "Homarr", Description: "Sleek server dashboard and app organizer",
		Category: "system", Version: "0.15", Website: "https://homarr.dev",
		Compose:    AppCompose{Image: "ghcr.io/ajnart/homarr:latest", Ports: []string{"7575:7575"}, Volumes: []string{"homarr_data:/app/data/configs", "/var/run/docker.sock:/var/run/docker.sock:ro"}},
		CaddyRoute: &CaddyRoute{Path: "/homarr", Port: 7575},
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
