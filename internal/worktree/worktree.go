// Package worktree provides git worktree management for crush sessions.
package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// WorktreeDir is the directory where worktrees are created.
	WorktreeDir = ".crush-trees"
)

var (
	// ErrNotGitRepo is returned when the directory is not a git repository.
	ErrNotGitRepo = errors.New("not a git repository")
	// ErrInvalidName is returned when the session name is invalid.
	ErrInvalidName = errors.New("invalid session name")
	// ErrWorktreeExists is returned when a worktree with the same name exists.
	ErrWorktreeExists = errors.New("worktree already exists")
)

// validNameRegex matches valid worktree/branch names.
var validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Manager handles git worktree operations.
type Manager struct {
	repoRoot string
}

// NewManager creates a new worktree manager for the given repository root.
func NewManager(repoRoot string) (*Manager, error) {
	if !isGitRepo(repoRoot) {
		return nil, ErrNotGitRepo
	}
	return &Manager{repoRoot: repoRoot}, nil
}

// Create creates a new git worktree for a session.
// It creates the .crush-trees directory if needed, adds it to .gitignore,
// and creates a new worktree with a branch named after the session.
func (m *Manager) Create(ctx context.Context, name string) (string, error) {
	if !isValidName(name) {
		return "", ErrInvalidName
	}

	// Ensure .crush-trees directory exists.
	treesDir := filepath.Join(m.repoRoot, WorktreeDir)
	if err := os.MkdirAll(treesDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Add .crush-trees to .gitignore if not already present.
	if err := m.ensureGitignore(); err != nil {
		return "", fmt.Errorf("failed to update .gitignore: %w", err)
	}

	// Check if worktree already exists.
	worktreePath := filepath.Join(treesDir, name)
	if _, err := os.Stat(worktreePath); err == nil {
		return "", ErrWorktreeExists
	}

	// Get current branch/HEAD to base the new worktree on.
	baseRef, err := m.getCurrentRef(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current ref: %w", err)
	}

	// Create the worktree with a new branch.
	branchName := "crush/" + name
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branchName, worktreePath, baseRef)
	cmd.Dir = m.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w: %s", err, string(output))
	}

	return worktreePath, nil
}

// Remove removes a worktree by name.
func (m *Manager) Remove(ctx context.Context, name string) error {
	worktreePath := filepath.Join(m.repoRoot, WorktreeDir, name)

	// Remove the worktree.
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
	cmd.Dir = m.repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w: %s", err, string(output))
	}

	// Optionally delete the branch.
	branchName := "crush/" + name
	cmd = exec.CommandContext(ctx, "git", "branch", "-D", branchName)
	cmd.Dir = m.repoRoot
	// Ignore error if branch doesn't exist or can't be deleted.
	_ = cmd.Run()

	return nil
}

// List returns all worktrees in the .crush-trees directory.
func (m *Manager) List(ctx context.Context) ([]string, error) {
	treesDir := filepath.Join(m.repoRoot, WorktreeDir)
	entries, err := os.ReadDir(treesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// GetPath returns the full path to a worktree by name.
func (m *Manager) GetPath(name string) string {
	return filepath.Join(m.repoRoot, WorktreeDir, name)
}

// RepoRoot returns the repository root directory.
func (m *Manager) RepoRoot() string {
	return m.repoRoot
}

// ensureGitignore adds .crush-trees to .gitignore if not already present.
func (m *Manager) ensureGitignore() error {
	gitignorePath := filepath.Join(m.repoRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already in .gitignore.
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == WorktreeDir || line == WorktreeDir+"/" {
			return nil
		}
	}

	// Append to .gitignore.
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline if file doesn't end with one.
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	// Add the worktree directory.
	_, err = f.WriteString(WorktreeDir + "/\n")
	return err
}

// getCurrentRef returns the current HEAD reference (branch name or commit).
func (m *Manager) getCurrentRef(ctx context.Context) (string, error) {
	// Try to get branch name first.
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = m.repoRoot
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		if ref != "HEAD" {
			return ref, nil
		}
	}

	// Fall back to commit SHA.
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = m.repoRoot
	output, err = cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// isGitRepo checks if the directory is a git repository.
func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree).
	return info.IsDir() || info.Mode().IsRegular()
}

// isValidName checks if the name is valid for a worktree/branch.
func isValidName(name string) bool {
	if name == "" || len(name) > 100 {
		return false
	}
	return validNameRegex.MatchString(name)
}

// SanitizeName converts a string into a valid worktree name.
func SanitizeName(name string) string {
	// Replace spaces and special characters with hyphens.
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	// Remove consecutive hyphens.
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim leading/trailing hyphens.
	name = strings.Trim(name, "-")

	// Ensure it starts with alphanumeric.
	if len(name) > 0 {
		c := name[0]
		isAlphanumeric := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlphanumeric {
			name = "session-" + name
		}
	}

	// Truncate if too long.
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}
