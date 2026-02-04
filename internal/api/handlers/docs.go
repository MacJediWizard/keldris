package handlers

import (
	"bufio"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// DocPage represents a documentation page.
type DocPage struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	HTMLContent string `json:"html_content,omitempty"`
}

// DocSearchResult represents a search result.
type DocSearchResult struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	Score   int    `json:"score"`
}

// DocsListResponse is the response for listing documentation.
type DocsListResponse struct {
	Pages []DocPage `json:"pages"`
}

// DocsSearchResponse is the response for searching documentation.
type DocsSearchResponse struct {
	Results []DocSearchResult `json:"results"`
	Query   string            `json:"query"`
}

// DocsHandler handles documentation-related HTTP endpoints.
type DocsHandler struct {
	docsFS   fs.FS
	embedded bool
	logger   zerolog.Logger
}

// NewDocsHandler creates a new DocsHandler using a filesystem.
func NewDocsHandler(docsFS fs.FS, logger zerolog.Logger) *DocsHandler {
	return &DocsHandler{
		docsFS:   docsFS,
		embedded: true,
		logger:   logger.With().Str("component", "docs_handler").Logger(),
	}
}

// NewDocsHandlerFromPath creates a new DocsHandler from a directory path.
func NewDocsHandlerFromPath(docsPath string, logger zerolog.Logger) *DocsHandler {
	return &DocsHandler{
		docsFS:   nil, // Will use os.ReadFile directly
		embedded: false,
		logger:   logger.With().Str("component", "docs_handler").Logger(),
	}
}

// RegisterPublicRoutes registers documentation routes that don't require authentication.
func (h *DocsHandler) RegisterPublicRoutes(r *gin.Engine) {
	docs := r.Group("/docs")
	{
		docs.GET("", h.List)
		docs.GET("/", h.List)
		docs.GET("/search", h.Search)
		docs.GET("/:slug", h.Get)
		docs.GET("/:slug/html", h.GetHTML)
	}
}

// RegisterRoutes registers documentation routes that require authentication.
func (h *DocsHandler) RegisterRoutes(r *gin.RouterGroup) {
	docs := r.Group("/docs")
	{
		docs.GET("", h.List)
		docs.GET("/search", h.Search)
		docs.GET("/:slug", h.Get)
		docs.GET("/:slug/html", h.GetHTML)
	}
}

// docMetadata holds the configuration for each doc page.
var docMetadata = map[string]DocPage{
	"getting-started": {
		Slug:        "getting-started",
		Title:       "Getting Started",
		Description: "Quick start guide for Keldris",
	},
	"installation": {
		Slug:        "installation",
		Title:       "Installation",
		Description: "Complete installation guide for server and agents",
	},
	"configuration": {
		Slug:        "configuration",
		Title:       "Configuration",
		Description: "All configuration options for Keldris",
	},
	"agent-deployment": {
		Slug:        "agent-deployment",
		Title:       "Agent Deployment",
		Description: "Deploy agents across your infrastructure",
	},
	"agent-installation": {
		Slug:        "agent-installation",
		Title:       "Agent Installation",
		Description: "Platform-specific agent installation instructions",
	},
	"api-reference": {
		Slug:        "api-reference",
		Title:       "API Reference",
		Description: "REST API documentation",
	},
	"troubleshooting": {
		Slug:        "troubleshooting",
		Title:       "Troubleshooting",
		Description: "Common issues and solutions",
	},
	"terraform": {
		Slug:        "terraform",
		Title:       "Terraform Provider",
		Description: "Infrastructure as code with Terraform",
	},
	"network-mounts": {
		Slug:        "network-mounts",
		Title:       "Network Mounts",
		Description: "Backing up network-mounted filesystems",
	},
	"rate-limits": {
		Slug:        "rate-limits",
		Title:       "Rate Limits",
		Description: "API rate limiting configuration",
	},
	"bare-metal-restore": {
		Slug:        "bare-metal-restore",
		Title:       "Bare Metal Restore",
		Description: "Full system recovery procedures",
	},
}

