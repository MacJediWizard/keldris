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

func TestGetPatternsByCategory(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		wantMin  int
	}{
		{"OS patterns", CategoryOS, 3},
		{"IDE patterns", CategoryIDE, 3},
		{"Language patterns", CategoryLanguage, 5},
		{"Build patterns", CategoryBuild, 1},
		{"Cache patterns", CategoryCache, 2},
		{"Temp patterns", CategoryTemp, 1},
		{"Logs patterns", CategoryLogs, 1},
		{"Security patterns", CategorySecurity, 1},
		{"Database patterns", CategoryDatabase, 1},
		{"Container patterns", CategoryContainer, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := GetPatternsByCategory(tt.category)
			if len(patterns) < tt.wantMin {
				t.Errorf("GetPatternsByCategory(%q) returned %d patterns, want at least %d",
					tt.category, len(patterns), tt.wantMin)
			}
			for _, p := range patterns {
				if p.Category != tt.category {
					t.Errorf("GetPatternsByCategory(%q) returned pattern %q with category %q",
						tt.category, p.Name, p.Category)
				}
			}
		})
	}
}

func TestGetPatternsByCategory_InvalidCategory(t *testing.T) {
	patterns := GetPatternsByCategory("nonexistent")
	if patterns != nil {
		t.Errorf("GetPatternsByCategory(nonexistent) = %v, want nil", patterns)
	}
}

func TestGetPatternsByCategory_EmptyCategory(t *testing.T) {
	patterns := GetPatternsByCategory("")
	if patterns != nil {
		t.Errorf("GetPatternsByCategory(\"\") = %v, want nil", patterns)
	}
}

func TestGetPatternsByCategory_CoversAllLibraryEntries(t *testing.T) {
	totalFromCategories := 0
	for _, cat := range GetAllCategories() {
		totalFromCategories += len(GetPatternsByCategory(cat))
	}

	if totalFromCategories != len(Library) {
		t.Errorf("sum of GetPatternsByCategory() across all categories = %d, want %d (Library size)",
			totalFromCategories, len(Library))
	}
}

func TestGetPatternsByCategory_OSContainsExpected(t *testing.T) {
	patterns := GetPatternsByCategory(CategoryOS)
	names := make(map[string]bool)
	for _, p := range patterns {
		names[p.Name] = true
	}

	for _, name := range []string{"macOS", "Windows", "Linux"} {
		if !names[name] {
			t.Errorf("OS category missing expected pattern group %q", name)
		}
	}
}

