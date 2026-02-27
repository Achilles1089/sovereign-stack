package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	info := Detect()

	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}

	switch runtime.GOOS {
	case "darwin":
		if info.OS != "darwin" {
			t.Errorf("expected OS 'darwin', got %q", info.OS)
		}
		if info.Mode != "personal" {
			t.Errorf("macOS should default to 'personal' mode, got %q", info.Mode)
		}
	case "linux":
		if info.OS != "linux" {
			t.Errorf("expected OS 'linux', got %q", info.OS)
		}
		// Mode could be "server" or "wsl2"
		if info.Mode != "server" && info.Mode != "wsl2" {
			t.Errorf("Linux should be 'server' or 'wsl2', got %q", info.Mode)
		}
	}
}

func TestIsWSL2(t *testing.T) {
	result := isWSL2()
	// On macOS this should always be false
	if runtime.GOOS == "darwin" && result {
		t.Error("isWSL2 should return false on macOS")
	}
}
