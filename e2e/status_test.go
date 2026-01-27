package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/status"
	"github.com/stretchr/testify/require"
)

// TestStatusReporting verifies that agent status is written to a file
// when CRUSH_STATUS_FILE is set.
func TestStatusReporting(t *testing.T) {
	SkipIfE2EDisabled(t)

	tmpDir := t.TempDir()
	statusPath := filepath.Join(tmpDir, "status.json")

	// Create isolated terminal with status file configured.
	term := NewIsolatedTerminalWithEnv(t, 120, 40, TestConfigJSON(), map[string]string{
		"CRUSH_STATUS_FILE": statusPath,
	})
	defer term.Close()

	// Wait for startup.
	time.Sleep(startupDelay)

	// Verify status file exists (crush should write idle state on startup).
	// Give it some extra time for the status to be written.
	var statusData status.Status
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(statusPath)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(data, &statusData); err != nil {
			return false
		}
		return statusData.State == status.StateIdle
	}, 5*time.Second, 100*time.Millisecond, "status file should show idle state")

	// Verify the status file structure.
	require.Equal(t, status.StateIdle, statusData.State)
	require.Empty(t, statusData.SessionID)
	require.Nil(t, statusData.Tool)
}
