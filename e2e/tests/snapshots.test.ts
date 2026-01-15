/**
 * E2E Tests: Visual Snapshots
 *
 * Tests that capture terminal output snapshots for visual regression.
 */

import { test, expect } from "@microsoft/tui-test";
import * as path from "path";

const CRUSH_BINARY = process.env.CRUSH_BINARY || path.resolve(process.cwd(), "../crush");

test.describe("Help output snapshot", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["--help"],
    },
    rows: 40,
    cols: 100,
  });

  test("matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("USAGE", { full: true })).toBeVisible();
    await expect(terminal).toMatchSnapshot("help-output.txt");
  });
});

test.describe("Version output snapshot", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["--version"],
    },
    rows: 20,
    cols: 80,
  });

  test("matches version format", async ({ terminal }) => {
    // Wait for version text to be visible
    await expect(terminal.getByText("version", { full: true })).toBeVisible();
    
    // Check that the version matches the expected format: v0.32.2-0.20260115130311-861e8fb77b67+dirty
    // Pattern: v<semver>-<timestamp>-<commit>+<dirty>?
    const buffer = terminal.getBuffer();
    const output = buffer.map(row => row.join("")).join("\n");
    expect(output).toMatch(/crush version v\d+\.\d+\.\d+-\d+\.\d+-[a-f0-9]+(\+dirty)?/);
  });
});

test.describe("Run help snapshot", () => {
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["run", "--help"],
    },
    rows: 40,
    cols: 100,
  });

  test("matches snapshot", async ({ terminal }) => {
    await expect(
      terminal.getByText("non-interactive", { full: true })
    ).toBeVisible();
    await expect(terminal).toMatchSnapshot("run-help.txt");
  });
});
