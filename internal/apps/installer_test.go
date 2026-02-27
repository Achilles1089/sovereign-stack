package apps

import (
	"testing"
)

func TestBuiltinApps(t *testing.T) {
	if len(BuiltinApps) == 0 {
		t.Fatal("BuiltinApps should not be empty")
	}

	names := make(map[string]bool)
	for _, app := range BuiltinApps {
		// Required fields
		if app.Name == "" {
			t.Error("app name should not be empty")
		}
		if app.DisplayName == "" {
			t.Errorf("app %q has no display name", app.Name)
		}
		if app.Description == "" {
			t.Errorf("app %q has no description", app.Name)
		}
		if app.Category == "" {
			t.Errorf("app %q has no category", app.Name)
		}
		if app.Compose.Image == "" {
			t.Errorf("app %q has no Docker image", app.Name)
		}
		if app.Version == "" {
			t.Errorf("app %q has no version", app.Name)
		}

		// Unique names
		if names[app.Name] {
			t.Errorf("duplicate app name: %q", app.Name)
		}
		names[app.Name] = true
	}
}

func TestFindApp(t *testing.T) {
	// Should find nextcloud
	app := FindApp("nextcloud")
	if app == nil {
		t.Fatal("should find nextcloud")
	}
	if app.Category != "productivity" {
		t.Errorf("nextcloud should be 'productivity', got %q", app.Category)
	}

	// Should not find nonexistent
	app = FindApp("does-not-exist")
	if app != nil {
		t.Error("should not find nonexistent app")
	}
}

func TestAppCount(t *testing.T) {
	// We should have 12 apps in the MVP catalog
	if len(BuiltinApps) < 12 {
		t.Errorf("expected at least 12 apps, got %d", len(BuiltinApps))
	}
}

func TestAppCategories(t *testing.T) {
	categories := make(map[string]int)
	for _, app := range BuiltinApps {
		categories[app.Category]++
	}

	// Should have multiple categories
	if len(categories) < 3 {
		t.Errorf("expected at least 3 categories, got %d", len(categories))
	}
}
