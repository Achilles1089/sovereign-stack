package hardware

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

// GPUTier represents the capability tier of detected GPU
type GPUTier int

const (
	GPUTierNone  GPUTier = iota // No GPU or insufficient VRAM
	GPUTierBasic                // 4-8 GB — small models
	GPUTierMid                  // 8-16 GB — medium models
	GPUTierHigh                 // 16-24 GB — large models
	GPUTierUltra                // 24+ GB — flagship models
	GPUTierApex                 // 64+ GB unified — biggest models
)

// detectGPU populates the GPU fields of a HardwareProfile
func detectGPU(p *config.HardwareProfile) {
	switch runtime.GOOS {
	case "linux":
		if detectNVIDIA(p) {
			return
		}
		if detectAMD(p) {
			return
		}
		detectIntelARC(p)
	case "darwin":
		detectAppleSilicon(p)
	}

	// Fallback: no GPU detected
	if p.GPUType == "" {
		p.GPUType = "none"
		p.GPUName = "CPU only"
		p.GPUMemoryMB = 0
	}
}

// detectNVIDIA checks for NVIDIA GPUs via nvidia-smi
func detectNVIDIA(p *config.HardwareProfile) bool {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return false
	}

	line := strings.TrimSpace(string(out))
	// Handle multi-GPU: take the first one
	lines := strings.Split(line, "\n")
	if len(lines) == 0 {
		return false
	}

	parts := strings.SplitN(lines[0], ",", 2)
	if len(parts) < 2 {
		return false
	}

	p.GPUType = "nvidia"
	p.GPUName = strings.TrimSpace(parts[0])
	mem, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err == nil {
		p.GPUMemoryMB = mem
	}

	return true
}

// detectAMD checks for AMD GPUs via rocm-smi
func detectAMD(p *config.HardwareProfile) bool {
	out, err := exec.Command("rocm-smi", "--showproductname").Output()
	if err != nil {
		return false
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return false
	}

	p.GPUType = "amd"
	// Parse the product name from rocm-smi output
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Card") || strings.Contains(line, "GPU") {
			p.GPUName = strings.TrimSpace(line)
			break
		}
	}
	if p.GPUName == "" {
		p.GPUName = "AMD GPU"
	}

	// Try to get memory
	memOut, err := exec.Command("rocm-smi", "--showmeminfo", "vram").Output()
	if err == nil {
		for _, line := range strings.Split(string(memOut), "\n") {
			if strings.Contains(strings.ToLower(line), "total") {
				fields := strings.Fields(line)
				for _, f := range fields {
					if mb, err := strconv.Atoi(f); err == nil && mb > 100 {
						p.GPUMemoryMB = mb
						break
					}
				}
			}
		}
	}

	return true
}

// detectIntelARC checks for Intel ARC GPUs
func detectIntelARC(p *config.HardwareProfile) bool {
	// Check for Intel GPU via lspci
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(out), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "intel") && (strings.Contains(lower, "arc") || strings.Contains(lower, "xe")) {
			p.GPUType = "intel_arc"
			p.GPUName = strings.TrimSpace(line)
			// Intel ARC A770 = 16GB, A750 = 8GB, A580 = 8GB
			// Rough detection — can be refined later
			if strings.Contains(lower, "a770") {
				p.GPUMemoryMB = 16384
			} else {
				p.GPUMemoryMB = 8192
			}
			return true
		}
	}

	return false
}

// detectAppleSilicon detects Apple Silicon GPU via unified memory
func detectAppleSilicon(p *config.HardwareProfile) {
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return
	}

	cpuName := strings.TrimSpace(string(out))

	// Check if it's Apple Silicon
	if !strings.Contains(cpuName, "Apple") {
		p.GPUType = "none"
		p.GPUName = "Intel Mac (no Metal GPU acceleration for LLMs)"
		return
	}

	p.GPUType = "apple_silicon"
	p.GPUName = cpuName

	// Apple Silicon uses unified memory — GPU memory = total RAM
	// Model routing will use this to determine which models fit
	memOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		bytes, err := strconv.ParseInt(strings.TrimSpace(string(memOut)), 10, 64)
		if err == nil {
			p.GPUMemoryMB = int(bytes / (1024 * 1024))
		}
	}
}

// GetGPUTier returns the capability tier for a GPU
func GetGPUTier(p *config.HardwareProfile) GPUTier {
	memGB := p.GPUMemoryMB / 1024

	if p.GPUType == "none" || memGB < 4 {
		return GPUTierNone
	}
	if memGB >= 64 {
		return GPUTierApex
	}
	if memGB >= 24 {
		return GPUTierUltra
	}
	if memGB >= 16 {
		return GPUTierHigh
	}
	if memGB >= 8 {
		return GPUTierMid
	}
	return GPUTierBasic
}

// RecommendedModel returns the suggested AI model based on GPU tier
func RecommendedModel(p *config.HardwareProfile) string {
	tier := GetGPUTier(p)

	switch tier {
	case GPUTierApex:
		return "qwen2.5:32b"
	case GPUTierUltra:
		return "qwen2.5:32b"
	case GPUTierHigh:
		return "qwen2.5:14b"
	case GPUTierMid:
		return "qwen2.5:7b"
	case GPUTierBasic:
		return "qwen2.5:3b"
	default:
		return "qwen2.5:0.5b"
	}
}

// RecommendedModelDescription returns a human-readable explanation of the recommendation
func RecommendedModelDescription(p *config.HardwareProfile) string {
	model := RecommendedModel(p)
	tier := GetGPUTier(p)

	descriptions := map[GPUTier]string{
		GPUTierApex:  "Flagship model — runs the biggest open models with room to spare",
		GPUTierUltra: "Large model — excellent for coding, analysis, and complex reasoning",
		GPUTierHigh:  "Strong model — great for most tasks including code and writing",
		GPUTierMid:   "Medium model — solid for general use, chat, and basic coding",
		GPUTierBasic: "Compact model — good for chat and simple tasks",
		GPUTierNone:  "Lightweight model — runs on CPU, good for basic Q&A and chat",
	}

	return model + " — " + descriptions[tier]
}
