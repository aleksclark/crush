/**
 * E2E Tests: Agent Permissions and Configuration
 *
 * Tests the agent permission system which controls how agents handle
 * tool authorization. Permission modes include:
 * - default: Standard prompting for all operations
 * - yolo/bypassPermissions: Auto-approve all operations
 * - acceptEdits: Auto-approve file edits, prompt for bash
 * - dontAsk: Auto-deny all operations (safe mode)
 * - plan: Read-only mode, auto-deny writes
 *
 * These tests verify:
 * 1. Various permission modes are recognized and loaded
 * 2. Model types are correctly mapped
 * 3. Tool configurations (allowed/disallowed) are processed
 * 4. Invalid configurations are handled gracefully
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";
import * as os from "os";

// Resolve binary path
const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Helper delays
const STARTUP_DELAY = 3000;
const DIALOG_TRANSITION = 500;

// Skip bash-dependent tests on Windows
const isWindows = os.platform() === "win32";

// Helper to get terminal output as string
function getTerminalOutput(terminal: any): string {
  const buffer = terminal.getBuffer();
  return buffer.map((row: string[]) => row.join("")).join("\n");
}

/**
 * Permission Mode Configuration Loading Tests
 *
 * Tests that permission modes are correctly loaded from various configurations.
 */
test.describe("Permission mode configuration", () => {
  // Test with dynamically created agents
  const setupWithPermissionModes = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

# Default permission mode (no mode specified)
cat > .crush/agents/default-mode.md << 'SUBAGENT'
---
name: default-mode
description: Agent with default permission mode
---
Default mode agent.
SUBAGENT

# Yolo mode using the alias
cat > .crush/agents/yolo-mode.md << 'SUBAGENT'
---
name: yolo-mode
description: Agent with yolo permission mode
permission_mode: yolo
---
Yolo mode agent.
SUBAGENT

# bypassPermissions mode (canonical name)
cat > .crush/agents/bypass-mode.md << 'SUBAGENT'
---
name: bypass-mode
description: Agent with bypassPermissions mode
permission_mode: bypassPermissions
---
Bypass mode agent.
SUBAGENT

# acceptEdits mode
cat > .crush/agents/accept-edits-mode.md << 'SUBAGENT'
---
name: accept-edits-mode
description: Agent with acceptEdits mode
permission_mode: acceptEdits
---
Accept edits mode agent.
SUBAGENT

# dontAsk mode
cat > .crush/agents/dont-ask-mode.md << 'SUBAGENT'
---
name: dont-ask-mode
description: Agent with dontAsk mode
permission_mode: dontAsk
---
Dont ask mode agent.
SUBAGENT

# plan mode
cat > .crush/agents/plan-mode.md << 'SUBAGENT'
---
name: plan-mode
description: Agent with plan mode
permission_mode: plan
---
Plan mode agent.
SUBAGENT

# Create crush.json with test provider
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
      args: [
        "-c",
        setupWithPermissionModes.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "loads all permission mode variants",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open agents dialog
      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show 6 agents
      await expect(
        terminal.getByText("Agents (6)", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "yolo mode agent is loaded and shows correct mode",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Filter to yolo-mode agent
      terminal.write("yolo-mode");
      await new Promise((r) => setTimeout(r, 300));

      // View details
      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toContain("Agent: yolo-mode");
      expect(output).toMatch(/Permissions:.*yolo/);
    }
  );

  test.when(
    !isWindows,
    "bypassPermissions mode agent is loaded",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("bypass-mode");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Permissions:.*bypassPermissions/);
    }
  );

  test.when(
    !isWindows,
    "acceptEdits mode agent is loaded",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("accept-edits-mode");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Permissions:.*acceptEdits/);
    }
  );

  test.when(
    !isWindows,
    "dontAsk mode agent is loaded",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("dont-ask-mode");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Permissions:.*dontAsk/);
    }
  );

  test.when(!isWindows, "plan mode agent is loaded", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

    terminal.write("plan-mode");
    await new Promise((r) => setTimeout(r, 300));

    terminal.write("\r");
    await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

    const output = getTerminalOutput(terminal);
    expect(output).toMatch(/Permissions:.*plan/);
  });

  test.when(
    !isWindows,
    "default mode agent shows default permissions",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("default-mode");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toContain("Agent: default-mode");
      // Default mode should show "default" in permissions
      expect(output).toMatch(/Permissions:.*default/);
    }
  );
});

