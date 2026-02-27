package hardware

import (
	"runtime"
	"testing"

	"github.com/Achilles1089/sovereign-stack/internal/config"
)

func TestDetect(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Should always detect something
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS mismatch: got %q, expected %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch mismatch: got %q, expected %q", info.Arch, runtime.GOARCH)
	}

	// CPU should be detected
	if info.CPUCores < 1 {
		t.Error("CPU cores should be at least 1")
	}
	if info.CPUModel == "" {
		t.Error("CPU model should not be empty")
	}

	// RAM should be detected
	if info.RAMTotalMB < 1 {
		t.Error("RAM should be at least 1 MB")
	}

	// Disk should be detected
	if info.DiskTotalGB < 1 {
		t.Error("Disk should be at least 1 GB")
	}
	if info.DiskFreeGB < 0 {
		t.Error("Free disk should not be negative")
	}
}

func TestDetectGPU(t *testing.T) {
	info, err := Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// GPU type should always be set (even if "none")
	validTypes := map[string]bool{
		"nvidia": true, "amd": true, "apple_silicon": true,
		"intel_arc": true, "none": true,
	}
	if !validTypes[info.GPUType] {
		t.Errorf("unexpected GPU type: %q", info.GPUType)
	}

	// On macOS with Apple Silicon, GPU should be detected
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if info.GPUType != "apple_silicon" {
			t.Errorf("expected apple_silicon GPU on M-series Mac, got %q", info.GPUType)
		}
		if info.GPUMemoryMB < 1 {
			t.Error("Apple Silicon GPU memory should be > 0")
		}
	}
}

func TestGetGPUTier(t *testing.T) {
	tests := []struct {
		name     string
		gpuType  string
		gpuMem   int
		expected GPUTier
	}{
		{"no GPU", "none", 0, GPUTierNone},
		{"basic NVIDIA", "nvidia", 4096, GPUTierBasic},
		{"mid NVIDIA", "nvidia", 12288, GPUTierMid},
		{"high NVIDIA", "nvidia", 20480, GPUTierHigh},
		{"ultra NVIDIA", "nvidia", 49152, GPUTierUltra},
		{"apex NVIDIA", "nvidia", 81920, GPUTierApex},
		{"Apple 8GB", "apple_silicon", 8192, GPUTierMid},
		{"Apple 16GB", "apple_silicon", 16384, GPUTierHigh},
		{"Apple 32GB", "apple_silicon", 32768, GPUTierUltra},
		{"Apple 64GB", "apple_silicon", 65536, GPUTierApex},
		{"Apple 128GB", "apple_silicon", 131072, GPUTierApex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hw := &config.HardwareProfile{
				GPUType:     tt.gpuType,
				GPUMemoryMB: tt.gpuMem,
			}
			tier := GetGPUTier(hw)
			if tier != tt.expected {
				t.Errorf("expected tier %d, got %d", tt.expected, tier)
			}
		})
	}
}

func TestRecommendedModel(t *testing.T) {
	// CPU-only should get smallest model
	hw := &config.HardwareProfile{GPUType: "none", GPUMemoryMB: 0}
	model := RecommendedModel(hw)
	if model == "" {
		t.Error("should always recommend a model")
	}
	if model != "qwen2.5:0.5b" {
		t.Errorf("CPU-only should recommend qwen2.5:0.5b, got %q", model)
	}

	// High-end Apple Silicon should get a bigger model
	hw = &config.HardwareProfile{GPUType: "apple_silicon", GPUMemoryMB: 131072}
	model = RecommendedModel(hw)
	if model == "qwen2.5:0.5b" {
		t.Error("128GB Apple Silicon should not recommend the smallest model")
	}
}
