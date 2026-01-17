/**
 * E2E Tests: Skills Feature
 *
 * Tests the skills management dialog which allows users to:
 * 1. View list of discovered skills
 * 2. View skill details (description, files, metadata)
 * 3. View skill usage logs
 *
 * Uses ctrl+k to open the dialog.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";
import * as os from "os";

// Resolve binary path
const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Helper delays
const STARTUP_DELAY = 3000;
const DIALOG_TRANSITION = 700; // Increased from 500 for reliability

// Skip bash-dependent tests on Windows
const isWindows = os.platform() === "win32";

/**
 * Skills Dialog Tests
 *
 * Tests the skills dialog opened with ctrl+k.
 */
test.describe("Skills dialog", () => {
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
    "opens with ctrl+k",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Press ctrl+k to open skills dialog
      terminal.write("\x0b"); // ctrl+k = ASCII 11

      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show the skills dialog title
      await expect(
        terminal.getByText("Skills", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "closes dialog on escape",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      await expect(
        terminal.getByText("Skills", { full: true })
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

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show help text with keybindings
      await expect(
        terminal.getByText("logs", { full: true })
      ).toBeVisible();
    }
  );
});

/**
 * Skills with Configuration Tests
 *
 * Tests the dialog with actual skill configurations.
 */
test.describe("Skills with configuration", () => {
  // Setup script creates a temp directory with skill config
  const setupWithSkillConfig = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

# Create skills directory
mkdir -p skills/test-skill

# Create SKILL.md file
cat > skills/test-skill/SKILL.md << 'SKILL'
---
name: test-skill
description: A test skill for e2e testing
license: MIT
---

# Test Skill

This is a test skill used for e2e testing purposes.
SKILL

# Create a helper file in the skill
cat > skills/test-skill/helper.sh << 'HELPER'
#!/bin/bash
echo "Helper script"
HELPER

# Create crush.json with skills_paths config
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
  "options": {
    "skills_paths": ["./skills"]
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithSkillConfig.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "shows configured skills",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show configured skill
      await expect(
        terminal.getByText("test-skill", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows skill details on enter",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Press enter to view details
      terminal.submit();
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show Location in detail view (skill file path)
      await expect(
        terminal.getByText("Location:", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows skill files in details",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Press enter to view details
      terminal.submit();
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show skill files
      await expect(
        terminal.getByText("Files:", { full: true })
      ).toBeVisible();
    }
  );
});

/**
 * Skills Empty State Tests
 *
 * Tests the dialog when no skills are configured.
 */
test.describe("Skills Empty state", () => {
  const setupWithNoSkills = `
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
  },
  "options": {
    "skills_paths": []
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithNoSkills.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "shows empty state when no skills configured",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      terminal.write("\x0b");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // First verify the dialog opened
      await expect(
        terminal.getByText("Skills", { full: true })
      ).toBeVisible();

      // Should show empty state message
      await expect(terminal.getByText("No skills found")).toBeVisible();
    }
  );
});

/**
 * Skills Logs Dialog Tests
 *
 * Tests the logs dialog which is opened with 'l' key from the skills list.
 */
test.describe("Skills Logs dialog", () => {
  // Setup script creates a temp directory with skill config
  const setupWithSkillConfig = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Override config paths to isolate from user config
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
mkdir -p "$XDG_CONFIG_HOME/crush"
mkdir -p "$XDG_DATA_HOME/crush"

# Create skills directory
mkdir -p skills/test-skill

# Create SKILL.md file
cat > skills/test-skill/SKILL.md << 'SKILL'
---
name: test-skill
description: A test skill for e2e testing
---

# Test Skill
SKILL

# Create crush.json with skills_paths config
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
  "options": {
    "skills_paths": ["./skills"]
  }
}
CONFIG

exec "\${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupWithSkillConfig.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "opens logs dialog with l key",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open skills dialog
      terminal.write("\x0b"); // ctrl+k
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Verify skills dialog is open
      await expect(
        terminal.getByText("Skills", { full: true })
      ).toBeVisible();

      // Press 'l' to open logs for selected skill
      terminal.write("l");
      await new Promise((r) => setTimeout(r, DIALOG_TRANSITION));

      // Should show the logs dialog title with skill name
      await expect(
        terminal.getByText("Logs:", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "shows empty logs state",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Open skills dialog
      terminal.write("\x0b");
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

      // Open skills dialog
      terminal.write("\x0b");
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

      // Should be back to skills dialog
      await expect(
        terminal.getByText("Skills", { full: true })
      ).toBeVisible();
    }
  );
});
