package session

import (
	"context"
	"testing"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory SQLite database with all required tables.
func setupTestDB(t *testing.T) *db.Queries {
	t.Helper()

	database, err := driver.Open(":memory:", nil)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	// Create all required tables (simplified schema for testing).
	_, err = database.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			parent_session_id TEXT,
			title TEXT NOT NULL,
			message_count INTEGER NOT NULL DEFAULT 0,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			cost REAL NOT NULL DEFAULT 0.0,
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			summary_message_id TEXT,
			todos TEXT,
			working_dir TEXT
		);

		CREATE TABLE files (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			is_new INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE,
			UNIQUE(path, session_id, version)
		);

		CREATE TABLE messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			parts TEXT NOT NULL DEFAULT '[]',
			model TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			finished_at INTEGER,
			provider TEXT,
			is_summary_message INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE
		);
	`)
	require.NoError(t, err)

	// Use db.New() which works without prepared statements.
	return db.New(database)
}

func TestCreateWithWorkingDir(t *testing.T) {
	t.Parallel()

	t.Run("creates session with working directory", func(t *testing.T) {
		t.Parallel()
		queries := setupTestDB(t)
		svc := NewService(queries)

		session, err := svc.CreateWithWorkingDir(context.Background(), "test", "/path/to/worktree")
		require.NoError(t, err)
		require.NotEmpty(t, session.ID)
		require.Equal(t, "test", session.Title)
		require.Equal(t, "/path/to/worktree", session.WorkingDir)
	})

	t.Run("creates session without working directory", func(t *testing.T) {
		t.Parallel()
		queries := setupTestDB(t)
		svc := NewService(queries)

		session, err := svc.CreateWithWorkingDir(context.Background(), "test", "")
		require.NoError(t, err)
		require.NotEmpty(t, session.ID)
		require.Equal(t, "test", session.Title)
		require.Empty(t, session.WorkingDir)
	})

	t.Run("Create delegates to CreateWithWorkingDir", func(t *testing.T) {
		t.Parallel()
		queries := setupTestDB(t)
		svc := NewService(queries)

		session, err := svc.Create(context.Background(), "test")
		require.NoError(t, err)
		require.NotEmpty(t, session.ID)
		require.Equal(t, "test", session.Title)
		require.Empty(t, session.WorkingDir)
	})
}

func TestGetSessionPreservesWorkingDir(t *testing.T) {
	t.Parallel()

	queries := setupTestDB(t)
	svc := NewService(queries)

	// Create a session with a working directory.
	created, err := svc.CreateWithWorkingDir(context.Background(), "test", "/path/to/worktree")
	require.NoError(t, err)

	// Retrieve the session.
	retrieved, err := svc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, retrieved.ID)
	require.Equal(t, "/path/to/worktree", retrieved.WorkingDir)
}

func TestListSessionsPreservesWorkingDir(t *testing.T) {
	t.Parallel()

	queries := setupTestDB(t)
	svc := NewService(queries)

	// Create sessions with different working directories.
	_, err := svc.CreateWithWorkingDir(context.Background(), "session1", "/path/to/worktree1")
	require.NoError(t, err)
	_, err = svc.CreateWithWorkingDir(context.Background(), "session2", "/path/to/worktree2")
	require.NoError(t, err)
	_, err = svc.CreateWithWorkingDir(context.Background(), "session3", "")
	require.NoError(t, err)

	// List all sessions.
	sessions, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, sessions, 3)

	// Verify working directories are preserved (sessions are ordered by updated_at DESC).
	workingDirs := make(map[string]string)
	for _, s := range sessions {
		workingDirs[s.Title] = s.WorkingDir
	}
	require.Equal(t, "/path/to/worktree1", workingDirs["session1"])
	require.Equal(t, "/path/to/worktree2", workingDirs["session2"])
	require.Empty(t, workingDirs["session3"])
}

func TestSaveSessionPreservesWorkingDir(t *testing.T) {
	t.Parallel()

	queries := setupTestDB(t)
	svc := NewService(queries)

	// Create a session with a working directory.
	created, err := svc.CreateWithWorkingDir(context.Background(), "test", "/path/to/worktree")
	require.NoError(t, err)

	// Update the session's title.
	created.Title = "updated title"
	saved, err := svc.Save(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, "updated title", saved.Title)
	require.Equal(t, "/path/to/worktree", saved.WorkingDir)

	// Verify persistence.
	retrieved, err := svc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "updated title", retrieved.Title)
	require.Equal(t, "/path/to/worktree", retrieved.WorkingDir)
}

func TestSaveSessionCanUpdateWorkingDir(t *testing.T) {
	t.Parallel()

	queries := setupTestDB(t)
	svc := NewService(queries)

	// Create a session without a working directory.
	created, err := svc.Create(context.Background(), "test")
	require.NoError(t, err)
	require.Empty(t, created.WorkingDir)

	// Update the session to add a working directory.
	created.WorkingDir = "/path/to/new/worktree"
	saved, err := svc.Save(context.Background(), created)
	require.NoError(t, err)
	require.Equal(t, "/path/to/new/worktree", saved.WorkingDir)

	// Verify persistence.
	retrieved, err := svc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "/path/to/new/worktree", retrieved.WorkingDir)
}
