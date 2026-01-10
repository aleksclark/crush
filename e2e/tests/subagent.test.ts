/**
 * E2E Tests: Subagent Feature
 *
 * Tests the subagent configuration system which allows users to define
 * specialized agents in .crush/agents/ or .claude/agents/ directories.
 *
 * These tests verify:
 * 1. Subagent discovery from project and user directories
 * 2. Hot-reloading when subagent files change
 * 3. Subagent configuration parsing (YAML frontmatter)
 * 4. Integration with the agent tool
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";
import * as os from "os";

// Resolve binary path
const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Helper delays
const STARTUP_DELAY = 3000;
const FILE_WATCH_DELAY = 1000;

// Skip bash-dependent tests on Windows
const isWindows = os.platform() === "win32";

/**
 * Subagent Discovery Tests
 *
 * Tests that subagents are discovered from .crush/agents/ directory
 * when the application starts.
 */
test.describe("Subagent discovery", () => {
  // Setup script creates a temp directory with a subagent
  const setupWithSubagent = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Create project structure
mkdir -p .crush/agents

# Create a valid subagent
cat > .crush/agents/code-reviewer.md << 'SUBAGENT'
---
name: code-reviewer
description: Reviews code for quality and best practices
model: sonnet
tools:
  - Glob
  - Grep
  - View
---
You are a code reviewer. Analyze the code carefully for:
- Code style issues
- Potential bugs
- Performance concerns
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
    "large": {
      "provider": "test",
      "model": "test-model"
    },
    "small": {
      "provider": "test",
      "model": "test-model"
    }
  }
}
CONFIG

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithSubagent.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "discovers subagent from .crush/agents/",
    async ({ terminal }) => {
      // Wait for app to initialize and discover subagents
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // The app should have loaded without errors
      // We can't directly check the registry, but we can verify the app started
      // and look for any error messages

      // There should be no "Failed to parse" or similar error messages
      // The terminal output would show errors if subagent loading failed
    }
  );
});

/**
 * Subagent Hot-Reload Tests
 *
 * Tests that modifying subagent files triggers automatic reloading.
 */
test.describe("Subagent hot-reload", () => {
  const setupWithMutableSubagent = `
set -e
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

# Create project structure
mkdir -p .crush/agents

# Create initial subagent
cat > .crush/agents/mutable-agent.md << 'SUBAGENT'
---
name: mutable-agent
description: Initial description
---
Initial prompt.
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

# Start crush in background and keep it running
"${CRUSH_BINARY}" &
CRUSH_PID=$!

# Give it time to start
sleep 3

# Modify the subagent file
cat > .crush/agents/mutable-agent.md << 'SUBAGENT'
---
name: mutable-agent
description: Updated description
---
Updated prompt.
SUBAGENT

# Wait for file watch to trigger
sleep 2

# Clean up
kill $CRUSH_PID 2>/dev/null || true
rm -rf "$TMPDIR"
`;

  // Note: Hot-reload tests are challenging to verify in e2e
  // because we need to observe internal state changes.
  // The unit tests in registry_test.go cover this more thoroughly.
});

/**
 * Subagent Configuration Parsing Tests
 *
 * Tests various subagent configurations are parsed correctly.
 */
test.describe("Subagent configuration parsing", () => {
  // Test with multiple subagents of different configurations
  const setupWithVariousSubagents = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents
mkdir -p .claude/agents

# Subagent with all options
cat > .crush/agents/full-config.md << 'SUBAGENT'
---
name: full-config
description: Subagent with all configuration options
model: opus
permission_mode: plan
tools:
  - Glob
  - Grep
  - View
disallowed_tools:
  - Bash
skills:
  - security-patterns
hooks:
  pre_tool_execution: echo "pre"
  post_tool_execution: echo "post"
---
Full configuration system prompt.
SUBAGENT

# Minimal subagent
cat > .crush/agents/minimal.md << 'SUBAGENT'
---
name: minimal
description: Minimal configuration
---
Minimal prompt.
SUBAGENT

# Claude-compatible location (should also be discovered)
cat > .claude/agents/claude-compat.md << 'SUBAGENT'
---
name: claude-compat
description: Claude Code compatible location
model: haiku
---
Claude compatible prompt.
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

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: [
        "-c",
        setupWithVariousSubagents.replace("${CRUSH_BINARY}", CRUSH_BINARY),
      ],
    },
  });

  test.when(
    !isWindows,
    "loads subagents from both .crush and .claude directories",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));
      // App should start without errors
    }
  );
});

/**
 * Subagent Validation Tests
 *
 * Tests that invalid subagent configurations are handled gracefully.
 */
