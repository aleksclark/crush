package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-git directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, err := NewManager(dir)
		require.ErrorIs(t, err, ErrNotGitRepo)
	})

	t.Run("succeeds for git repository", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)

		mgr, err := NewManager(dir)
		require.NoError(t, err)
		require.NotNil(t, mgr)
		require.Equal(t, dir, mgr.RepoRoot())
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates worktree with valid name", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)
		mgr, err := NewManager(dir)
		require.NoError(t, err)

		path, err := mgr.Create(context.Background(), "test-feature")
		require.NoError(t, err)
		require.DirExists(t, path)
		require.Equal(t, filepath.Join(dir, WorktreeDir, "test-feature"), path)

		// Verify .gitignore was updated.
		gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		require.Contains(t, string(gitignore), WorktreeDir+"/")
	})

	t.Run("fails with invalid name", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)
		mgr, err := NewManager(dir)
		require.NoError(t, err)

		_, err = mgr.Create(context.Background(), "")
		require.ErrorIs(t, err, ErrInvalidName)

		_, err = mgr.Create(context.Background(), "../escape")
		require.ErrorIs(t, err, ErrInvalidName)
	})

	t.Run("fails when worktree already exists", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)
		mgr, err := NewManager(dir)
		require.NoError(t, err)

		_, err = mgr.Create(context.Background(), "duplicate")
		require.NoError(t, err)

		_, err = mgr.Create(context.Background(), "duplicate")
		require.ErrorIs(t, err, ErrWorktreeExists)
	})
}

func TestRemove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing worktree", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)
		mgr, err := NewManager(dir)
		require.NoError(t, err)

		path, err := mgr.Create(context.Background(), "to-remove")
		require.NoError(t, err)
		require.DirExists(t, path)

		err = mgr.Remove(context.Background(), "to-remove")
		require.NoError(t, err)
		require.NoDirExists(t, path)
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("lists worktrees", func(t *testing.T) {
		t.Parallel()
		dir := setupGitRepo(t)
		mgr, err := NewManager(dir)
		require.NoError(t, err)

		// Initially empty.
		names, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Empty(t, names)

		// Create some worktrees.
		_, err = mgr.Create(context.Background(), "feature-a")
		require.NoError(t, err)
		_, err = mgr.Create(context.Background(), "feature-b")
		require.NoError(t, err)

		names, err = mgr.List(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"feature-a", "feature-b"}, names)
	})
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"feature name", "feature-name"},
		{"Feature/Branch", "Feature-Branch"},
		{"test_123", "test_123"},
		{"  spaced  ", "spaced"},
		{"--leading", "leading"},
		{"trailing--", "trailing"},
		{"a", "a"},
		{"", ""},
		{"CAPS", "CAPS"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := SanitizeName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidName(t *testing.T) {
	t.Parallel()

	valid := []string{"feature", "test-123", "my_branch", "CamelCase", "a1"}
	invalid := []string{"", "../escape", "with space", "-leading", "_leading", "a/b"}

	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			t.Parallel()
			require.True(t, isValidName(name), "expected %q to be valid", name)
		})
	}

	for _, name := range invalid {
		t.Run("invalid_"+name, func(t *testing.T) {
			t.Parallel()
			require.False(t, isValidName(name), "expected %q to be invalid", name)
		})
	}
}

// setupGitRepo creates a temporary git repository for testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo.
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	// Configure git user for commits.
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	// Create initial commit (required for worktrees).
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# Test\n"), 0o644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	return dir
}