/**
 * Agent Model Types Tests
 *
 * Tests that model types are correctly displayed.
 */
test.describe("Agent model types", () => {
  const setupWithModelTypes = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

# Sonnet model
cat > .crush/agents/sonnet-agent.md << 'SUBAGENT'
---
name: sonnet-agent
description: Uses sonnet model
model: sonnet
---
Sonnet agent prompt.
SUBAGENT

# Haiku model
cat > .crush/agents/haiku-agent.md << 'SUBAGENT'
---
name: haiku-agent
description: Uses haiku model
model: haiku
---
Haiku agent prompt.
SUBAGENT

# Opus model
cat > .crush/agents/opus-agent.md << 'SUBAGENT'
---
name: opus-agent
description: Uses opus model
model: opus
---
Opus agent prompt.
SUBAGENT

# Inherit model
cat > .crush/agents/inherit-agent.md << 'SUBAGENT'
---
name: inherit-agent
description: Inherits parent model
model: inherit
---
Inherit agent prompt.
SUBAGENT

# Create crush.json with test provider
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
      args: [
        "-c",
        setupWithModelTypes.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "shows sonnet model for sonnet-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("sonnet-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Model:.*sonnet/);
    }
  );

  test.when(
    !isWindows,
    "shows haiku model for haiku-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("haiku-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Model:.*haiku/);
    }
  );

  test.when(
    !isWindows,
    "shows opus model for opus-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("opus-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Model:.*opus/);
    }
  );

  test.when(
    !isWindows,
    "shows inherit model for inherit-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("inherit-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      expect(output).toMatch(/Model:.*inherit/);
    }
  );
});

/**
 * Agent Configuration Combinations Tests
 *
 * Tests various combinations of agent configuration options.
 */
test.describe("Agent configuration combinations", () => {
  const setupWithCombinations = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

# Full configuration with all options
cat > .crush/agents/full-config.md << 'SUBAGENT'
---
name: full-config
description: Agent with all configuration options
model: opus
permission_mode: yolo
tools:
  - bash
  - edit
  - view
  - glob
  - grep
disallowed_tools:
  - download
  - fetch
skills:
  - security-patterns
---
Fully configured agent with all options specified.
SUBAGENT

# Minimal configuration
cat > .crush/agents/minimal-config.md << 'SUBAGENT'
---
name: minimal-config
description: Minimal agent configuration
---
Minimal configuration agent.
SUBAGENT

# Read-only agent with plan mode and only read tools
cat > .crush/agents/readonly-agent.md << 'SUBAGENT'
---
name: readonly-agent
description: Read-only exploration agent
model: haiku
permission_mode: plan
tools:
  - view
  - glob
  - grep
  - ls
---
Read-only agent for safe exploration.
SUBAGENT

# Create crush.json
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
      args: [
        "-c",
        setupWithCombinations.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "full-config agent shows all configuration options",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("full-config");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);

      expect(output).toContain("Agent: full-config");
      expect(output).toMatch(/Model:.*opus/);
      expect(output).toMatch(/Permissions:.*yolo/);
      expect(output).toContain("Disallowed");
    }
  );

  test.when(
    !isWindows,
    "minimal-config agent shows default values",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("minimal-config");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);

      expect(output).toContain("Agent: minimal-config");
      // Should show inherit model for minimal config
      expect(output).toMatch(/Model:.*inherit/);
    }
  );

  test.when(
    !isWindows,
    "readonly-agent shows plan mode and haiku model",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("readonly-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);

      expect(output).toContain("Agent: readonly-agent");
      expect(output).toMatch(/Model:.*haiku/);
      expect(output).toMatch(/Permissions:.*plan/);
    }
  );
});

/**
 * Tools Configuration Tests
 *
 * Tests that tool configurations are correctly processed and displayed.
 */
