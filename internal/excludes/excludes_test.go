package excludes

import (
	"testing"
)

func TestLibraryNotEmpty(t *testing.T) {
	if len(Library) == 0 {
		t.Fatal("Library should not be empty")
	}
}

func TestLibraryEntriesHaveRequiredFields(t *testing.T) {
	for i, p := range Library {
		if p.Name == "" {
			t.Errorf("Library[%d] has empty Name", i)
		}
		if p.Description == "" {
			t.Errorf("Library[%d] (%s) has empty Description", i, p.Name)
		}
		if p.Category == "" {
			t.Errorf("Library[%d] (%s) has empty Category", i, p.Name)
		}
		if len(p.Patterns) == 0 {
			t.Errorf("Library[%d] (%s) has no Patterns", i, p.Name)
		}
	}
}

func TestLibraryPatternsNotEmpty(t *testing.T) {
	for _, p := range Library {
		for j, pattern := range p.Patterns {
			if pattern == "" {
				t.Errorf("Library entry %q has empty pattern at index %d", p.Name, j)
			}
		}
	}
}

func TestLibraryCategoriesAreValid(t *testing.T) {
	validCategories := map[Category]bool{
		CategoryOS:        true,
		CategoryIDE:       true,
		CategoryLanguage:  true,
		CategoryBuild:     true,
		CategoryCache:     true,
		CategoryTemp:      true,
		CategoryLogs:      true,
		CategorySecurity:  true,
		CategoryDatabase:  true,
		CategoryContainer: true,
	}

	for _, p := range Library {
		if !validCategories[p.Category] {
			t.Errorf("Library entry %q has invalid category %q", p.Name, p.Category)
		}
	}
}

func TestLibraryContainsExpectedEntries(t *testing.T) {
	expected := []string{
		"macOS", "Windows", "Linux",
		"Visual Studio Code", "JetBrains IDEs",
		"Node.js", "Python", "Go", "Java",
		"Security & Secrets",
		"Docker",
	}

	names := make(map[string]bool)
	for _, p := range Library {
		names[p.Name] = true
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("Library missing expected entry %q", name)
		}
	}
}

func TestCategoriesMapNotEmpty(t *testing.T) {
	if len(Categories) == 0 {
		t.Fatal("Categories map should not be empty")
	}
}

func TestCategoriesMapHasAllCategories(t *testing.T) {
	allCategories := GetAllCategories()
	for _, cat := range allCategories {
		info, ok := Categories[cat]
		if !ok {
			t.Errorf("Categories map missing entry for %q", cat)
			continue
		}
		if info.Name == "" {
			t.Errorf("Categories[%q] has empty Name", cat)
		}
		if info.Description == "" {
			t.Errorf("Categories[%q] has empty Description", cat)
		}
		if info.Icon == "" {
			t.Errorf("Categories[%q] has empty Icon", cat)
		}
	}
}

func TestCategoriesMapHasNoExtraEntries(t *testing.T) {
	allCategories := make(map[Category]bool)
	for _, cat := range GetAllCategories() {
		allCategories[cat] = true
	}

	for cat := range Categories {
		if !allCategories[cat] {
			t.Errorf("Categories map has extra entry %q not in GetAllCategories()", cat)
		}
	}
}

func TestGetAllCategories(t *testing.T) {
	categories := GetAllCategories()

	if len(categories) != 10 {
		t.Fatalf("GetAllCategories() returned %d categories, want 10", len(categories))
	}

	expected := []Category{
		CategoryOS,
		CategoryIDE,
		CategoryLanguage,
		CategoryBuild,
		CategoryCache,
		CategoryTemp,
		CategoryLogs,
		CategorySecurity,
		CategoryDatabase,
		CategoryContainer,
	}

	for i, cat := range categories {
		if cat != expected[i] {
			t.Errorf("GetAllCategories()[%d] = %q, want %q", i, cat, expected[i])
		}
	}
}

func TestGetAllCategoriesNoDuplicates(t *testing.T) {
	categories := GetAllCategories()
	seen := make(map[Category]bool)
	for _, cat := range categories {
		if seen[cat] {
			t.Errorf("GetAllCategories() has duplicate category %q", cat)
		}
		seen[cat] = true
	}
}

func TestCategoryConstants(t *testing.T) {
	tests := []struct {
		category Category
		value    string
	}{
		{CategoryOS, "os"},
		{CategoryIDE, "ide"},
		{CategoryLanguage, "language"},
		{CategoryBuild, "build"},
		{CategoryCache, "cache"},
		{CategoryTemp, "temp"},
		{CategoryLogs, "logs"},
		{CategorySecurity, "security"},
		{CategoryDatabase, "database"},
		{CategoryContainer, "container"},
	}

	for _, tt := range tests {
		if string(tt.category) != tt.value {
			t.Errorf("Category constant %q = %q, want %q", tt.value, string(tt.category), tt.value)
		}
	}
}

func TestPatternTypes(t *testing.T) {
	// Verify the library contains various pattern types
	hasWildcard := false
	hasDirectory := false
	hasNegation := false
	hasSimpleFile := false

	for _, p := range Library {
		for _, pattern := range p.Patterns {
			if len(pattern) > 0 && pattern[0] == '!' {
				hasNegation = true
			}
			if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
				hasDirectory = true
			}
			if contains(pattern, '*') {
				hasWildcard = true
			}
			if pattern == ".DS_Store" || pattern == "Thumbs.db" {
				hasSimpleFile = true
			}
		}
	}

	if !hasWildcard {
		t.Error("Library should contain wildcard patterns (e.g. *.log)")
	}
	if !hasDirectory {
		t.Error("Library should contain directory patterns (e.g. node_modules/)")
	}
	if !hasNegation {
		t.Error("Library should contain negation patterns (e.g. !.vscode/settings.json)")
	}
	if !hasSimpleFile {
		t.Error("Library should contain simple file patterns (e.g. .DS_Store)")
	}
}

func TestLibraryNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range Library {
		if seen[p.Name] {
			t.Errorf("Library has duplicate name %q", p.Name)
		}
		seen[p.Name] = true
	}
}

func TestCategoryInfoFields(t *testing.T) {
	tests := []struct {
		category Category
		name     string
		icon     string
	}{
		{CategoryOS, "Operating System", "computer"},
		{CategoryIDE, "IDE & Editors", "code"},
		{CategoryLanguage, "Languages", "language"},
		{CategoryBuild, "Build Artifacts", "build"},
		{CategoryCache, "Caches", "database"},
		{CategoryTemp, "Temporary Files", "clock"},
		{CategoryLogs, "Logs", "file-text"},
		{CategorySecurity, "Security & Secrets", "shield"},
		{CategoryDatabase, "Databases", "database"},
		{CategoryContainer, "Containers & VMs", "box"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			info, ok := Categories[tt.category]
			if !ok {
				t.Fatalf("Categories[%q] not found", tt.category)
			}
			if info.Name != tt.name {
				t.Errorf("Categories[%q].Name = %q, want %q", tt.category, info.Name, tt.name)
			}
			if info.Icon != tt.icon {
				t.Errorf("Categories[%q].Icon = %q, want %q", tt.category, info.Icon, tt.icon)
			}
		})
	}
}


func contains(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}
