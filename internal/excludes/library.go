// Package excludes provides a library of pre-built exclude patterns for backups.
package excludes

// Category represents a category of exclude patterns.
type Category string

const (
	CategoryOS        Category = "os"
	CategoryIDE       Category = "ide"
	CategoryLanguage  Category = "language"
	CategoryBuild     Category = "build"
	CategoryCache     Category = "cache"
	CategoryTemp      Category = "temp"
	CategoryLogs      Category = "logs"
	CategorySecurity  Category = "security"
	CategoryDatabase  Category = "database"
	CategoryContainer Category = "container"
)

// BuiltInPattern represents a pre-defined exclude pattern.
type BuiltInPattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
	Category    Category `json:"category"`
}

// Library contains all built-in exclude patterns.
var Library = []BuiltInPattern{
	// OS-specific patterns
	{
		Name:        "macOS",
		Description: "macOS system files and metadata",
		Category:    CategoryOS,
		Patterns: []string{
			".DS_Store",
			".AppleDouble",
			".LSOverride",
			"._*",
			".Spotlight-V100",
			".Trashes",
			".fseventsd",
			".TemporaryItems",
			".VolumeIcon.icns",
			".com.apple.timemachine.donotpresent",
		},
	},
	{
		Name:        "Windows",
		Description: "Windows system files and metadata",
		Category:    CategoryOS,
		Patterns: []string{
			"Thumbs.db",
			"Thumbs.db:encryptable",
			"ehthumbs.db",
			"ehthumbs_vista.db",
			"desktop.ini",
			"$RECYCLE.BIN/",
			"*.lnk",
			"*.stackdump",
		},
	},
	{
		Name:        "Linux",
		Description: "Linux system files",
		Category:    CategoryOS,
		Patterns: []string{
			"*~",
			".nfs*",
			".fuse_hidden*",
			".directory",
			".Trash-*",
		},
	},

	// IDE patterns
	{
		Name:        "Visual Studio Code",
		Description: "VS Code workspace settings and cache",
		Category:    CategoryIDE,
		Patterns: []string{
			".vscode/*",
			"!.vscode/settings.json",
			"!.vscode/tasks.json",
			"!.vscode/launch.json",
			"!.vscode/extensions.json",
			"*.code-workspace",
			".history/",
		},
	},
	{
		Name:        "JetBrains IDEs",
		Description: "IntelliJ, WebStorm, PyCharm, etc.",
		Category:    CategoryIDE,
		Patterns: []string{
			".idea/",
			"*.iml",
			"*.ipr",
			"*.iws",
			".idea_modules/",
			"atlassian-ide-plugin.xml",
			"cmake-build-*/",
		},
	},
	{
		Name:        "Visual Studio",
		Description: "Visual Studio files and build output",
		Category:    CategoryIDE,
		Patterns: []string{
			".vs/",
			"*.suo",
			"*.user",
			"*.userosscache",
			"*.sln.docstates",
			"*.userprefs",
		},
	},
	{
		Name:        "Vim/Neovim",
		Description: "Vim swap and backup files",
		Category:    CategoryIDE,
		Patterns: []string{
			"*.swp",
			"*.swo",
			"*.swn",
			"*~",
			".netrwhist",
			"Session.vim",
			"Sessionx.vim",
		},
	},
	{
		Name:        "Emacs",
		Description: "Emacs backup and lock files",
		Category:    CategoryIDE,
		Patterns: []string{
			"*~",
			"\\#*\\#",
			"/.emacs.desktop",
			"/.emacs.desktop.lock",
			"*.elc",
			"auto-save-list",
			"tramp",
			".\\#*",
		},
	},
	{
		Name:        "Sublime Text",
		Description: "Sublime Text workspace files",
		Category:    CategoryIDE,
		Patterns: []string{
			"*.sublime-workspace",
			"*.sublime-project",
		},
	},

	// Language-specific patterns
	{
		Name:        "Node.js",
		Description: "Node.js dependencies and build artifacts",
		Category:    CategoryLanguage,
		Patterns: []string{
			"node_modules/",
			"npm-debug.log*",
			"yarn-debug.log*",
			"yarn-error.log*",
			".npm",
			".yarn-integrity",
			".pnp.*",
			".yarn/*",
			"!.yarn/patches",
			"!.yarn/plugins",
			"!.yarn/releases",
			"!.yarn/sdks",
			"!.yarn/versions",
		},
	},
	{
		Name:        "Python",
		Description: "Python bytecode, virtual environments, and caches",
		Category:    CategoryLanguage,
		Patterns: []string{
			"__pycache__/",
			"*.py[cod]",
			"*$py.class",
			"*.so",
			".Python",
			"build/",
			"develop-eggs/",
			"dist/",
			"downloads/",
			"eggs/",
			".eggs/",
			"lib/",
			"lib64/",
			"parts/",
			"sdist/",
			"var/",
			"wheels/",
			"*.egg-info/",
			".installed.cfg",
			"*.egg",
			"venv/",
			".venv/",
			"ENV/",
			"env/",
			".pyenv/",
			"Pipfile.lock",
			"poetry.lock",
		},
	},
	{
		Name:        "Go",
		Description: "Go build artifacts and vendor cache",
		Category:    CategoryLanguage,
		Patterns: []string{
			"*.exe",
			"*.exe~",
			"*.dll",
			"*.so",
			"*.dylib",
			"*.test",
			"*.out",
			"vendor/",
			"go.sum",
		},
	},
	{
		Name:        "Java",
		Description: "Java compiled classes and build directories",
		Category:    CategoryLanguage,
		Patterns: []string{
			"*.class",
			"*.jar",
			"*.war",
			"*.ear",
			"*.nar",
			"target/",
			"pom.xml.tag",
			"pom.xml.releaseBackup",
			"pom.xml.versionsBackup",
			"pom.xml.next",
			"release.properties",
			"dependency-reduced-pom.xml",
			"buildNumber.properties",
			".mvn/timing.properties",
			".mvn/wrapper/maven-wrapper.jar",
		},
	},
	{
		Name:        "Rust",
		Description: "Rust build artifacts",
		Category:    CategoryLanguage,
		Patterns: []string{
			"target/",
			"Cargo.lock",
			"**/*.rs.bk",
		},
	},
	{
		Name:        "Ruby",
		Description: "Ruby gems and bundle cache",
		Category:    CategoryLanguage,
		Patterns: []string{
			"*.gem",
			"*.rbc",
			"/.config",
			"/coverage/",
			"/InstalledFiles",
			"/pkg/",
			"/spec/reports/",
			"/spec/examples.txt",
			"/test/tmp/",
			"/test/version_tmp/",
			"/tmp/",
			".bundle/",
			"vendor/bundle",
			"lib/bundler/man/",
			".rvmrc",
		},
	},
	{
		Name:        ".NET/C#",
		Description: ".NET build output and packages",
		Category:    CategoryLanguage,
		Patterns: []string{
			"[Bb]in/",
			"[Oo]bj/",
			"[Ll]og/",
			"[Ll]ogs/",
			"*.nupkg",
			"*.snupkg",
			"packages/",
			".nuget/",
			"project.lock.json",
			"project.fragment.lock.json",
			"artifacts/",
		},
	},
	{
		Name:        "PHP",
		Description: "PHP dependencies and caches",
		Category:    CategoryLanguage,
		Patterns: []string{
			"vendor/",
			"composer.lock",
			".phpunit.result.cache",
			".php_cs.cache",
			".php-cs-fixer.cache",
		},
	},

	// Build artifacts
	{
		Name:        "Build Output",
		Description: "Common build output directories",
		Category:    CategoryBuild,
		Patterns: []string{
			"build/",
			"dist/",
			"out/",
			"output/",
			"bin/",
			"obj/",
			"target/",
			"release/",
			"debug/",
			"*.min.js",
			"*.min.css",
		},
	},
	{
		Name:        "Coverage Reports",
		Description: "Code coverage output",
		Category:    CategoryBuild,
		Patterns: []string{
			"coverage/",
			".nyc_output/",
			"*.lcov",
			".coverage",
			"htmlcov/",
			"coverage.xml",
			"*.cover",
		},
	},

	// Cache patterns
	{
		Name:        "Package Manager Caches",
		Description: "npm, yarn, pip, and other package manager caches",
		Category:    CategoryCache,
		Patterns: []string{
			".npm/",
			".yarn/cache/",
			".pnpm-store/",
			".cache/",
			".parcel-cache/",
			".next/cache/",
			".nuxt/",
			".turbo/",
			".gradle/",
			".m2/",
		},
	},
	{
		Name:        "Browser Caches",
		Description: "Browser and dev tool caches",
		Category:    CategoryCache,
		Patterns: []string{
			".sass-cache/",
			".stylelintcache",
			".eslintcache",
			".prettiercache",
			".browserslistrc",
		},
	},

	// Temporary files
	{
		Name:        "Temporary Files",
		Description: "Common temporary file patterns",
		Category:    CategoryTemp,
		Patterns: []string{
			"*.tmp",
			"*.temp",
			"*.bak",
			"*.backup",
			"*.old",
			"*.orig",
			"*.swp",
			"*.swo",
			"*~",
			"tmp/",
			"temp/",
		},
	},

	// Log files
	{
		Name:        "Log Files",
		Description: "Common log file patterns",
		Category:    CategoryLogs,
		Patterns: []string{
			"*.log",
			"logs/",
			"log/",
			"*.log.*",
			"npm-debug.log*",
			"yarn-debug.log*",
			"yarn-error.log*",
			"lerna-debug.log*",
		},
	},

	// Security-sensitive files
	{
		Name:        "Security & Secrets",
		Description: "Files that may contain sensitive information (ALWAYS exclude these)",
		Category:    CategorySecurity,
		Patterns: []string{
			".env",
			".env.*",
			"*.pem",
			"*.key",
			"*.p12",
			"*.pfx",
			".htpasswd",
			"credentials.json",
			"secrets.json",
			"*.secret",
			".netrc",
			".npmrc",
			".pypirc",
		},
	},

	// Database files
	{
		Name:        "Database Files",
		Description: "Database data files and dumps",
		Category:    CategoryDatabase,
		Patterns: []string{
			"*.sql",
			"*.sqlite",
			"*.sqlite3",
			"*.db",
			"*.mdb",
			"*.accdb",
			"dump.rdb",
			"appendonly.aof",
		},
	},

	// Container/VM patterns
	{
		Name:        "Docker",
		Description: "Docker build context exclusions",
		Category:    CategoryContainer,
		Patterns: []string{
			".docker/",
			"docker-compose.override.yml",
			"*.dockerfile",
			"Dockerfile.*",
		},
	},
	{
		Name:        "Vagrant",
		Description: "Vagrant VM files",
		Category:    CategoryContainer,
		Patterns: []string{
			".vagrant/",
			"*.box",
		},
	},

	// Version control
	{
		Name:        "Git",
		Description: "Git directory and related files",
		Category:    CategoryCache,
		Patterns: []string{
			".git/",
			".gitattributes",
			".gitmodules",
		},
	},
	{
		Name:        "SVN",
		Description: "Subversion metadata",
		Category:    CategoryCache,
		Patterns: []string{
			".svn/",
		},
	},
	{
		Name:        "Mercurial",
		Description: "Mercurial metadata",
		Category:    CategoryCache,
		Patterns: []string{
			".hg/",
			".hgignore",
			".hgsub",
			".hgsubstate",
			".hgtags",
		},
	},
}

