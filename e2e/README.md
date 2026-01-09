# Crush E2E Tests

End-to-end tests for the Crush TUI application using [Microsoft's tui-test](https://github.com/microsoft/tui-test) framework.

## Requirements

- Node.js 16.6.0 - 20.x (tui-test does not support Node 21+)
- Go (for building the crush binary)

If you use `asdf`, the `.tool-versions` file in this directory will automatically select Node 20.

## Setup

```bash
# From the project root
task e2e:install

# Or manually
cd e2e && npm install
```

## Running Tests

```bash
# Run all E2E tests (builds crush first)
task e2e

# Run with tracing enabled (for debugging)
task e2e:trace

# Update snapshots
task e2e:update

# Run specific test file
task e2e -- tests/startup.test.ts

# Run tests matching a pattern
task e2e -- --grep "version"
```

## Test Structure

```
e2e/
├── tests/
│   ├── __snapshots__/     # Snapshot files for visual regression
│   ├── startup.test.ts    # Application startup and CLI args
│   ├── commands.test.ts   # Non-interactive commands (run, schema, etc.)
│   ├── snapshots.test.ts  # Visual regression tests
│   └── worktree.test.ts   # Git worktree feature tests
├── crush.json             # Config enabling worktree_mode for tests
├── tui-test.config.ts     # tui-test configuration
├── tsconfig.json          # TypeScript configuration
└── package.json           # Node.js dependencies
```

## Test Categories

### Startup Tests (`startup.test.ts`)
- Version and help flags
- Debug and yolo flags
- Dirs command output

### Command Tests (`commands.test.ts`)
- Run command help and errors
- Projects command output
- Schema command output

### Snapshot Tests (`snapshots.test.ts`)
- Help output visual regression
- Version output visual regression
- Run help visual regression

### Worktree Tests (`worktree.test.ts`)
- Worktree dialog opens on ctrl+n
- Dialog shows title, description, input field
- Error handling for empty name
- Error handling for non-git directories

## Writing New Tests

The key pattern with tui-test is that `test.use()` must be called at the `describe` level,
not inside individual tests:

```typescript
import { test, expect } from "@microsoft/tui-test";

const CRUSH_BINARY = "/absolute/path/to/crush";

test.describe("My feature", () => {
  // Configuration at describe level
  test.use({
    program: {
      file: CRUSH_BINARY,
      args: ["--help"],
    },
  });

  // Tests use the configuration above
  test("does something", async ({ terminal }) => {
    await expect(terminal.getByText("expected text", { full: true })).toBeVisible();
  });
});
```

### Key Points

1. **Use absolute paths** for the binary
2. **`test.use()` at describe level**, not inside tests
3. **Use `{ full: true }`** for text searches to match partial content
4. **Use snapshots** for complex output verification

### Sending Keyboard Input

```typescript
// Write text
terminal.write("hello");

// Submit (press Enter)
terminal.submit();

// Special keys
terminal.keyEscape();
terminal.keyUp();
terminal.keyDown();

// Control characters
terminal.write("\x0e"); // ctrl+n (ASCII 14)
terminal.write("\x03"); // ctrl+c (ASCII 3)
terminal.write("\x10"); // ctrl+p (ASCII 16)
```

### Running in a Custom Working Directory

tui-test doesn't support `cwd` in program options, but you can use a bash wrapper
to set up a custom environment:

```typescript
const CRUSH_BINARY = path.resolve(process.cwd(), "../crush");

// Create a bash script that sets up the environment and runs crush
const setupScript = `
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"
git init -q
git config user.email "test@test.com"
git config user.name "Test"
echo "# Test" > README.md
git add . && git commit -q -m "init"
exec "${CRUSH_BINARY}"  # Binary path embedded at test definition time
`;

test.describe("Tests in temp git repo", () => {
  test.use({
    program: {
      file: "bash",
      args: ["-c", setupScript.replace("${CRUSH_BINARY}", CRUSH_BINARY)],
    },
  });

  test("does something", async ({ terminal }) => {
    // Test runs in temp git repo
  });
});
```

## Configuration

The `tui-test.config.ts` file controls test behavior:

```typescript
import { defineConfig } from "@microsoft/tui-test";

export default defineConfig({
  retries: 2,        // Retry failed tests
  trace: false,      // Enable tracing (useful for debugging)
  timeout: 30000,    // Test timeout in ms
});
```

## Debugging

### Enable Tracing

```bash
task e2e:trace
```

Traces are saved to `e2e/tui-traces/` and can be replayed with:

```bash
cd e2e && npx tui-test show-trace tui-traces/<trace-file>
```

### Common Issues

1. **Node version mismatch**: Ensure you're using Node 16-20
2. **Binary not found**: Run `task build` first or use `task e2e` which builds automatically
3. **Timeout errors**: Increase timeout in test or `tui-test.config.ts`
4. **Text not found**: Use `{ full: true }` option for partial matches

## CI Integration

The E2E tests can be run in CI by ensuring:

1. Node.js 20.x is installed
2. Go is installed and can build the binary
3. A PTY is available (most CI systems support this)

Example GitHub Actions step:

```yaml
- name: Run E2E Tests
  run: |
    cd e2e && npm ci
    cd .. && task e2e
```

## Note on Test Coverage

The current test suite focuses on non-interactive commands and CLI flags. Testing
the interactive TUI components requires a different approach since they depend on
mock providers and database state. See the existing Go unit tests in
`internal/tui/` for component-level testing.
