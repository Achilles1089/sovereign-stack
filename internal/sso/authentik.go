package sso

import (
	"fmt"

	"github.com/Achilles1089/sovereign-stack/internal/apps"
	"github.com/Achilles1089/sovereign-stack/internal/config"
	"github.com/Achilles1089/sovereign-stack/internal/docker"
)

// AuthentikConfig holds Authentik SSO configuration
type AuthentikConfig struct {
	Enabled    bool   `yaml:"enabled"`
	AdminEmail string `yaml:"admin_email"`
	SecretKey  string `yaml:"secret_key"`
	Port       int    `yaml:"port"`
}

// AuthentikApp returns the AppManifest for Authentik
func AuthentikApp() *apps.AppManifest {
	return &apps.AppManifest{
		Name:        "authentik",
		DisplayName: "Authentik",
		Description: "Identity provider — SSO for all your apps",
		Category:    "security",
		Version:     "2024.12",
		Website:     "https://goauthentik.io",
		Requires:    apps.AppRequirements{Services: []string{"postgres"}, MinRAMMB: 1024},
		Compose: apps.AppCompose{
			Image: "ghcr.io/goauthentik/server:2024.12",
			Ports: []string{"9443:9443", "9080:9000"},
			Volumes: []string{
				"authentik_media:/media",
				"authentik_templates:/templates",
			},
			Environment: []string{
				"AUTHENTIK_REDIS__HOST=sovereign-redis",
				"AUTHENTIK_POSTGRESQL__HOST=sovereign-postgres",
				"AUTHENTIK_POSTGRESQL__USER=sovereign",
				"AUTHENTIK_POSTGRESQL__NAME=authentik",
				"AUTHENTIK_POSTGRESQL__PASSWORD=sovereign",
				"AUTHENTIK_SECRET_KEY=sovereign-secret-change-me",
			},
		},
		CaddyRoute: &apps.CaddyRoute{Path: "/authentik", Port: 9080},
	}
}

// SupportedApps returns apps that can be configured for SSO via OIDC
func SupportedApps() []string {
	return []string{
		"nextcloud", "gitea", "grafana", "bookstack", "wikijs",
		"portainer", "n8n", "paperless-ngx", "mealie", "firefly",
	}
}

// GenerateOIDCConfig generates the OIDC provider config for an app
func GenerateOIDCConfig(appName string, authentikURL string) map[string]string {
	configs := map[string]map[string]string{
		"nextcloud": {
			"OIDC_LOGIN_PROVIDER_URL":  authentikURL + "/application/o/nextcloud/",
			"OIDC_LOGIN_CLIENT_ID":     "sovereign-nextcloud",
			"OIDC_LOGIN_CLIENT_SECRET": "sovereign-oidc-secret",
			"OIDC_LOGIN_AUTO_REDIRECT": "true",
		},
		"gitea": {
			"GITEA__openid__ENABLE_OPENID_SIGNIN":              "true",
			"GITEA__oauth2__PROVIDER":                          "openidConnect",
			"GITEA__oauth2__OPENID_CONNECT_AUTO_DISCOVERY_URL": authentikURL + "/application/o/gitea/.well-known/openid-configuration",
		},
		"grafana": {
			"GF_AUTH_GENERIC_OAUTH_ENABLED":       "true",
			"GF_AUTH_GENERIC_OAUTH_NAME":          "Sovereign SSO",
			"GF_AUTH_GENERIC_OAUTH_CLIENT_ID":     "sovereign-grafana",
			"GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET": "sovereign-oidc-secret",
			"GF_AUTH_GENERIC_OAUTH_AUTH_URL":      authentikURL + "/application/o/authorize/",
			"GF_AUTH_GENERIC_OAUTH_TOKEN_URL":     authentikURL + "/application/o/token/",
			"GF_AUTH_GENERIC_OAUTH_API_URL":       authentikURL + "/application/o/userinfo/",
			"GF_AUTH_GENERIC_OAUTH_SCOPES":        "openid email profile",
		},
	}

	if cfg, ok := configs[appName]; ok {
		return cfg
	}
	return nil
}

// InstallAuthentik installs Authentik and configures Redis dependency
func InstallAuthentik(cfg *config.Config) error {
	composePath := config.ConfigDir() + "/docker-compose.yml"

	compose, err := docker.LoadComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to load compose: %w", err)
	}

	// Add Redis if not present (Authentik requires it)
	if _, exists := compose.Services["redis"]; !exists {
		redis := &docker.ComposeService{
			Image:         "redis:7-alpine",
			ContainerName: "sovereign-redis",
			Restart:       "unless-stopped",
			Ports:         []string{"6379:6379"},
			Volumes:       []string{"redis_data:/data"},
		}
		docker.AddAppToCompose(compose, "redis", redis)
		compose.Volumes["redis_data"] = nil
	}

	// Install Authentik via the app installer
	return apps.InstallApp(AuthentikApp())
}