// CategoryInfo provides metadata about pattern categories.
type CategoryInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// Categories returns metadata for all pattern categories.
var Categories = map[Category]CategoryInfo{
	CategoryOS: {
		Name:        "Operating System",
		Description: "OS-specific system files and metadata",
		Icon:        "computer",
	},
	CategoryIDE: {
		Name:        "IDE & Editors",
		Description: "IDE workspace settings and editor files",
		Icon:        "code",
	},
	CategoryLanguage: {
		Name:        "Languages",
		Description: "Language-specific files and dependencies",
		Icon:        "language",
	},
	CategoryBuild: {
		Name:        "Build Artifacts",
		Description: "Build output and compiled files",
		Icon:        "build",
	},
	CategoryCache: {
		Name:        "Caches",
		Description: "Package manager and tool caches",
		Icon:        "database",
	},
	CategoryTemp: {
		Name:        "Temporary Files",
		Description: "Temporary and backup files",
		Icon:        "clock",
	},
	CategoryLogs: {
		Name:        "Logs",
		Description: "Log files and debug output",
		Icon:        "file-text",
	},
	CategorySecurity: {
		Name:        "Security & Secrets",
		Description: "Sensitive files that should never be backed up",
		Icon:        "shield",
	},
	CategoryDatabase: {
		Name:        "Databases",
		Description: "Database files and dumps",
		Icon:        "database",
	},
	CategoryContainer: {
		Name:        "Containers & VMs",
		Description: "Container and virtual machine files",
		Icon:        "box",
	},
}

// GetAllCategories returns a list of all available categories.
func GetAllCategories() []Category {
	return []Category{
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
}
// GetPatternsByCategory returns all built-in patterns for a given category.
func GetPatternsByCategory(category Category) []BuiltInPattern {
	var patterns []BuiltInPattern
	for _, p := range Library {
		if p.Category == category {
			patterns = append(patterns, p)
		}
	}
	return patterns
}
// FlattenPatterns takes a list of BuiltInPatterns and returns all patterns as a single slice.
func FlattenPatterns(patterns []BuiltInPattern) []string {
	var result []string
	seen := make(map[string]bool)
	for _, p := range patterns {
		for _, pattern := range p.Patterns {
			if !seen[pattern] {
				seen[pattern] = true
				result = append(result, pattern)
			}
		}
	}
	return result
}
