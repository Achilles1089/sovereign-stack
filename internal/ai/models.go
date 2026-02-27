package ai

// ModelCatalog defines the curated model catalog with recommendations
var ModelCatalog = []ModelEntry{
	// Lightweight (CPU-only)
	{Name: "qwen2.5:0.5b", DisplayName: "Qwen 2.5 0.5B", SizeGB: 0.4, MinRAMMB: 2048, Tier: "cpu", Description: "Tiny model for basic Q&A, runs on anything"},
	{Name: "phi3:mini", DisplayName: "Phi-3 Mini", SizeGB: 2.3, MinRAMMB: 4096, Tier: "cpu", Description: "Microsoft's compact model, good reasoning"},

	// Basic (4-8GB GPU)
	{Name: "qwen2.5:3b", DisplayName: "Qwen 2.5 3B", SizeGB: 2.0, MinRAMMB: 4096, Tier: "basic", Description: "Solid for chat and simple coding"},
	{Name: "llama3.2:3b", DisplayName: "Llama 3.2 3B", SizeGB: 2.0, MinRAMMB: 4096, Tier: "basic", Description: "Meta's compact model, versatile"},

	// Mid (8-16GB GPU)
	{Name: "qwen2.5:7b", DisplayName: "Qwen 2.5 7B", SizeGB: 4.7, MinRAMMB: 8192, Tier: "mid", Description: "Great all-around model for most tasks"},
	{Name: "llama3.2:8b", DisplayName: "Llama 3.2 8B", SizeGB: 4.9, MinRAMMB: 8192, Tier: "mid", Description: "Meta's balanced model, strong coding"},
	{Name: "deepseek-r1:7b", DisplayName: "DeepSeek R1 7B", SizeGB: 4.7, MinRAMMB: 8192, Tier: "mid", Description: "Deep reasoning and chain-of-thought"},

	// High (16-24GB GPU)
	{Name: "qwen2.5:14b", DisplayName: "Qwen 2.5 14B", SizeGB: 9.0, MinRAMMB: 16384, Tier: "high", Description: "Strong for coding, analysis, and writing"},
	{Name: "deepseek-r1:14b", DisplayName: "DeepSeek R1 14B", SizeGB: 9.0, MinRAMMB: 16384, Tier: "high", Description: "Excellent reasoning capabilities"},

	// Ultra (24GB+ GPU)
	{Name: "qwen2.5:32b", DisplayName: "Qwen 2.5 32B", SizeGB: 20.0, MinRAMMB: 24576, Tier: "ultra", Description: "Near-frontier performance locally"},
	{Name: "deepseek-r1:32b", DisplayName: "DeepSeek R1 32B", SizeGB: 20.0, MinRAMMB: 24576, Tier: "ultra", Description: "Top-tier local reasoning"},
	{Name: "llama3.1:70b", DisplayName: "Llama 3.1 70B", SizeGB: 40.0, MinRAMMB: 49152, Tier: "apex", Description: "Massive model, needs 48GB+ VRAM"},
}

// ModelEntry represents a model in the curated catalog
type ModelEntry struct {
	Name        string  `yaml:"name"`
	DisplayName string  `yaml:"display_name"`
	SizeGB      float64 `yaml:"size_gb"`
	MinRAMMB    int     `yaml:"min_ram_mb"`
	Tier        string  `yaml:"tier"` // cpu, basic, mid, high, ultra, apex
	Description string  `yaml:"description"`
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
	return nil
}
