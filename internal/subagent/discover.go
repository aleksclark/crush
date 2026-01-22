package subagent

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/charlievieth/fastwalk"
)

const (
	// CrushAgentsDir is the directory name for Crush subagents.
	CrushAgentsDir = ".crush/agents"
	// ClaudeAgentsDir is the directory name for Claude Code compatibility.
	ClaudeAgentsDir = ".claude/agents"
)

// DiscoverPaths returns the standard paths to search for subagent definitions.
// Additional paths are prepended (higher priority than defaults).
func DiscoverPaths(workingDir, userConfigDir string, additionalPaths []string) []string {
	var paths []string

	// Additional paths have highest priority.
	paths = append(paths, additionalPaths...)

	// Project-level paths (higher priority).
	if workingDir != "" {
		paths = append(paths, filepath.Join(workingDir, CrushAgentsDir))
		paths = append(paths, filepath.Join(workingDir, ClaudeAgentsDir))
	}

	// User-level paths (lower priority).
	if userConfigDir != "" {
		paths = append(paths, filepath.Join(userConfigDir, "agents"))
	}

	// Home directory ~/.claude/agents for Claude Code compatibility.
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ClaudeAgentsDir))
	}

	return paths
}

// Discover finds and parses all valid subagent definitions from the given paths.
// Paths earlier in the list have higher priority (project > user).
// Returns subagents sorted by priority (highest first) then by name.
func Discover(paths []string) []*Subagent {
	var subagents []*Subagent
	var mu sync.Mutex
	seen := make(map[string]bool) // Track seen names to handle priority.

	for priority, basePath := range paths {
		// Skip if path doesn't exist.
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}

		conf := fastwalk.Config{
			Follow:  true,
			ToSlash: fastwalk.DefaultToSlash(),
		}

		err := fastwalk.Walk(&conf, basePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Only process .md files.
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}

			subagent, err := Parse(path)
			if err != nil {
				slog.Warn("Failed to parse subagent file", "path", path, "error", err)
				return nil
			}

			if err := subagent.Validate(); err != nil {
				slog.Warn("Subagent validation failed", "path", path, "name", subagent.Name, "error", err)
				return nil
			}

			mu.Lock()
			defer mu.Unlock()

			// Skip if we've already seen this name (higher priority path wins).
			if seen[subagent.Name] {
				slog.Debug("Skipping duplicate subagent", "name", subagent.Name, "path", path)
				return nil
			}

			seen[subagent.Name] = true
			subagent.Priority = len(paths) - priority // Higher index = lower priority.
			subagents = append(subagents, subagent)

			slog.Info("Discovered subagent", "name", subagent.Name, "path", path, "priority", subagent.Priority)
			return nil
		})
		if err != nil {
			slog.Warn("Error walking path", "path", basePath, "error", err)
		}
	}

	// Sort by priority (descending), then by name (ascending).
	sort.Slice(subagents, func(i, j int) bool {
		if subagents[i].Priority != subagents[j].Priority {
			return subagents[i].Priority > subagents[j].Priority
		}
		return subagents[i].Name < subagents[j].Name
	})

	return subagents
}

// DiscoverFromDir discovers subagents from a single directory.
func DiscoverFromDir(dir string) []*Subagent {
	return Discover([]string{dir})
}

// FindAgentFile looks for a subagent file by name in the given paths.
func FindAgentFile(name string, paths []string) (string, bool) {
	for _, basePath := range paths {
		// Try exact name with .md extension.
		candidate := filepath.Join(basePath, name+".md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}

		// Try in a subdirectory with the same name.
		candidate = filepath.Join(basePath, name, name+".md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}
