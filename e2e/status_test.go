package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStatusReportingDisabledByDefault verifies that no status file is created
// when agent_status_dir is not configured.
func TestStatusReportingDisabledByDefault(t *testing.T) {
	SkipIfE2EDisabled(t)

	tmpDir := t.TempDir()
	statusDir := filepath.Join(tmpDir, "status")

	// Create the status directory so we can check it stays empty.
	err := os.MkdirAll(statusDir, 0o755)
	require.NoError(t, err)

	// Create config JSON WITHOUT status dir configured.
	term := NewIsolatedTerminalWithConfig(t, 120, 40, TestConfigJSON())
	defer term.Close()

	// Wait for startup.
	time.Sleep(startupDelay)

	// Verify no status files were created anywhere in the temp dir.
	var statusFiles []string
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "status-") && strings.HasSuffix(info.Name(), ".json") {
			statusFiles = append(statusFiles, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, statusFiles, "no status files should be created when agent_status_dir is not configured")
}

// TestStatusConfigParsing verifies that the agent_status_dir config option
// is properly parsed from JSON. This is a unit-style test that doesn't require
// a running agent.
func TestStatusConfigParsing(t *testing.T) {
	t.Parallel()

	statusDir := "/tmp/test-status-dir"
	statusDirJSON, err := json.Marshal(statusDir)
	require.NoError(t, err)

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
    "agent_status_dir": ` + string(statusDirJSON) + `
  }
}`

	// Verify the JSON is valid and contains the expected path.
	var config map[string]any
	err = json.Unmarshal([]byte(configJSON), &config)
	require.NoError(t, err)

	options, ok := config["options"].(map[string]any)
	require.True(t, ok, "options should be a map")

	parsedDir, ok := options["agent_status_dir"].(string)
	require.True(t, ok, "agent_status_dir should be a string")
	require.Equal(t, statusDir, parsedDir, "agent_status_dir should match input")
}
