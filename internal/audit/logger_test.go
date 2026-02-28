package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".sovereign", "audit"), 0755)
	defer os.Unsetenv("HOME")

	logger := NewLogger()
	logger.logDir = filepath.Join(tmpDir, "audit")

	// Log some events
	logger.LogAppInstall("nextcloud", true)
	logger.LogAppInstall("grafana", true)
	logger.LogBackup("daily", true)
	logger.LogConfigChange("domain", "localhost", "myserver.com")

	// Query all
	events, err := logger.Query("", 0)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(events) != 4 {
		t.Errorf("expected 4 events, got %d", len(events))
	}

	// Query by action
	appEvents, err := logger.Query("app.install", 0)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(appEvents) != 2 {
		t.Errorf("expected 2 app.install events, got %d", len(appEvents))
	}

	// Query with limit
	limited, err := logger.Query("", 2)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 limited events, got %d", len(limited))
	}
}

func TestEventSeverity(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewLogger()
	logger.logDir = filepath.Join(tmpDir, "audit")

	logger.LogAuthEvent("admin", true)
	logger.LogAuthEvent("hacker", false)

	events, _ := logger.Query("auth.login", 0)
	if len(events) != 2 {
		t.Fatalf("expected 2 auth events, got %d", len(events))
	}

	// Successful login should be info
	if events[0].Severity != "info" {
		t.Errorf("successful login should be info, got %s", events[0].Severity)
	}

	// Failed login should be warning
	if events[1].Severity != "warning" {
		t.Errorf("failed login should be warning, got %s", events[1].Severity)
	}
}