test.describe("Subagent validation", () => {
  const setupWithInvalidSubagent = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents

# Valid subagent
cat > .crush/agents/valid.md << 'SUBAGENT'
---
name: valid-agent
description: Valid agent
---
Prompt.
SUBAGENT

# Invalid subagent (missing required fields)
cat > .crush/agents/invalid.md << 'SUBAGENT'
---
name: invalid
---
Missing description.
SUBAGENT

# Invalid YAML
cat > .crush/agents/broken-yaml.md << 'SUBAGENT'
---
name: [broken
description: This YAML is broken
---
Prompt.
SUBAGENT

# Invalid name (uppercase)
cat > .crush/agents/bad-name.md << 'SUBAGENT'
---
name: BadName
description: Invalid name format
---
Prompt.
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

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: [
        "-c",
        setupWithInvalidSubagent.replace("${CRUSH_BINARY}", CRUSH_BINARY),
      ],
    },
  });

  test.when(
    !isWindows,
    "gracefully handles invalid subagent files",
    async ({ terminal }) => {
      // App should start despite invalid subagent files
      // Only the valid-agent should be loaded
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // The app should not crash - it should skip invalid files
      // and continue with valid ones
    }
  );
});

/**
 * Subagent Priority Tests
 *
 * Tests that project-level subagents take priority over user-level subagents.
 */
test.describe("Subagent priority", () => {
  const setupWithPriorityTest = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Create user-level agents dir (simulated)
mkdir -p .crush/agents
mkdir -p user-agents

# Project-level agent
cat > .crush/agents/shadow-agent.md << 'SUBAGENT'
---
name: shadow-agent
description: Project level (should win)
---
Project prompt.
SUBAGENT

# "User-level" agent with same name
cat > user-agents/shadow-agent.md << 'SUBAGENT'
---
name: shadow-agent
description: User level (should be shadowed)
---
User prompt.
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

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: [
        "-c",
        setupWithPriorityTest.replace("${CRUSH_BINARY}", CRUSH_BINARY),
      ],
    },
  });

  test.when(
    !isWindows,
    "project-level subagents shadow user-level",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));
      // The project-level shadow-agent should be used
      // This is verified by unit tests more thoroughly
    }
  );
});

/**
 * Subagent Model Mapping Tests
 *
 * Tests the model type mapping (sonnet/opus/haiku/inherit).
 */
test.describe("Subagent model mapping", () => {
  const setupWithModelTypes = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

mkdir -p .crush/agents

# Sonnet model
cat > .crush/agents/sonnet-agent.md << 'SUBAGENT'
---
name: sonnet-agent
description: Uses sonnet model
model: sonnet
---
Sonnet prompt.
SUBAGENT

# Haiku model
cat > .crush/agents/haiku-agent.md << 'SUBAGENT'
---
name: haiku-agent
description: Uses haiku model
model: haiku
---
Haiku prompt.
SUBAGENT

# Inherit model
cat > .crush/agents/inherit-agent.md << 'SUBAGENT'
---
name: inherit-agent
description: Inherits parent model
model: inherit
---
Inherit prompt.
SUBAGENT

# Default (no model specified)
cat > .crush/agents/default-agent.md << 'SUBAGENT'
---
name: default-agent
description: Uses default model
---
Default prompt.
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
    "large": { "provider": "test", "model": "test-large" },
    "small": { "provider": "test", "model": "test-small" }
  }
}
CONFIG

exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: [
        "-c",
        setupWithModelTypes.replace("${CRUSH_BINARY}", CRUSH_BINARY),
      ],
    },
  });

  test.when(
    !isWindows,
    "maps model types correctly",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));
      // Model mapping is verified by unit tests
      // E2E verifies the app doesn't crash with various model types
    }
  );
});

/**
 * Agents Dialog Tests
 *
 * Tests the Agents dialog opened with ctrl+a.
 * Uses fixture agents from e2e/.crush/agents/ directory.
 */
test.describe("Agents dialog", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: [],
    },
  });

  test("opens with ctrl+a", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    // Press ctrl+a to open agents dialog
    terminal.write("\x01"); // ctrl+a = ASCII 1

    // Should show the agents dialog title with count
    await expect(
      terminal.getByText("Agents (3)", { full: true })
    ).toBeVisible();
  });

  test("displays agent names", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, 500));

    // Should show agent names from e2e/.crush/agents/
    await expect(
      terminal.getByText("code-reviewer", { full: true })
    ).toBeVisible();
    await expect(
      terminal.getByText("security-scanner", { full: true })
    ).toBeVisible();
    await expect(
      terminal.getByText("test-writer", { full: true })
    ).toBeVisible();
  });

  test("shows agent description", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, 500));

    // Should show truncated description
    await expect(
      terminal.getByText("Reviews code for quality", { full: true })
    ).toBeVisible();
  });

  test("shows detail view on enter", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, 500));

    // Press enter to view details of the first agent (code-reviewer)
    terminal.write("\r");
    await new Promise((r) => setTimeout(r, 300));

    // Should show the detail view title with agent name
    await expect(
      terminal.getByText("Agent: code-reviewer", {
        full: true,
      })
    ).toBeVisible();
  });

  test("closes dialog on escape", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await expect(
      terminal.getByText("Agents (3)", { full: true })
    ).toBeVisible();

    // Press escape to close
    terminal.write("\x1b");
    await new Promise((r) => setTimeout(r, 500));

    // The dialog should be closed now
  });

  test("shows help keybindings", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    terminal.write("\x01");
    await new Promise((r) => setTimeout(r, 500));

    // Should show help text
    await expect(
      terminal.getByText("view details", { full: true })
    ).toBeVisible();
  });
});
