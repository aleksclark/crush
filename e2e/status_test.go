package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/agent/status"
	"github.com/stretchr/testify/require"
)

// TestStatusReporting verifies that agent status is written to a file
// when agent_status_dir is configured in crush.json.
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

	// Find and read the status file.
	var statusFilePath string
	var rawJSON []byte
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "status-") && strings.HasSuffix(entry.Name(), ".json") {
				statusFilePath = filepath.Join(statusDir, entry.Name())
				rawJSON, err = os.ReadFile(statusFilePath)
				if err != nil {
					continue
				}
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "status file should be created in configured directory")

	// Validate filename format: status-<pid>.json
	filenamePattern := regexp.MustCompile(`^status-\d+\.json$`)
	require.True(t, filenamePattern.MatchString(filepath.Base(statusFilePath)),
		"status filename should match pattern status-<pid>.json, got: %s", filepath.Base(statusFilePath))

	// Parse as generic map first to validate JSON structure.
	var rawStatus map[string]any
	err := json.Unmarshal(rawJSON, &rawStatus)
	require.NoError(t, err, "status file should contain valid JSON")

	// Validate required fields exist.
	require.Contains(t, rawStatus, "state", "status must have 'state' field")
	require.Contains(t, rawStatus, "updated_at", "status must have 'updated_at' field")

	// Validate state is a valid value.
	stateVal, ok := rawStatus["state"].(string)
	require.True(t, ok, "state must be a string")
	validStates := []string{"idle", "thinking", "streaming", "tool_running"}
	require.Contains(t, validStates, stateVal, "state must be one of: %v", validStates)

	// Validate updated_at is a valid RFC3339 timestamp.
	updatedAtVal, ok := rawStatus["updated_at"].(string)
	require.True(t, ok, "updated_at must be a string")
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtVal)
	require.NoError(t, err, "updated_at must be a valid RFC3339 timestamp")
	require.WithinDuration(t, time.Now(), updatedAt, 30*time.Second,
		"updated_at should be recent")

	// Parse into typed struct for additional validation.
	var statusData status.Status
	err = json.Unmarshal(rawJSON, &statusData)
	require.NoError(t, err, "status file should unmarshal to Status struct")

	// For idle state, verify optional fields are empty/nil.
	require.Equal(t, status.StateIdle, statusData.State, "initial state should be idle")
	require.Empty(t, statusData.SessionID, "session_id should be empty for idle state")
	require.Nil(t, statusData.Tool, "tool should be nil for idle state")
	require.Nil(t, statusData.StartedAt, "started_at should be nil for idle state")
}

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

// TestStatusFileStructure validates the complete JSON schema of the status file.
func TestStatusFileStructure(t *testing.T) {
	SkipIfE2EDisabled(t)

	tmpDir := t.TempDir()
	statusDir := filepath.Join(tmpDir, "status")

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

	term := NewIsolatedTerminalWithConfig(t, 120, 40, configJSON)
	defer term.Close()

	time.Sleep(startupDelay)

	// Find and read status file.
	var rawJSON []byte
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "status-") && strings.HasSuffix(entry.Name(), ".json") {
				rawJSON, err = os.ReadFile(filepath.Join(statusDir, entry.Name()))
				return err == nil
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	// Parse as generic map to check for unexpected fields.
	var rawStatus map[string]any
	err := json.Unmarshal(rawJSON, &rawStatus)
	require.NoError(t, err)

	// Define allowed top-level fields per spec.
	allowedFields := map[string]bool{
		"state":      true,
		"session_id": true,
		"tool":       true,
		"updated_at": true,
		"started_at": true,
	}

	// Check no unexpected fields exist.
	for key := range rawStatus {
		require.True(t, allowedFields[key], "unexpected field in status JSON: %s", key)
	}

	// If tool field exists, validate its structure.
	if toolRaw, exists := rawStatus["tool"]; exists && toolRaw != nil {
		toolMap, ok := toolRaw.(map[string]any)
		require.True(t, ok, "tool must be an object")

		allowedToolFields := map[string]bool{
			"name":        true,
			"id":          true,
			"description": true,
		}
		for key := range toolMap {
			require.True(t, allowedToolFields[key], "unexpected field in tool object: %s", key)
		}

		// name and id are required when tool is present.
		require.Contains(t, toolMap, "name", "tool.name is required")
		require.Contains(t, toolMap, "id", "tool.id is required")
	}
}
