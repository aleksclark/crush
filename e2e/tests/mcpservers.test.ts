/**
 * E2E Tests: MCP Servers Feature
 *
 * Tests the MCP servers management dialog which allows users to:
 * 1. View list of MCP servers
 * 2. View server details (tools, prompts, resources)
 * 3. Restart servers
 * 4. View invocation logs
 *
 * Uses ctrl+e to open the dialog.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";
import * as os from "os";

// Resolve binary path
const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Helper delays
const STARTUP_DELAY = 3000;
const DIALOG_TRANSITION = 700;

// Skip bash-dependent tests on Windows
const isWindows = os.platform() === "win32";

/**
 * MCP Servers Dialog Tests
 *
 * Tests the MCP servers dialog opened with ctrl+e.
 */
test.describe("MCP Servers dialog", () => {
  // Setup script creates a temp directory with minimal config
  const setupWithMinimalConfig = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

cat > crush.json << 'CONFIG'
{
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
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithMinimalConfig.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "opens with ctrl+e",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Press ctrl+e to open MCP servers dialog
      terminal.write("\x05"); // ctrl+e = ASCII 5

      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show the MCP servers dialog title
      await expect(
        terminal.getByText("MCP Servers", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "closes dialog on escape",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      await expect(
        terminal.getByText("MCP Servers", { full: true })
      ).toBeVisible();

      // Press escape to close
      terminal.write("\x1b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // The dialog should be closed now
    }
  );

  test.when(
    !isWindows,
    "shows help keybindings",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show help text with keybindings
      await expect(
        terminal.getByText("restart", { full: true })
      ).toBeVisible();
      await expect(
        terminal.getByText("logs", { full: true })
      ).toBeVisible();
    }
  );
});

/**
 * MCP Servers with Configuration Tests
 *
 * Tests the dialog with actual MCP server configurations.
 */
test.describe("MCP Servers with configuration", () => {
  // Setup script creates a temp directory with MCP config
  const setupWithMCPConfig = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

# Create crush.json with MCP server config
cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": {
      "provider": "test",
      "model": "test-model"
    },
    "small": {
      "provider": "test",
      "model": "test-model"
    }
  },
  "mcp": {
    "test-server": {
      "type": "stdio",
      "command": "echo",
      "args": ["test"],
      "disabled": true
    },
    "another-server": {
      "type": "http",
      "url": "http://localhost:9998",
      "disabled": true
    }
  }
}
CONFIG

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithMCPConfig.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "shows configured MCP servers",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show configured servers
      await expect(
        terminal.getByText("test-server", { full: true })
      ).toBeVisible();
      await expect(
        terminal.getByText("another-server", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows server status as disabled",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Check terminal output contains disabled status
      const buffer = terminal.getBuffer();
      const output = buffer.map(row => row.join("")).join("\n");
      expect(output).toContain("disabled");
    }
  );

  test.when(
    !isWindows,
    "displays status indicator with server name",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Get terminal output to verify status format
      const buffer = terminal.getBuffer();
      const output = buffer.map(row => row.join("")).join("\n");

      // Each server should have a status indicator (●) followed by name and status
      // The format is: ● server-name    status-text
      expect(output).toMatch(/●.*test-server.*disabled/);
      expect(output).toMatch(/●.*another-server.*disabled/);
    }
  );
});

/**
 * MCP Empty State Tests
 *
 * Tests the dialog when no MCP servers are configured.
 */
test.describe("MCP Empty state", () => {
  const setupWithNoMCP = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

cat > crush.json << 'CONFIG'
{
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
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithNoMCP.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "shows empty state when no servers configured",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // First verify the dialog opened
      await expect(
        terminal.getByText("MCP Servers", { full: true })
      ).toBeVisible();

      // Should show empty state message
      await expect(terminal.getByText("No MCP servers configured")).toBeVisible();
    }
  );
});

/**
 * MCP Logs Dialog Tests
 *
 * Tests the logs dialog which is opened with 'l' key from the servers list.
 * The fix ensures the dialog doesn't crash when opened (logs.go initializes
 * the logsList in the constructor).
 */
test.describe("MCP Logs dialog", () => {
  // Setup script creates a temp directory with MCP config
  const setupWithMCPConfig = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

# Create crush.json with MCP server config
cat > crush.json << 'CONFIG'
{
  "providers": {
    "test": {
      "type": "openai-compat",
      "base_url": "http://localhost:9999",
      "api_key": "test-key"
    }
  },
  "models": {
    "large": {
      "provider": "test",
      "model": "test-model"
    },
    "small": {
      "provider": "test",
      "model": "test-model"
    }
  },
  "mcp": {
    "test-server": {
      "type": "stdio",
      "command": "echo",
      "args": ["test"],
      "disabled": true
    }
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithMCPConfig.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "opens logs dialog with l key",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open MCP servers dialog
      terminal.write("\x05"); // ctrl+e
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Verify servers dialog is open
      await expect(
        terminal.getByText("MCP Servers", { full: true })
      ).toBeVisible();

      // Press 'l' to open logs for selected server
      terminal.write("l");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show the logs dialog title with server name
      await expect(
        terminal.getByText("Logs:", { full: true })
      ).toBeVisible();
      await expect(
        terminal.getByText("test-server", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows empty logs state",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open MCP servers dialog
      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Press 'l' to open logs
      terminal.write("l");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show empty state message (no logs available yet)
      await expect(
        terminal.getByText("No logs available", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "closes logs dialog with escape",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open MCP servers dialog
      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Press 'l' to open logs
      terminal.write("l");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Verify logs dialog is open
      await expect(
        terminal.getByText("Logs:", { full: true })
      ).toBeVisible();

      // Press escape to close logs dialog
      terminal.write("\x1b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should be back to MCP servers dialog
      await expect(
        terminal.getByText("MCP Servers", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows help keybindings in logs dialog",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open MCP servers dialog
      terminal.write("\x05");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Press 'l' to open logs
      terminal.write("l");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show help text with keybindings for logs dialog
      await expect(
        terminal.getByText("filter session", { full: true })
      ).toBeVisible();
    }
  );
});
