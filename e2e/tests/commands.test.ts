/**
 * E2E Tests: Non-Interactive Commands
 *
 * Tests commands that run without requiring interactive TUI mode.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";

const CRUSH_BINARY =
  process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Test: Run command with help
test.describe("Run command", () => {
  test.describe("help", () => {
    test.use({
      program: {
        file: CRUSH_BINARY,
        args: ["run", "--help"],
      },
    });

    test("shows run help", async ({ terminal }) => {
      await expect(
        terminal.getByText("non-interactive", { full: true })
      ).toBeVisible();
    });
  });

  test.describe("missing prompt", () => {
    test.use({
      program: {
        file: CRUSH_BINARY,
        args: ["run"],
      },
    });

    test("shows error", async ({ terminal }) => {
      await expect(
        terminal.getByText("No prompt provided", { full: true })
      ).toBeVisible();
    });
  });
});

// Test: Projects command
test.describe("Projects command", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["projects"],
    },
  });

  test("lists projects", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, 1000));
    // Should show the projects table header.
    await expect(terminal.getByText("Path", { full: true })).toBeVisible();
  });
});

// Test: Schema command
test.describe("Schema command", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["schema"],
    },
  });

  test("outputs JSON schema", async ({ terminal }) => {
    // Wait for schema output to complete.
    await new Promise((r) => setTimeout(r, 3000));
    // Verify it outputs JSON schema. The schema contains "$id".
    await expect(terminal.getByText("$id", { full: true })).toBeVisible();
  });
});
