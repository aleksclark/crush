package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReporter_SetIdle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")

	r := NewReporter(path)
	err := r.SetIdle()
	require.NoError(t, err)

	// Read and verify.
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	require.Equal(t, StateIdle, status.State)
	require.Empty(t, status.SessionID)
	require.Nil(t, status.Tool)
}

func TestReporter_SetThinking(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")

	r := NewReporter(path)
	err := r.SetThinking("session-123")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	require.Equal(t, StateThinking, status.State)
	require.Equal(t, "session-123", status.SessionID)
	require.NotNil(t, status.StartedAt)
	require.Nil(t, status.Tool)
}

func TestReporter_SetToolRunning(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")

	r := NewReporter(path)

	// First set thinking to establish StartedAt.
	err := r.SetThinking("session-123")
	require.NoError(t, err)

	startedAt := r.Current().StartedAt

	// Small delay to ensure time difference.
	time.Sleep(10 * time.Millisecond)

	err = r.SetToolRunning("session-123", "bash", "tool-456", "Running tests")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	require.Equal(t, StateToolRunning, status.State)
	require.Equal(t, "session-123", status.SessionID)
	require.NotNil(t, status.Tool)
	require.Equal(t, "bash", status.Tool.Name)
	require.Equal(t, "tool-456", status.Tool.ID)
	require.Equal(t, "Running tests", status.Tool.Description)

	// StartedAt should be preserved from thinking state.
	require.Equal(t, startedAt.Unix(), status.StartedAt.Unix())
}

func TestReporter_SetStreaming(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")

	r := NewReporter(path)

	err := r.SetThinking("session-123")
	require.NoError(t, err)

	err = r.SetStreaming("session-123")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	require.Equal(t, StateStreaming, status.State)
	require.Equal(t, "session-123", status.SessionID)
	require.NotNil(t, status.StartedAt)
}

func TestReporter_Disabled(t *testing.T) {
	t.Parallel()

	// Empty path disables reporting.
	r := NewReporter("")

	// All operations should succeed but not write anything.
	require.NoError(t, r.SetIdle())
	require.NoError(t, r.SetThinking("session-123"))
	require.NoError(t, r.SetToolRunning("session-123", "bash", "tool-456", ""))
	require.NoError(t, r.SetStreaming("session-123"))

	// Current should still track state internally.
	status := r.Current()
	require.Equal(t, StateStreaming, status.State)
}

func TestReporter_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Nested path that doesn't exist.
	path := filepath.Join(dir, "nested", "dir", "status.json")

	r := NewReporter(path)
	err := r.SetIdle()
	require.NoError(t, err)

	// File should exist.
	_, err = os.Stat(path)
	require.NoError(t, err)
}

func TestReporter_Close(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")

	r := NewReporter(path)
	err := r.SetThinking("session-123")
	require.NoError(t, err)

	err = r.Close()
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	// Should be idle after close.
	require.Equal(t, StateIdle, status.State)
}

func TestNewReporterInDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	r := NewReporterInDir(dir)
	err := r.SetIdle()
	require.NoError(t, err)

	// Verify file was created with PID-based name.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Contains(t, entries[0].Name(), "status-")
	require.Contains(t, entries[0].Name(), ".json")

	// Read and verify content.
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	require.NoError(t, err)

	var status Status
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)
	require.Equal(t, StateIdle, status.State)
}

func TestNewReporterInDir_EmptyDir(t *testing.T) {
	t.Parallel()

	// Empty directory path should disable reporting.
	r := NewReporterInDir("")
	require.NoError(t, r.SetIdle())
	require.NoError(t, r.SetThinking("session-123"))

	// Should still track state internally.
	status := r.Current()
	require.Equal(t, StateThinking, status.State)
}
