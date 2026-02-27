package ai

import (
	"testing"
)

func TestModelCatalog(t *testing.T) {
	if len(ModelCatalog) == 0 {
		t.Fatal("model catalog should not be empty")
	}

	// Verify each model has required fields
	for _, m := range ModelCatalog {
		if m.Name == "" {
			t.Error("model name should not be empty")
		}
		if m.Description == "" {
			t.Errorf("model %q has no description", m.Name)
		}
		if m.SizeGB <= 0 {
			t.Errorf("model %q has invalid size: %f", m.Name, m.SizeGB)
		}
		if m.MinRAMMB < 0 {
			t.Errorf("model %q has negative MinRAMMB", m.Name)
		}

		validTiers := map[string]bool{
			"cpu": true, "basic": true, "mid": true,
			"high": true, "ultra": true, "apex": true,
		}
		if !validTiers[m.Tier] {
			t.Errorf("model %q has invalid tier: %q", m.Name, m.Tier)
		}
	}
}

func TestGetModelsForTier(t *testing.T) {
	// CPU tier should have at least 1 model
	cpuModels := GetModelsForTier("cpu")
	if len(cpuModels) == 0 {
		t.Error("should have at least one CPU-tier model")
	}

	// Apex tier should have all models
	apexModels := GetModelsForTier("apex")
	if len(apexModels) != len(ModelCatalog) {
		t.Errorf("apex tier should include all %d models, got %d", len(ModelCatalog), len(apexModels))
	}

	// Higher tiers should have more models
	basicModels := GetModelsForTier("basic")
	midModels := GetModelsForTier("mid")
	if len(midModels) < len(basicModels) {
		t.Error("mid tier should have at least as many models as basic")
	}
}

func TestGetModelByName(t *testing.T) {
	m := GetModelByName("qwen2.5:7b")
	if m == nil {
		t.Fatal("should find qwen2.5:7b")
	}
	if m.Tier != "mid" {
		t.Errorf("qwen2.5:7b should be mid tier, got %q", m.Tier)
	}

	m = GetModelByName("nonexistent")
	if m != nil {
		t.Error("should not find nonexistent model")
	}
}
