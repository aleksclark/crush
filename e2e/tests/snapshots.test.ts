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

  test("matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("version", { full: true })).toBeVisible();
    await expect(terminal).toMatchSnapshot("version-output.txt");
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
