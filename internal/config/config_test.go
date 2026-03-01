package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrDefault_NoFile(t *testing.T) {
	cfg := LoadOrDefault("/nonexistent/path/config.yaml")
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.Domain != "" {
		// Domain defaults to empty in DefaultConfig, only set during init
	}
	if !cfg.AI.Enabled {
		t.Error("expected AI to be enabled by default")
	}
	if !cfg.Backup.Enabled {
		t.Error("expected backup to be enabled by default")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Domain = "test.example.com"
	cfg.Platform = "linux"
	cfg.Mode = "server"
	cfg.AI.DefaultModel = "llama3.2:3b"
	cfg.Hardware.CPUModel = "Test CPU"
	cfg.Hardware.CPUCores = 8
	cfg.Hardware.RAMTotalMB = 16384

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back
	loaded := LoadOrDefault(cfgPath)
	if loaded.Domain != "test.example.com" {
		t.Errorf("domain mismatch: got %q", loaded.Domain)
	}
	if loaded.Platform != "linux" {
		t.Errorf("platform mismatch: got %q", loaded.Platform)
	}
	if loaded.AI.DefaultModel != "llama3.2:3b" {
		t.Errorf("model mismatch: got %q", loaded.AI.DefaultModel)
	}
	if loaded.Hardware.CPUCores != 8 {
		t.Errorf("cpu cores mismatch: got %d", loaded.Hardware.CPUCores)
	}
	if loaded.Hardware.RAMTotalMB != 16384 {
		t.Errorf("ram mismatch: got %d", loaded.Hardware.RAMTotalMB)
	}
}

func TestConfigPath(t *testing.T) {
	result := ConfigPath("")
	if result == "" {
		t.Error("expected non-empty default config path")
	}

	result = ConfigPath("/custom/path.yaml")
	if result != "/custom/path.yaml" {
		t.Errorf("expected custom path, got %q", result)
	}
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Error("expected non-empty config dir")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Services.LlamaServer {
		t.Error("LlamaServer should be enabled by default")
	}
	if !cfg.Services.Postgres {
		t.Error("Postgres should be enabled by default")
	}
	if !cfg.Services.Caddy {
		t.Error("Caddy should be enabled by default")
	}
	if cfg.Services.MinIO {
		t.Error("MinIO should be disabled by default")
	}
	if cfg.Backup.Schedule != "0 3 * * *" {
		t.Errorf("unexpected default backup schedule: %q", cfg.Backup.Schedule)
	}
}
