package subagent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// AgentFileName is the expected filename for subagent definitions.
	AgentFileName = "*.md"

	// MaxNameLength is the maximum length for a subagent name.
	MaxNameLength = 64

	// MaxDescriptionLength is the maximum length for a subagent description.
	MaxDescriptionLength = 2048
)

// namePattern validates subagent names: lowercase letters, numbers, and hyphens.
// No leading, trailing, or consecutive hyphens.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Parse reads and parses a subagent definition file.
func Parse(path string) (*Subagent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	return ParseContent(string(content), path, info.ModTime())
}

// ParseContent parses subagent content from a string.
func ParseContent(content, filePath string, modTime ...time.Time) (*Subagent, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	var subagent Subagent
	if err := yaml.Unmarshal([]byte(frontmatter), &subagent); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	subagent.SystemPrompt = strings.TrimSpace(body)
	subagent.FilePath = filePath
	subagent.Path = filepath.Dir(filePath)

	// Set mod time if provided.
	if len(modTime) > 0 {
		subagent.ModTime = modTime[0]
	}

	return &subagent, nil
}

// splitFrontmatter extracts YAML frontmatter and body from markdown content.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	// Normalize line endings.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return "", "", errors.New("no YAML frontmatter found (must start with ---)")
	}

	rest := strings.TrimPrefix(content, "---\n")
	rest = strings.TrimPrefix(rest, "---\r\n")

	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return "", "", errors.New("unclosed frontmatter (missing closing ---)")
	}

	frontmatter = rest[:idx]
	body = strings.TrimPrefix(rest[idx:], "\n---")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	return frontmatter, body, nil
}

// Validate checks if the subagent meets all requirements.
func (s *Subagent) Validate() error {
	var errs []error

	// Validate name.
	if s.Name == "" {
		errs = append(errs, errors.New("name is required"))
	} else {
		if len(s.Name) > MaxNameLength {
			errs = append(errs, fmt.Errorf("name exceeds %d characters", MaxNameLength))
		}
		if !namePattern.MatchString(s.Name) {
			errs = append(errs, errors.New("name must be lowercase alphanumeric with hyphens, starting with a letter"))
		}
	}

	// Validate description.
	if s.Description == "" {
		errs = append(errs, errors.New("description is required"))
	} else if len(s.Description) > MaxDescriptionLength {
		errs = append(errs, fmt.Errorf("description exceeds %d characters", MaxDescriptionLength))
	}

	// Validate model.
	if !s.Model.IsValid() {
		errs = append(errs, fmt.Errorf("invalid model %q: must be one of sonnet, opus, haiku, inherit", s.Model))
	}

	// Validate permission mode.
	if !s.PermissionMode.IsValid() {
		errs = append(errs, fmt.Errorf("invalid permission_mode %q: must be one of default, acceptEdits, dontAsk, bypassPermissions, plan", s.PermissionMode))
	}

	return errors.Join(errs...)
}

// ValidateTools checks that all specified tools are in the allowed list.
func (s *Subagent) ValidateTools(availableTools []string) error {
	if len(s.Tools) == 0 {
		return nil
	}

	toolSet := make(map[string]bool, len(availableTools))
	for _, t := range availableTools {
		toolSet[t] = true
	}

	var unknown []string
	for _, t := range s.Tools {
		if !toolSet[t] {
			unknown = append(unknown, t)
		}
	}

	if len(unknown) > 0 {
		return fmt.Errorf("unknown tools: %s", strings.Join(unknown, ", "))
	}

	return nil
}
