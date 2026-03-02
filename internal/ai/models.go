package ai

import "strings"

// ModelCatalog defines the curated GGUF model catalog with download URLs
// All models chosen to fit within 6GB phone RAM with KV cache overhead
var ModelCatalog = []ModelEntry{
	// === RWKV-7 "Goose" — RNN architecture, O(1) memory ===
	{Name: "rwkv7-2.9B", DisplayName: "RWKV-7 2.9B ★", Filename: "rwkv7-2.9B-world-q4_k_m.gguf",
		SizeGB: 1.88, MinRAMMB: 3072, Tier: "cpu", Architecture: "rwkv",
		Description: "Flagship RNN — beats LLaMA 3.2, constant memory",
		URL:         "https://huggingface.co/Mungert/rwkv7-2.9B-world-GGUF/resolve/main/rwkv7-2.9B-world-q4_k_m.gguf"},
	{Name: "rwkv7-2.9B-q8", DisplayName: "RWKV-7 2.9B HQ", Filename: "rwkv7-2.9B-world-q8_0.gguf",
		SizeGB: 3.1, MinRAMMB: 4096, Tier: "basic", Architecture: "rwkv",
		Description: "High-quality RWKV — sharper output, same speed",
		URL:         "https://huggingface.co/Mungert/rwkv7-2.9B-world-GGUF/resolve/main/rwkv7-2.9B-world-q8_0.gguf"},
	{Name: "rwkv7-1.5B", DisplayName: "RWKV-7 1.5B", Filename: "rwkv7-1.5B-world-q4_k_m.gguf",
		SizeGB: 0.95, MinRAMMB: 2048, Tier: "cpu", Architecture: "rwkv",
		Description: "Fast RNN — good chat quality, very efficient",
		URL:         "https://huggingface.co/Mungert/rwkv7-1.5B-world-GGUF/resolve/main/rwkv7-1.5B-world-q4_k_m.gguf"},

	// === Qwen 2.5 — Strong transformer family ===
	{Name: "qwen2.5-7b-q4", DisplayName: "Qwen 2.5 7B HQ", Filename: "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
		SizeGB: 4.3, MinRAMMB: 5632, Tier: "basic", Architecture: "qwen2",
		Description: "Best quality — tight fit, may need reduced context",
		URL:         "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/Qwen2.5-7B-Instruct-Q4_K_M.gguf"},
	{Name: "qwen2.5-7b", DisplayName: "Qwen 2.5 7B", Filename: "qwen2.5-7b-instruct-q3_k_m.gguf",
		SizeGB: 3.3, MinRAMMB: 5120, Tier: "basic", Architecture: "qwen2",
		Description: "Largest comfortable fit — excellent quality",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-7B-Instruct-GGUF/resolve/main/qwen2.5-7b-instruct-q3_k_m.gguf"},
	{Name: "qwen2.5-3b", DisplayName: "Qwen 2.5 3B", Filename: "qwen2.5-3b-instruct-q4_k_m.gguf",
		SizeGB: 2.0, MinRAMMB: 4096, Tier: "basic", Architecture: "qwen2",
		Description: "Strong all-rounder — chat, coding, analysis",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf"},
	{Name: "qwen2.5-1.5b", DisplayName: "Qwen 2.5 1.5B", Filename: "qwen2.5-1.5b-instruct-q4_k_m.gguf",
		SizeGB: 1.1, MinRAMMB: 2048, Tier: "cpu", Architecture: "qwen2",
		Description: "Fast and capable — great speed/quality ratio",
		URL:         "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf"},

	// === Other architectures ===
	{Name: "llama-3.2-3b", DisplayName: "Llama 3.2 3B", Filename: "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		SizeGB: 2.0, MinRAMMB: 4096, Tier: "basic", Architecture: "llama",
		Description: "Meta's latest small model — solid general purpose",
		URL:         "https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/Llama-3.2-3B-Instruct-Q4_K_M.gguf"},
	{Name: "phi-3-mini", DisplayName: "Phi-3 Mini", Filename: "Phi-3-mini-4k-instruct-q4.gguf",
		SizeGB: 2.3, MinRAMMB: 4096, Tier: "basic", Architecture: "phi3",
		Description: "Microsoft's reasoning model — strong at logic",
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
