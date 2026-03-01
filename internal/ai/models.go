package ai

import "strings"

// ModelCatalog defines the curated GGUF model catalog with download URLs
var ModelCatalog = []ModelEntry{
	// RWKV-7 "Goose" — RNN architecture, O(1) memory, beats LLaMA 3.2
	{Name: "rwkv7-2.9B", DisplayName: "RWKV-7 2.9B ★", Filename: "rwkv7-2.9B-world-q4_k_m.gguf",
		SizeGB: 1.88, MinRAMMB: 3072, Tier: "cpu", Architecture: "rwkv",
		Description: "Flagship RNN model — beats LLaMA 3.2, O(1) memory",
		URL:         "https://huggingface.co/Mungert/rwkv7-2.9B-world-GGUF/resolve/main/rwkv7-2.9B-world-q4_k_m.gguf"},
	{Name: "rwkv7-1.5B", DisplayName: "RWKV-7 1.5B", Filename: "rwkv7-1.5B-world-q4_k_m.gguf",
		SizeGB: 0.95, MinRAMMB: 2048, Tier: "cpu", Architecture: "rwkv",
		Description: "Fast RNN model, good for chat",
		URL:         "https://huggingface.co/Mungert/rwkv7-1.5B-world-GGUF/resolve/main/rwkv7-1.5B-world-q4_k_m.gguf"},
	{Name: "rwkv7-0.4B", DisplayName: "RWKV-7 0.4B", Filename: "rwkv7-0.4B-world-q8_0.gguf",
		SizeGB: 0.5, MinRAMMB: 1024, Tier: "cpu", Architecture: "rwkv",
		Description: "Tiny RNN — ultra fast, runs on anything",
		URL:         "https://huggingface.co/Mungert/rwkv7-0.4B-world-GGUF/resolve/main/rwkv7-0.4B-world-q8_0.gguf"},

	// Transformer models (GGUF format)
	{Name: "smollm2-360m", DisplayName: "SmolLM2 360M", Filename: "smollm2-360m-f16.gguf",
		SizeGB: 0.7, MinRAMMB: 1024, Tier: "cpu", Architecture: "llama",
		Description: "Ultra-light transformer for basic Q&A",
		URL:         "https://huggingface.co/bartowski/SmolLM2-360M-Instruct-GGUF/resolve/main/SmolLM2-360M-Instruct-f16.gguf"},
	{Name: "qwen2.5-0.5b", DisplayName: "Qwen 2.5 0.5B", Filename: "qwen2.5-0.5b-instruct-q4_k_m.gguf",
		SizeGB: 0.4, MinRAMMB: 1024, Tier: "cpu", Architecture: "qwen2",
		Description: "Tiny but capable Qwen model",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"},
	{Name: "qwen2.5-1.5b", DisplayName: "Qwen 2.5 1.5B", Filename: "qwen2.5-1.5b-instruct-q4_k_m.gguf",
		SizeGB: 1.1, MinRAMMB: 2048, Tier: "cpu", Architecture: "qwen2",
		Description: "Good all-around model for chat and coding",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf"},
	{Name: "qwen2.5-3b", DisplayName: "Qwen 2.5 3B", Filename: "qwen2.5-3b-instruct-q4_k_m.gguf",
		SizeGB: 2.0, MinRAMMB: 4096, Tier: "basic", Architecture: "qwen2",
		Description: "Strong for chat, coding, and analysis",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf"},
	{Name: "phi-3-mini", DisplayName: "Phi-3 Mini", Filename: "Phi-3-mini-4k-instruct-q4.gguf",
		SizeGB: 2.3, MinRAMMB: 4096, Tier: "basic", Architecture: "phi3",
		Description: "Microsoft's compact reasoning model",
		URL:         "https://huggingface.co/microsoft/Phi-3-mini-4k-instruct-gguf/resolve/main/Phi-3-mini-4k-instruct-q4.gguf"},
}

// ModelEntry represents a model in the curated catalog
type ModelEntry struct {
	Name         string  `json:"name"`
	DisplayName  string  `json:"display_name"`
	Filename     string  `json:"filename"`
	SizeGB       float64 `json:"size_gb"`
	MinRAMMB     int     `json:"min_ram_mb"`
	Tier         string  `json:"tier"` // cpu, basic, mid, high, ultra, apex
	Architecture string  `json:"architecture"`
	Description  string  `json:"description"`
	URL          string  `json:"url"`
}

// GetModelsForTier returns all models that can run on a given hardware tier
func GetModelsForTier(tier string) []ModelEntry {
	tierOrder := map[string]int{
		"cpu": 0, "basic": 1, "mid": 2, "high": 3, "ultra": 4, "apex": 5,
	}

	maxTier := tierOrder[tier]
	var models []ModelEntry

	for _, m := range ModelCatalog {
		if tierOrder[m.Tier] <= maxTier {
			models = append(models, m)
		}
	}

	return models
}

// GetModelByName finds a model in the catalog
func GetModelByName(name string) *ModelEntry {
	for _, m := range ModelCatalog {
		if m.Name == name {
			return &m
		}
	}
	// Also try matching by filename without extension
	for _, m := range ModelCatalog {
		if strings.TrimSuffix(m.Filename, ".gguf") == name {
			return &m
		}
	}
	return nil
}
