/**
 * E2E Tests: Application Startup and Basic Navigation
 *
 * Tests basic startup behavior, initial screen rendering, and fundamental
 * navigation through the Crush TUI application.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";

// Resolve binary path. CRUSH_BINARY env var can override for CI.
// Default assumes running from e2e/ directory with binary in parent.
const CRUSH_BINARY = process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

// Test: Version flag
test.describe("Version flag", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["--version"],
    },
  });

  test("displays version information", async ({ terminal }) => {
    await expect(terminal.getByText("version", { full: true })).toBeVisible();
  });
});

// Test: Help flag
test.describe("Help flag", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["--help"],
    },
  });

  test("displays help information", async ({ terminal }) => {
    // Check for unique strings in help output
    await expect(terminal.getByText("AI assistant", { full: true })).toBeVisible();
  });
});

// Test: Run command help
test.describe("Run command help", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["run", "--help"],
    },
  });

  test("shows run command usage", async ({ terminal }) => {
    await expect(terminal.getByText("Run a single", { full: true })).toBeVisible();
  });

  test("shows quiet flag option", async ({ terminal }) => {
    await expect(terminal.getByText("Hide spinner", { full: true })).toBeVisible();
  });
});

// Test: Run command without prompt
test.describe("Run command without prompt", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["run"],
    },
  });

  test("shows error for missing prompt", async ({ terminal }) => {
    await expect(terminal.getByText("No prompt provided", { full: true })).toBeVisible();
  });
});

// Test: Debug flag with help
test.describe("Debug flag", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["-d", "--help"],
    },
  });

  test("accepts debug flag", async ({ terminal }) => {
    await expect(terminal.getByText("USAGE", { full: true })).toBeVisible();
  });
});

// Test: Yolo flag with help
test.describe("Yolo flag", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["-y", "--help"],
    },
  });

  test("accepts yolo flag", async ({ terminal }) => {
    await expect(terminal.getByText("USAGE", { full: true })).toBeVisible();
  });
});

// Test: Dirs command
test.describe("Dirs command", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["dirs"],
    },
  });

  test("shows directory information", async ({ terminal }) => {
    await new Promise((r) => setTimeout(r, 1000));
    // dirs command should output something.
    await expect(terminal).toMatchSnapshot("dirs-output.txt");
  });
});
