/**
 * E2E Tests: Worktree Feature
 *
 * Tests the git worktree mode functionality which allows creating
 * isolated development sessions in separate git worktrees.
 *
 * These tests require a crush.json with worktree_mode: true in the
 * e2e directory.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";
import * as os from "os";

// Resolve binary path
const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Helper to wait for TUI to be ready - needs enough time for init dialog
const STARTUP_DELAY = 3000;

// Skip git repo tests on Windows (bash script)
const isWindows = os.platform() === "win32";

/**
 * Worktree Dialog Tests
 *
 * Tests the worktree creation dialog that appears when pressing ctrl+n
 * with worktree_mode enabled.
 *
 * Note: These tests run from the e2e directory which has crush.json
 * with worktree_mode: true configured.
 */
test.describe("Worktree dialog", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: [],
    },
  });

  test("opens on ctrl+n", async ({ terminal }) => {
    // Wait for TUI to initialize (project init dialog appears first)
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));

    // Press ctrl+n to open new session dialog
    terminal.write("\x0e"); // ctrl+n = ASCII 14

    // Should show the worktree dialog
    await expect(
      terminal.getByText("New Worktree Session", { full: true })
    ).toBeVisible();
  });

  test("shows dialog description", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");

    await expect(
      terminal.getByText("Enter a name for the git worktree branch", {
        full: true,
      })
    ).toBeVisible();
  });

  test("shows session name label", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");

    await expect(
      terminal.getByText("Session name:", { full: true })
    ).toBeVisible();
  });

  test("shows placeholder text", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");

    await expect(
      terminal.getByText("feature-name", { full: true })
    ).toBeVisible();
  });

  test("shows keybinding hints", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");

    await expect(
      terminal.getByText("create worktree", { full: true })
    ).toBeVisible();
    await expect(terminal.getByText("cancel", { full: true })).toBeVisible();
  });

  test("closes dialog on escape", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");

    // Verify dialog is open
    await expect(
      terminal.getByText("New Worktree Session", { full: true })
    ).toBeVisible();

    // Press escape to close
    terminal.keyEscape();
    await new Promise((r) => setTimeout(r, 500));

    // Dialog should be closed - check that project init dialog is back
    await expect(
      terminal.getByText("initialize this project", { full: true })
    ).toBeVisible();
  });

  test("shows error for empty name on submit", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");
    await new Promise((r) => setTimeout(r, 300));

    // Press enter without typing a name
    terminal.submit();
    await new Promise((r) => setTimeout(r, 300));

    // Should show error message
    await expect(
      terminal.getByText("Name is required", { full: true })
    ).toBeVisible();
  });

  test("accepts input in text field", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");
    await new Promise((r) => setTimeout(r, 300));

    // Type a branch name
    terminal.write("my-test-feature");
    await new Promise((r) => setTimeout(r, 200));

    // Should show typed text
    await expect(
      terminal.getByText("my-test-feature", { full: true })
    ).toBeVisible();
  });
});

/**
 * Worktree Creation Tests (in temp git repo)
 *
 * These tests create a temporary git repository to test actual worktree
 * creation. Uses a bash wrapper that:
 * 1. Creates a temp directory
 * 2. Initializes a git repo with an initial commit
 * 3. Creates crush.json with worktree_mode enabled
 * 4. Runs crush in that directory
 */
test.describe("Worktree creation in git repo", () => {
  // Build the bash script with the binary path embedded
  const setupAndRunCrush = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"
git init -q
git config user.email "test@test.com"
git config user.name "Test User"
echo "# Test Project" > README.md
git add README.md
git commit -q -m "Initial commit"
cat > crush.json << 'CRUSHCONFIG'
{
  "options": {
    "worktree_mode": true
  }
}
CRUSHCONFIG
exec "${CRUSH_BINARY}"
`;

  test.use({
    program: {
      file: "bash",
      args: ["-c", setupAndRunCrush.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test.when(
    !isWindows,
    "creates worktree with valid name",
    async ({ terminal }) => {
      // Wait for TUI to initialize
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));

      // Press ctrl+n to open worktree dialog
      terminal.write("\x0e");
      await new Promise((r) => setTimeout(r, 500));

      // Verify dialog opened
      await expect(
        terminal.getByText("New Worktree Session", { full: true })
      ).toBeVisible();

      // Type a valid branch name
      const branchName = `test-feature-${Date.now()}`;
      terminal.write(branchName);
      await new Promise((r) => setTimeout(r, 200));

      // Submit the form
      terminal.submit();
      await new Promise((r) => setTimeout(r, 2000));

      // Should show success message
      await expect(
        terminal.getByText("Worktree created", { full: true })
      ).toBeVisible();
    }
  );

  test.when(
    !isWindows,
    "sanitizes special characters in branch name",
    async ({ terminal }) => {
      await new Promise((r) => setTimeout(r, STARTUP_DELAY));
      terminal.write("\x0e");
      await new Promise((r) => setTimeout(r, 500));

      // Type a name with spaces (will be sanitized to hyphens)
      terminal.write("my new feature");
      await new Promise((r) => setTimeout(r, 200));

      terminal.submit();
      await new Promise((r) => setTimeout(r, 2000));

      // Should succeed (name gets sanitized)
      await expect(
        terminal.getByText("Worktree created", { full: true })
      ).toBeVisible();
    }
  );
});

/**
 * Worktree Error Handling Tests
 *
 * Tests error conditions when worktree creation fails.
 */
test.describe("Worktree error handling", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: [],
    },
  });

  test("shows error when not in git repo", async ({ terminal }) => {
    // When creating worktree from e2e/ (not a git repo root), should show error
    await new Promise((r) => setTimeout(r, STARTUP_DELAY));
    terminal.write("\x0e");
    await new Promise((r) => setTimeout(r, 300));

    terminal.write("test-feature");
    await new Promise((r) => setTimeout(r, 200));

    terminal.submit();
    await new Promise((r) => setTimeout(r, 1000));

    // Should show error about not being a git repository
    await expect(
      terminal.getByText("not a git repository", { full: true })
    ).toBeVisible();
  });
});