func TestGetPatternsByCategory_SecurityContainsExpected(t *testing.T) {
	patterns := GetPatternsByCategory(CategorySecurity)
	if len(patterns) == 0 {
		t.Fatal("Security category should have patterns")
	}

	found := false
	for _, p := range patterns {
		for _, pattern := range p.Patterns {
			if pattern == ".env" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Security patterns should include .env")
	}
}

func TestFlattenPatterns(t *testing.T) {
	input := []BuiltInPattern{
		{
			Name:     "test1",
			Category: CategoryOS,
			Patterns: []string{"*.log", "*.tmp"},
		},
		{
			Name:     "test2",
			Category: CategoryTemp,
			Patterns: []string{"*.bak", "*.old"},
		},
	}

	result := FlattenPatterns(input)
	if len(result) != 4 {
		t.Fatalf("FlattenPatterns() returned %d patterns, want 4", len(result))
	}

	expected := []string{"*.log", "*.tmp", "*.bak", "*.old"}
	for i, pattern := range result {
		if pattern != expected[i] {
			t.Errorf("FlattenPatterns()[%d] = %q, want %q", i, pattern, expected[i])
		}
	}
}

func TestFlattenPatterns_Deduplication(t *testing.T) {
	input := []BuiltInPattern{
		{
			Name:     "test1",
			Patterns: []string{"*.log", "*.tmp", "build/"},
		},
		{
			Name:     "test2",
			Patterns: []string{"*.tmp", "build/", "*.bak"},
		},
	}

	result := FlattenPatterns(input)
	if len(result) != 4 {
		t.Fatalf("FlattenPatterns() with duplicates returned %d patterns, want 4", len(result))
	}

	expected := []string{"*.log", "*.tmp", "build/", "*.bak"}
	for i, pattern := range result {
		if pattern != expected[i] {
			t.Errorf("FlattenPatterns()[%d] = %q, want %q", i, pattern, expected[i])
		}
	}
}

func TestFlattenPatterns_PreservesOrder(t *testing.T) {
	input := []BuiltInPattern{
		{
			Name:     "group1",
			Patterns: []string{"c", "a", "b"},
		},
		{
			Name:     "group2",
			Patterns: []string{"d", "a"},
		},
	}

	result := FlattenPatterns(input)
	expected := []string{"c", "a", "b", "d"}
	if len(result) != len(expected) {
		t.Fatalf("FlattenPatterns() returned %d patterns, want %d", len(result), len(expected))
	}
	for i, pattern := range result {
		if pattern != expected[i] {
			t.Errorf("FlattenPatterns()[%d] = %q, want %q", i, pattern, expected[i])
		}
	}
}

func TestFlattenPatterns_EmptyInput(t *testing.T) {
	result := FlattenPatterns(nil)
	if result != nil {
		t.Errorf("FlattenPatterns(nil) = %v, want nil", result)
	}

	result = FlattenPatterns([]BuiltInPattern{})
	if result != nil {
		t.Errorf("FlattenPatterns([]) = %v, want nil", result)
	}
}

func TestFlattenPatterns_EmptyPatterns(t *testing.T) {
	input := []BuiltInPattern{
		{
			Name:     "empty",
			Patterns: []string{},
		},
	}
	result := FlattenPatterns(input)
	if result != nil {
		t.Errorf("FlattenPatterns(empty patterns) = %v, want nil", result)
	}
}

func TestFlattenPatterns_SinglePattern(t *testing.T) {
	input := []BuiltInPattern{
		{
			Name:     "single",
			Patterns: []string{"*.log"},
		},
	}
	result := FlattenPatterns(input)
	if len(result) != 1 || result[0] != "*.log" {
		t.Errorf("FlattenPatterns(single) = %v, want [*.log]", result)
	}
}

func TestFlattenPatterns_AllDuplicates(t *testing.T) {
	input := []BuiltInPattern{
		{Name: "a", Patterns: []string{"*.log", "*.tmp"}},
		{Name: "b", Patterns: []string{"*.log", "*.tmp"}},
		{Name: "c", Patterns: []string{"*.log", "*.tmp"}},
	}
	result := FlattenPatterns(input)
	if len(result) != 2 {
		t.Errorf("FlattenPatterns(all duplicates) returned %d patterns, want 2", len(result))
	}
}

func TestFlattenPatterns_WithLibraryData(t *testing.T) {
	osPatterns := GetPatternsByCategory(CategoryOS)
	result := FlattenPatterns(osPatterns)
	if len(result) == 0 {
		t.Fatal("FlattenPatterns(OS patterns) should return patterns")
	}

	// Verify deduplication: result should have no more patterns than total across groups
	total := 0
	for _, p := range osPatterns {
		total += len(p.Patterns)
	}
	if len(result) > total {
		t.Errorf("FlattenPatterns produced more patterns (%d) than input total (%d)", len(result), total)
	}
}

func TestFlattenPatterns_FullLibrary(t *testing.T) {
	result := FlattenPatterns(Library)
	if len(result) == 0 {
		t.Fatal("FlattenPatterns(Library) should return patterns")
	}

	// Count total patterns across library
	total := 0
	for _, p := range Library {
		total += len(p.Patterns)
	}

	// With deduplication, result should be <= total
	if len(result) > total {
		t.Errorf("FlattenPatterns(Library) produced %d patterns, but total is %d", len(result), total)
	}
	// With known duplicates (e.g. *~ appears in multiple groups), result should be < total
	if len(result) >= total {
		t.Logf("FlattenPatterns(Library): %d unique out of %d total", len(result), total)
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

func TestSecurityPatternsComprehensive(t *testing.T) {
	securityPatterns := GetPatternsByCategory(CategorySecurity)
	allPatterns := FlattenPatterns(securityPatterns)

	critical := []string{".env", "*.pem", "*.key", "credentials.json", "secrets.json"}
	patternSet := make(map[string]bool)
	for _, p := range allPatterns {
		patternSet[p] = true
	}

	for _, c := range critical {
		if !patternSet[c] {
			t.Errorf("Security patterns missing critical pattern %q", c)
		}
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
