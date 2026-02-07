package handlers

import (
	"bufio"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ChangelogEntry represents a single changelog entry for a version.
type ChangelogEntry struct {
	Version      string   `json:"version"`
	Date         string   `json:"date"`
	Added        []string `json:"added,omitempty"`
	Changed      []string `json:"changed,omitempty"`
	Deprecated   []string `json:"deprecated,omitempty"`
	Removed      []string `json:"removed,omitempty"`
	Fixed        []string `json:"fixed,omitempty"`
	Security     []string `json:"security,omitempty"`
	IsUnreleased bool     `json:"is_unreleased,omitempty"`
}

// ChangelogResponse is the response for the changelog endpoint.
type ChangelogResponse struct {
	Entries        []ChangelogEntry `json:"entries"`
	CurrentVersion string           `json:"current_version"`
}

// ChangelogHandler handles changelog-related HTTP endpoints.
type ChangelogHandler struct {
	changelogPath  string
	currentVersion string
	logger         zerolog.Logger
}

// NewChangelogHandler creates a new ChangelogHandler.
func NewChangelogHandler(changelogPath, currentVersion string, logger zerolog.Logger) *ChangelogHandler {
	return &ChangelogHandler{
		changelogPath:  changelogPath,
		currentVersion: currentVersion,
		logger:         logger.With().Str("component", "changelog_handler").Logger(),
	}
}

// RegisterRoutes registers changelog routes on the given router group.
func (h *ChangelogHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/changelog", h.List)
	r.GET("/changelog/:version", h.Get)
}

// RegisterPublicRoutes registers changelog routes that don't require authentication.
func (h *ChangelogHandler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/changelog", h.List)
}

// List returns all changelog entries.
//
//	@Summary		List changelog entries
//	@Description	Returns all changelog entries from the CHANGELOG.md file
//	@Tags			Changelog
//	@Produce		json
//	@Success		200	{object}	ChangelogResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/changelog [get]
func (h *ChangelogHandler) List(c *gin.Context) {
	entries, err := h.parseChangelog()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to parse changelog")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read changelog"})
		return
	}

	c.JSON(http.StatusOK, ChangelogResponse{
		Entries:        entries,
		CurrentVersion: h.currentVersion,
	})
}

// Get returns a specific changelog entry by version.
//
//	@Summary		Get changelog entry
//	@Description	Returns a specific changelog entry by version
//	@Tags			Changelog
//	@Produce		json
//	@Param			version	path		string	true	"Version number"
//	@Success		200		{object}	ChangelogEntry
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/changelog/{version} [get]
func (h *ChangelogHandler) Get(c *gin.Context) {
	version := c.Param("version")

	entries, err := h.parseChangelog()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to parse changelog")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read changelog"})
		return
	}

	for _, entry := range entries {
		if entry.Version == version {
			c.JSON(http.StatusOK, entry)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
}

// parseChangelog parses the CHANGELOG.md file and returns structured entries.
func (h *ChangelogHandler) parseChangelog() ([]ChangelogEntry, error) {
	file, err := os.Open(h.changelogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []ChangelogEntry
	var currentEntry *ChangelogEntry
	var currentSection string

	// Regex patterns for parsing
	versionPattern := regexp.MustCompile(`^## \[([^\]]+)\](?:\s*-\s*(\d{4}-\d{2}-\d{2}))?`)
	sectionPattern := regexp.MustCompile(`^### (Added|Changed|Deprecated|Removed|Fixed|Security)`)
	itemPattern := regexp.MustCompile(`^- (.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for version header
		if matches := versionPattern.FindStringSubmatch(line); matches != nil {
			// Save previous entry
			if currentEntry != nil {
				entries = append(entries, *currentEntry)
			}

			version := matches[1]
			date := ""
			if len(matches) > 2 {
				date = matches[2]
			}

			currentEntry = &ChangelogEntry{
				Version:      version,
				Date:         date,
				IsUnreleased: strings.ToLower(version) == "unreleased",
			}
			currentSection = ""
			continue
		}

		// Check for section header
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			currentSection = matches[1]
			continue
		}

		// Check for list item
		if currentEntry != nil && currentSection != "" {
			if matches := itemPattern.FindStringSubmatch(line); matches != nil {
				item := matches[1]
				switch currentSection {
				case "Added":
					currentEntry.Added = append(currentEntry.Added, item)
				case "Changed":
					currentEntry.Changed = append(currentEntry.Changed, item)
				case "Deprecated":
					currentEntry.Deprecated = append(currentEntry.Deprecated, item)
				case "Removed":
					currentEntry.Removed = append(currentEntry.Removed, item)
				case "Fixed":
					currentEntry.Fixed = append(currentEntry.Fixed, item)
				case "Security":
					currentEntry.Security = append(currentEntry.Security, item)
				}
			}
		}
	}

	// Save the last entry
	if currentEntry != nil {
		entries = append(entries, *currentEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