// List returns all available documentation pages.
// GET /docs
func (h *DocsHandler) List(c *gin.Context) {
	pages := make([]DocPage, 0, len(docMetadata))

	// Define the order of pages
	order := []string{
		"getting-started",
		"installation",
		"configuration",
		"agent-deployment",
		"agent-installation",
		"api-reference",
		"troubleshooting",
		"terraform",
		"network-mounts",
		"rate-limits",
		"bare-metal-restore",
	}

	for _, slug := range order {
		if page, exists := docMetadata[slug]; exists {
			pages = append(pages, page)
		}
	}

	c.JSON(http.StatusOK, DocsListResponse{Pages: pages})
}

// Get returns a specific documentation page.
// GET /docs/:slug
func (h *DocsHandler) Get(c *gin.Context) {
	slug := c.Param("slug")

	// Validate slug exists in metadata
	meta, exists := docMetadata[slug]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "documentation page not found"})
		return
	}

	content, err := h.readDocFile(slug + ".md")
	if err != nil {
		h.logger.Warn().Err(err).Str("slug", slug).Msg("failed to read doc file")
		c.JSON(http.StatusNotFound, gin.H{"error": "documentation page not found"})
		return
	}

	page := meta
	page.Content = content

	c.JSON(http.StatusOK, page)
}

// GetHTML returns a documentation page rendered as HTML.
// GET /docs/:slug/html
func (h *DocsHandler) GetHTML(c *gin.Context) {
	slug := c.Param("slug")

	// Validate slug exists in metadata
	meta, exists := docMetadata[slug]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "documentation page not found"})
		return
	}

	content, err := h.readDocFile(slug + ".md")
	if err != nil {
		h.logger.Warn().Err(err).Str("slug", slug).Msg("failed to read doc file")
		c.JSON(http.StatusNotFound, gin.H{"error": "documentation page not found"})
		return
	}

	// Simple markdown to HTML conversion
	html := h.markdownToHTML(content)

	page := meta
	page.HTMLContent = html

	c.JSON(http.StatusOK, page)
}

