package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/status"
	"github.com/stretchr/testify/require"
)

// TestStatusReporting verifies that agent status is written to a file
// when agent_status_dir is configured.
func TestStatusReporting(t *testing.T) {
	SkipIfE2EDisabled(t)

	tmpDir := t.TempDir()
	statusDir := filepath.Join(tmpDir, "status")

	// Create config JSON with status dir configured.
	configJSON := `{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": { "provider": "test", "model": "test-model" },
    "small": { "provider": "test", "model": "test-model" }
  },
  "options": {
    "agent_status_dir": "` + strings.ReplaceAll(statusDir, `\`, `\\`) + `"
  }
}`

	// Create isolated terminal with status dir configured via config.
	term := NewIsolatedTerminalWithConfig(t, 120, 40, configJSON)
	defer term.Close()

	// Wait for startup.
	time.Sleep(startupDelay)

	// Verify status file exists in the directory (crush should write idle state on startup).
	// The filename includes the PID, so we need to find it.
	var statusData status.Status
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "status-") && strings.HasSuffix(entry.Name(), ".json") {
				data, err := os.ReadFile(filepath.Join(statusDir, entry.Name()))
				if err != nil {
					continue
				}
				if err := json.Unmarshal(data, &statusData); err != nil {
					continue
				}
				return statusData.State == status.StateIdle
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "status file should show idle state")

	// Verify the status file structure.
	require.Equal(t, status.StateIdle, statusData.State)
	require.Empty(t, statusData.SessionID)
	require.Nil(t, statusData.Tool)
}
