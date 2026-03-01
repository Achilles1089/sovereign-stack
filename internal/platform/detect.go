package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Platform represents the detected operating system platform
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformMacOS   Platform = "darwin"
	PlatformWSL2    Platform = "wsl2"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

// Mode represents how Sovereign Stack should behave on this platform
type Mode string

const (
	ModeServer   Mode = "server"   // Full headless server (Linux)
	ModePersonal Mode = "personal" // Development/personal use (macOS)
	ModeWSL2     Mode = "wsl2"     // Windows Subsystem for Linux
)

// Info holds all platform detection results
type Info struct {
	Platform Platform
	Mode     Mode
	OS       string // "linux", "darwin", "windows"
	Arch     string // "amd64", "arm64"
	IsRoot   bool
	Distro   string // Linux distribution name (e.g., "ubuntu", "debian")
	Version  string // OS version
}

// Detect performs full platform detection
func Detect() *Info {
	info := &Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "linux":
		if isWSL2() {
			info.Platform = PlatformWSL2
			info.Mode = ModeWSL2
		} else {
			info.Platform = PlatformLinux
			info.Mode = ModeServer
		}
		info.Distro = detectLinuxDistro()
		info.Version = detectLinuxVersion()
	case "darwin":
		info.Platform = PlatformMacOS
		info.Mode = ModePersonal
	case "windows":
		info.Platform = PlatformWindows
		info.Mode = ModePersonal
	default:
		info.Platform = PlatformUnknown
		info.Mode = ModePersonal
	}

	info.IsRoot = os.Geteuid() == 0

	return info
}

// isWSL2 checks if we're running inside Windows Subsystem for Linux
func isWSL2() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

// detectLinuxDistro returns the Linux distribution name
func detectLinuxDistro() string {
	// Try /etc/os-release first (modern standard)
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}

	// Fallback to lsb_release
	out, err := exec.Command("lsb_release", "-is").Output()
	if err == nil {
		return strings.TrimSpace(strings.ToLower(string(out)))
	}

	return "unknown"
}

// detectLinuxVersion returns the Linux version string
func detectLinuxVersion() string {
	data, err := os.ReadFile("/etc/os-release")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VERSION_ID=") {
				return strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}
	return "unknown"
}

// SupportsGPUPassthrough returns true if the platform supports Docker GPU passthrough
func (i *Info) SupportsGPUPassthrough() bool {
	return i.Platform == PlatformLinux // Only native Linux supports Docker GPU passthrough
}

// NeedsDockerDesktop returns true if Docker Desktop is required (vs Docker Engine)
func (i *Info) NeedsDockerDesktop() bool {
	return i.Platform == PlatformMacOS || i.Platform == PlatformWindows
}

// String returns a human-readable platform description
func (i *Info) String() string {
	switch i.Platform {
	case PlatformLinux:
		if i.Distro != "unknown" {
			return strings.Title(i.Distro) + " Linux " + i.Version + " (" + i.Arch + ")"
		}
		return "Linux (" + i.Arch + ")"
	case PlatformMacOS:
		return "macOS (" + i.Arch + ")"
	case PlatformWSL2:
		return "WSL2 / " + strings.Title(i.Distro) + " (" + i.Arch + ")"
	default:
		return i.OS + " (" + i.Arch + ")"
	}
}