// Search searches within documentation.
// GET /docs/search?q=query
func (h *DocsHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query required"})
		return
	}

	query = strings.ToLower(query)
	queryWords := strings.Fields(query)

	var results []DocSearchResult

	for slug, meta := range docMetadata {
		content, err := h.readDocFile(slug + ".md")
		if err != nil {
			continue
		}

		lowerContent := strings.ToLower(content)
		score := 0

		// Score based on matches
		for _, word := range queryWords {
			// Title match is worth more
			if strings.Contains(strings.ToLower(meta.Title), word) {
				score += 10
			}
			// Description match
			if strings.Contains(strings.ToLower(meta.Description), word) {
				score += 5
			}
			// Content matches
			score += strings.Count(lowerContent, word)
		}

		if score > 0 {
			excerpt := h.extractExcerpt(content, queryWords[0], 150)
			results = append(results, DocSearchResult{
				Slug:    slug,
				Title:   meta.Title,
				Excerpt: excerpt,
				Score:   score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > 20 {
		results = results[:20]
	}

	c.JSON(http.StatusOK, DocsSearchResponse{
		Results: results,
		Query:   query,
	})
}

// readDocFile reads a documentation file.
func (h *DocsHandler) readDocFile(filename string) (string, error) {
	if h.docsFS != nil {
		data, err := fs.ReadFile(h.docsFS, filename)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Fallback to reading from docs directory
	data, err := fs.ReadFile(h.docsFS, filepath.Join("docs", filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// extractExcerpt extracts a text excerpt around the first match.
func (h *DocsHandler) extractExcerpt(content, query string, maxLen int) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		// Return beginning of content if no match
		if len(content) > maxLen {
			return strings.TrimSpace(content[:maxLen]) + "..."
		}
		return strings.TrimSpace(content)
	}

	// Find excerpt around the match
	start := idx - maxLen/2
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + maxLen/2
	if end > len(content) {
		end = len(content)
	}

	excerpt := content[start:end]

	// Clean up excerpt
	excerpt = strings.ReplaceAll(excerpt, "\n", " ")
	excerpt = strings.ReplaceAll(excerpt, "  ", " ")
	excerpt = strings.TrimSpace(excerpt)

	if start > 0 {
		excerpt = "..." + excerpt
	}
	if end < len(content) {
		excerpt = excerpt + "..."
	}

	return excerpt
}

// markdownToHTML performs a simple markdown to HTML conversion.
// This is a basic implementation - consider using a proper markdown library for production.
func (h *DocsHandler) markdownToHTML(md string) string {
	var html strings.Builder

	// Process line by line
	scanner := bufio.NewScanner(strings.NewReader(md))
	inCodeBlock := false
	inList := false
	inTable := false
	tableHeaderDone := false

	for scanner.Scan() {
		line := scanner.Text()

		// Code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				html.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				lang := strings.TrimPrefix(line, "```")
				html.WriteString("<pre><code class=\"language-" + template.HTMLEscapeString(lang) + "\">")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			html.WriteString(template.HTMLEscapeString(line) + "\n")
			continue
		}

		// Empty lines
		if strings.TrimSpace(line) == "" {
			if inList {
				html.WriteString("</ul>\n")
				inList = false
			}
			if inTable {
				html.WriteString("</tbody></table>\n")
				inTable = false
				tableHeaderDone = false
			}
			html.WriteString("\n")
			continue
		}

		// Tables
		if strings.HasPrefix(line, "|") && strings.HasSuffix(strings.TrimSpace(line), "|") {
			// Check if this is a separator line
			if regexp.MustCompile(`^\|[-:\s|]+\|$`).MatchString(line) {
				tableHeaderDone = true
				continue
			}

			if !inTable {
				html.WriteString("<table class=\"doc-table\">\n<thead>\n")
				inTable = true
			}

			cells := strings.Split(strings.Trim(line, "|"), "|")
			if !tableHeaderDone {
				html.WriteString("<tr>")
				for _, cell := range cells {
					html.WriteString("<th>" + template.HTMLEscapeString(strings.TrimSpace(cell)) + "</th>")
				}
				html.WriteString("</tr>\n</thead>\n<tbody>\n")
			} else {
				html.WriteString("<tr>")
				for _, cell := range cells {
					html.WriteString("<td>" + h.inlineMarkdown(strings.TrimSpace(cell)) + "</td>")
				}
				html.WriteString("</tr>\n")
			}
			continue
		}

		// Headers
		if strings.HasPrefix(line, "# ") {
			html.WriteString("<h1>" + template.HTMLEscapeString(strings.TrimPrefix(line, "# ")) + "</h1>\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			html.WriteString("<h2>" + template.HTMLEscapeString(strings.TrimPrefix(line, "## ")) + "</h2>\n")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			html.WriteString("<h3>" + template.HTMLEscapeString(strings.TrimPrefix(line, "### ")) + "</h3>\n")
			continue
		}
		if strings.HasPrefix(line, "#### ") {
			html.WriteString("<h4>" + template.HTMLEscapeString(strings.TrimPrefix(line, "#### ")) + "</h4>\n")
			continue
		}

		// Lists
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if !inList {
				html.WriteString("<ul>\n")
				inList = true
			}
			content := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			html.WriteString("<li>" + h.inlineMarkdown(content) + "</li>\n")
			continue
		}

		// Numbered lists
		if matched, _ := regexp.MatchString(`^\d+\. `, line); matched {
			if !inList {
				html.WriteString("<ol>\n")
				inList = true
			}
			content := regexp.MustCompile(`^\d+\. `).ReplaceAllString(line, "")
			html.WriteString("<li>" + h.inlineMarkdown(content) + "</li>\n")
			continue
		}

		// Close list if we're not in a list item
		if inList {
			html.WriteString("</ul>\n")
			inList = false
		}

		// Paragraphs
		html.WriteString("<p>" + h.inlineMarkdown(line) + "</p>\n")
	}

	// Close any open tags
	if inCodeBlock {
		html.WriteString("</code></pre>\n")
	}
	if inList {
		html.WriteString("</ul>\n")
	}
	if inTable {
		html.WriteString("</tbody></table>\n")
	}

	return html.String()
}

// inlineMarkdown converts inline markdown elements.
func (h *DocsHandler) inlineMarkdown(text string) string {
	// Escape HTML first
	text = template.HTMLEscapeString(text)

	// Inline code
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")

	// Bold
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")

	// Italic
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "<em>$1</em>")

	// Links
	text = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)

	return text
}

// EmbedDocs is a helper for embedding the docs directory.
// Usage in main:
//
//	//go:embed docs
//	var docsFS embed.FS
//	docsHandler := handlers.NewDocsHandler(docsFS, logger)
func EmbedDocs(docsFS embed.FS) (fs.FS, error) {
	return fs.Sub(docsFS, "docs")
}
