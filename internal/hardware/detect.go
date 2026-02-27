package hardware

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

// Detect scans the host hardware and returns a HardwareProfile
func Detect() (*config.HardwareProfile, error) {
	profile := &config.HardwareProfile{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "linux":
		detectLinuxCPU(profile)
		detectLinuxRAM(profile)
		detectLinuxDisk(profile)
	case "darwin":
		detectMacOSCPU(profile)
		detectMacOSRAM(profile)
		detectMacOSDisk(profile)
	}

	// GPU detection (cross-platform)
	detectGPU(profile)

	return profile, nil
}

// --- Linux Detection ---

func detectLinuxCPU(p *config.HardwareProfile) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	cores := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				p.CPUModel = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "processor") {
			cores++
		}
	}
	p.CPUCores = cores
}

func detectLinuxRAM(p *config.HardwareProfile) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					p.RAMTotalMB = int(kb / 1024)
				}
			}
			break
		}
	}
}

func detectLinuxDisk(p *config.HardwareProfile) {
	out, err := exec.Command("df", "-BG", "--output=size,avail", "/").Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 2 {
			p.DiskTotalGB = parseGBValue(fields[0])
			p.DiskFreeGB = parseGBValue(fields[1])
		}
	}
}

// --- macOS Detection ---

func detectMacOSCPU(p *config.HardwareProfile) {
	// CPU brand string
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err == nil {
		p.CPUModel = strings.TrimSpace(string(out))
	}

	// Core count
	out, err = exec.Command("sysctl", "-n", "hw.ncpu").Output()
	if err == nil {
		cores, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err == nil {
			p.CPUCores = cores
		}
	}
}

func detectMacOSRAM(p *config.HardwareProfile) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return
	}

	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err == nil {
		p.RAMTotalMB = int(bytes / (1024 * 1024))
	}
}

func detectMacOSDisk(p *config.HardwareProfile) {
	out, err := exec.Command("df", "-g", "/").Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			p.DiskTotalGB, _ = strconv.Atoi(fields[1])
			p.DiskFreeGB, _ = strconv.Atoi(fields[3])
		}
	}
}

// --- Helpers ---

func parseGBValue(s string) int {
	s = strings.TrimSuffix(s, "G")
	val, _ := strconv.Atoi(s)
	return val
}

// Summary returns a human-readable summary of the hardware
func Summary(p *config.HardwareProfile) string {
	gpu := "None"
	if p.GPUType != "none" && p.GPUType != "" {
		gpu = fmt.Sprintf("%s (%s, %d MB)", p.GPUName, p.GPUType, p.GPUMemoryMB)
	}

	return fmt.Sprintf(`Hardware Detected:
  CPU:    %s (%d cores)
  RAM:    %d MB (%.1f GB)
  Disk:   %d GB total, %d GB free
  GPU:    %s
  Arch:   %s/%s`,
		p.CPUModel, p.CPUCores,
		p.RAMTotalMB, float64(p.RAMTotalMB)/1024,
		p.DiskTotalGB, p.DiskFreeGB,
		gpu,
		p.OS, p.Arch,
	)
}