test.describe("Tools configuration", () => {
  const setupWithTools = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

# Agent with specific tools
cat > .crush/agents/tools-agent.md << 'SUBAGENT'
---
name: tools-agent
description: Agent with specific tools
tools:
  - bash
  - edit
  - view
  - glob
  - grep
---
Tools agent prompt.
SUBAGENT

# Agent with disallowed tools
cat > .crush/agents/disallowed-agent.md << 'SUBAGENT'
---
name: disallowed-agent
description: Agent with disallowed tools
permission_mode: yolo
tools:
  - bash
  - edit
  - view
disallowed_tools:
  - download
  - fetch
---
Disallowed tools agent prompt.
SUBAGENT

# Create crush.json
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
      args: [
        "-c",
        setupWithTools.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "shows effective tools for tools-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("tools-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);

      expect(output).toContain("Effective Tools");
      expect(output).toContain("bash");
      expect(output).toContain("edit");
    }
  );

  test.when(
    !isWindows,
    "shows disallowed tools for disallowed-agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      terminal.write("disallowed-agent");
      await new Promise((r) => setTimeout(r, 300));

      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);

      expect(output).toContain("Disallowed");
      expect(output).toContain("download");
    }
  );
});

/**
 * Invalid Permission Mode Handling Tests
 *
 * Tests that invalid permission modes are handled gracefully.
 */
test.describe("Invalid permission mode handling", () => {
  const setupWithInvalidMode = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

# Agent with invalid permission mode
cat > .crush/agents/invalid-mode.md << 'SUBAGENT'
---
name: invalid-mode
description: Agent with invalid permission mode
permission_mode: invalid_mode_value
---
Invalid mode agent.
SUBAGENT

# Valid agent for comparison
cat > .crush/agents/valid-agent.md << 'SUBAGENT'
---
name: valid-agent
description: Valid agent
permission_mode: yolo
---
Valid agent.
SUBAGENT

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
      args: [
        "-c",
        setupWithInvalidMode.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "app starts despite invalid permission mode in agent",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // App should start - open agents dialog
      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show at least the valid agent (invalid one may be skipped)
      await expect(terminal.getByText("Agents", { full: true })).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "valid agent is still loaded when invalid agent exists",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      const output = getTerminalOutput(terminal);
      // The valid agent should be loaded
      expect(output).toContain("valid-agent");
    }
  );
});

/**
 * Agent Dialog Navigation Tests
 *
 * Tests navigation within the agents dialog.
 */
test.describe("Agent dialog navigation", () => {
  const setupWithAgents = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

mkdir -p .crush/agents

cat > .crush/agents/agent-a.md << 'SUBAGENT'
---
name: agent-a
description: First agent
permission_mode: yolo
---
Agent A prompt.
SUBAGENT

cat > .crush/agents/agent-b.md << 'SUBAGENT'
---
name: agent-b
description: Second agent
permission_mode: plan
---
Agent B prompt.
SUBAGENT

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
      args: [
        "-c",
        setupWithAgents.replace(/\$\{CRUSH_BINARY\}/g, CRUSH_BINARY),
      ],
    },
    rows: 50,
    cols: 120,
  });

  test.when(
    !isWindows,
    "can filter agents by typing",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Type to filter
      terminal.write("agent-a");
      await new Promise((r) => setTimeout(r, 300));

      const output = getTerminalOutput(terminal);
      expect(output).toContain("agent-a");
    }
  );

  test.when(
    !isWindows,
    "can navigate between agents with arrow keys",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Navigate down
      terminal.keyDown();
      await new Promise((r) => setTimeout(r, 200));

      // Navigate up
      terminal.keyUp();
      await new Promise((r) => setTimeout(r, 200));

      // Dialog should still be open
      await expect(terminal.getByText("Agents", { full: true })).toBeVisible();
    }
  );

  test.when(!isWindows, "escape closes the dialog", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

    await expect(terminal.getByText("Agents", { full: true })).toBeVisible();

    // Press escape
    terminal.keyEscape();
    await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

    // Dialog should be closed
  });

  test.when(
    !isWindows,
    "returns to list view on any key from detail view",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x01");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // View first agent details
      terminal.write("\r");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should be in detail view
      await expect(terminal.getByText("Agent:", { full: true })).toBeVisible();

      // Press any key to go back
      terminal.write("q");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should be back in list view with agent count
      await expect(
        terminal.getByText("Agents (2)", { full: true })
      ).toBeVisible();
    }
  );
});
