package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main Sovereign Stack configuration
type Config struct {
	// Platform info
	Platform string `yaml:"platform"` // "linux", "darwin", "wsl2"
	Mode     string `yaml:"mode"`     // "server" (Linux), "personal" (macOS), "wsl2"

	// Domain and networking
	Domain string `yaml:"domain"` // e.g., "myserver.example.com" or "localhost"
	Port   int    `yaml:"port"`   // Dashboard port, default 8080

	// Services
	Services ServicesConfig `yaml:"services"`

	// AI configuration
	AI AIConfig `yaml:"ai"`

	// Backup configuration
	Backup BackupConfig `yaml:"backup"`

	// Hardware profile (populated during init)
	Hardware HardwareProfile `yaml:"hardware"`
}

// ServicesConfig tracks which core services are enabled
type ServicesConfig struct {
	Ollama   bool `yaml:"ollama"`
	Postgres bool `yaml:"postgres"`
	Caddy    bool `yaml:"caddy"`
	MinIO    bool `yaml:"minio"`
}

// AIConfig holds AI inference settings
type AIConfig struct {
	Enabled      bool   `yaml:"enabled"`
	DefaultModel string `yaml:"default_model"` // e.g., "qwen2.5:7b"
	OllamaHost   string `yaml:"ollama_host"`   // "localhost:11434" or container address
	OllamaMode   string `yaml:"ollama_mode"`   // "container" (Linux) or "native" (macOS)
}

// BackupConfig holds backup settings
type BackupConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Destination string `yaml:"destination"` // Local path or S3 URL
	Schedule    string `yaml:"schedule"`    // Cron expression
	Password    string `yaml:"password"`    // Restic repo password
}

// HardwareProfile stores detected hardware info
type HardwareProfile struct {
	OS          string `yaml:"os"`
	Arch        string `yaml:"arch"`
	CPUModel    string `yaml:"cpu_model"`
	CPUCores    int    `yaml:"cpu_cores"`
	RAMTotalMB  int    `yaml:"ram_total_mb"`
	DiskTotalGB int    `yaml:"disk_total_gb"`
	DiskFreeGB  int    `yaml:"disk_free_gb"`
	GPUType     string `yaml:"gpu_type"` // "nvidia", "amd", "apple_silicon", "intel_arc", "none"
	GPUName     string `yaml:"gpu_name"`
	GPUMemoryMB int    `yaml:"gpu_memory_mb"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Port: 8080,
		Services: ServicesConfig{
			Ollama:   true,
			Postgres: true,
			Caddy:    true,
			MinIO:    false,
		},
		AI: AIConfig{
			Enabled:    true,
			OllamaHost: "localhost:11434",
		},
		Backup: BackupConfig{
			Enabled:  true,
			Schedule: "0 3 * * *", // Daily at 3am
		},
	}
}

// ConfigDir returns the path to the sovereign config directory
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sovereign"
	}
	return filepath.Join(home, ".sovereign")
}

// ConfigPath returns the full path to the config file
func ConfigPath(override string) string {
	if override != "" {
		return override
	}
	return filepath.Join(ConfigDir(), "config.yaml")
}

// DataDir returns the path to sovereign data directory
func DataDir() string {
	return filepath.Join(ConfigDir(), "data")
}

// Load reads the configuration from disk
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadOrDefault loads config from path, or returns defaults if not found
func LoadOrDefault(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}
