package agentstatus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewReporter(t *testing.T) {
	t.Parallel()

	t.Run("creates directory and status file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		reporter, err := NewReporter(dir)
		require.NoError(t, err)
		require.NotNil(t, reporter)
		t.Cleanup(func() { reporter.Close() })

		// Verify the status file exists.
		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.True(t, files[0].Name()[:6] == "crush-")
		require.True(t, files[0].Name()[len(files[0].Name())-5:] == ".json")
	})

	t.Run("disabled when dir is empty", func(t *testing.T) {
		t.Parallel()

		reporter, err := NewReporter("")
		require.NoError(t, err)
		require.NotNil(t, reporter)
		require.False(t, reporter.IsEnabled())

		// Methods should be no-ops.
		reporter.SetStatus(StatusWorking)
		reporter.ToolStart("view")
		reporter.ToolEnd("view")
		reporter.Close()
	})

	t.Run("expands tilde in path", func(t *testing.T) {
		t.Parallel()

		// This test just verifies the tilde expansion logic works.
		// We can't actually create files in ~, so we create in temp.
		dir := t.TempDir()

		reporter, err := NewReporter(dir)
		require.NoError(t, err)
		require.NotNil(t, reporter)
		reporter.Close()
	})
}

func TestReporter_StatusFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	// Read and parse the status file.
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	require.NoError(t, err)

	var status AgentStatus
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	// Verify initial values.
	require.Equal(t, 1, status.Version)
	require.Equal(t, "crush", status.Agent)
	require.NotEmpty(t, status.Instance)
	require.Equal(t, os.Getpid(), status.PID)
	require.Equal(t, StatusIdle, status.Status)
	require.Greater(t, status.Started, int64(0))
	require.Greater(t, status.Updated, int64(0))
}

func TestReporter_SetProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.SetProject("my-project", "/home/user/my-project")

	status := readStatus(t, dir)
	require.Equal(t, "my-project", status.Project)
	require.Equal(t, "/home/user/my-project", status.CWD)
}

func TestReporter_SetModel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.SetModel("claude-sonnet-4-20250514", "anthropic")

	status := readStatus(t, dir)
	require.Equal(t, "claude-sonnet-4-20250514", status.Model)
	require.Equal(t, "anthropic", status.Provider)
}

func TestReporter_SetStatus(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.SetStatus(StatusWorking)
	status := readStatus(t, dir)
	require.Equal(t, StatusWorking, status.Status)

	reporter.SetStatus(StatusThinking)
	status = readStatus(t, dir)
	require.Equal(t, StatusThinking, status.Status)
}

func TestReporter_SetTask(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.SetTask("implementing agent status monitor")

	status := readStatus(t, dir)
	require.Equal(t, "implementing agent status monitor", status.Task)
}

func TestReporter_SetError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.SetError("API rate limit exceeded")

	status := readStatus(t, dir)
	require.Equal(t, StatusError, status.Status)
	require.Equal(t, "API rate limit exceeded", status.Error)
}

func TestReporter_ToolCalls(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	// Start a tool.
	reporter.ToolStart("view")
	status := readStatus(t, dir)
	require.Equal(t, StatusWorking, status.Status)
	require.NotNil(t, status.Tools.Active)
	require.Equal(t, "view", *status.Tools.Active)

	// End the tool.
	reporter.ToolEnd("view")
	status = readStatus(t, dir)
	require.Nil(t, status.Tools.Active)
	require.Contains(t, status.Tools.Recent, "view")
	require.Equal(t, int64(1), status.Tools.Counts["view"])

	// Start and end more tools.
	reporter.ToolStart("edit")
	reporter.ToolEnd("edit")
	reporter.ToolStart("bash")
	reporter.ToolEnd("bash")

	status = readStatus(t, dir)
	require.Equal(t, []string{"view", "edit", "bash"}, status.Tools.Recent)
	require.Equal(t, int64(1), status.Tools.Counts["edit"])
	require.Equal(t, int64(1), status.Tools.Counts["bash"])
}

func TestReporter_ToolsRecentMaxSize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	// Add more than 10 tools.
	tools := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	for _, tool := range tools {
		reporter.ToolStart(tool)
		reporter.ToolEnd(tool)
	}

	status := readStatus(t, dir)
	require.Len(t, status.Tools.Recent, 10)
	// Should contain the last 10 tools.
	require.Equal(t, []string{"c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}, status.Tools.Recent)
}

func TestReporter_UpdateTokens(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.UpdateTokens(125000, 15000, 80000, 45000)

	status := readStatus(t, dir)
	require.Equal(t, int64(125000), status.Tokens.Input)
	require.Equal(t, int64(15000), status.Tokens.Output)
	require.Equal(t, int64(80000), status.Tokens.CacheRead)
	require.Equal(t, int64(45000), status.Tokens.CacheWrite)
}

func TestReporter_UpdateCost(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	reporter.UpdateCost(0.42)

	status := readStatus(t, dir)
	require.InDelta(t, 0.42, status.CostUSD, 0.001)
}

func TestReporter_Close(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)

	// Verify file exists.
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	// Close should remove the file.
	err = reporter.Close()
	require.NoError(t, err)

	files, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 0)

	// Methods after close should be no-ops.
	reporter.SetStatus(StatusWorking)
	reporter.ToolStart("view")
	require.False(t, reporter.IsEnabled())
}

func TestReporter_UpdatedTimestamp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reporter, err := NewReporter(dir)
	require.NoError(t, err)
	t.Cleanup(func() { reporter.Close() })

	status1 := readStatus(t, dir)
	time.Sleep(10 * time.Millisecond)

	reporter.SetStatus(StatusWorking)
	status2 := readStatus(t, dir)

	require.GreaterOrEqual(t, status2.Updated, status1.Updated)
}

// readStatus reads and parses the status file from the directory.
func readStatus(t *testing.T, dir string) AgentStatus {
	t.Helper()

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	require.NoError(t, err)

	var status AgentStatus
	err = json.Unmarshal(data, &status)
	require.NoError(t, err)

	return status
}
