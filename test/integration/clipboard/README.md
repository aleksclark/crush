# Clipboard Integration Tests

Tests for clipboard and PRIMARY selection functionality on X11 systems.

## How Clipboard Works in Crush

Crush uses **OSC 52** escape sequences for all clipboard operations. OSC 52 is a terminal
escape sequence that allows applications to read from and write to the system clipboard
through the terminal emulator (like Ghostty, iTerm2, kitty, etc.).

### Copy Operations

When you copy text in Crush (via selection, 'y' key, or 'c' key):
1. `tea.SetClipboard(text)` - writes to system CLIPBOARD via OSC 52
2. `tea.SetPrimaryClipboard(text)` - writes to PRIMARY selection via OSC 52

### Paste Operations

- **Ctrl+V**: Terminal sends bracketed paste with CLIPBOARD content → `tea.PasteMsg`
- **Middle-click**: Crush requests PRIMARY via `tea.ReadPrimaryClipboard()` → terminal responds with `tea.ClipboardMsg` → converted to `tea.PasteMsg`

### Why OSC 52?

OSC 52 has several advantages:
1. Works without requiring xsel/xclip to be installed
2. Works in containerized environments
3. Works over SSH (terminal handles clipboard access)
4. Supported by modern terminal emulators (Ghostty, kitty, iTerm2, etc.)

### Terminal Requirements

Your terminal must support OSC 52 for clipboard operations to work:
- **Ghostty**: Full OSC 52 support enabled by default
- **kitty**: Full OSC 52 support
- **iTerm2**: Enable "Allow clipboard access to terminal apps" in Preferences
- **Alacritty**: Full OSC 52 support
- **xterm**: Limited OSC 52 support (may need configuration)

## Running Tests

### Docker (for wrapper tests)

The Docker tests verify the `internal/clipboard` wrapper functions work correctly
with xsel/xclip. These serve as a fallback test but are not used in production.

```bash
docker build -f Dockerfile.clipboard-test -t crush-clipboard-test .
docker run --rm crush-clipboard-test sh -c 'Xvfb :99 -screen 0 1024x768x24 & sleep 3 && DISPLAY=:99 go test -v -timeout=20s ./test/integration/clipboard'
```

### Unit Tests

```bash
go test ./internal/tui/page/chat -v -run "Clipboard"
```

## X11 Clipboard Architecture

X11 has two independent clipboard selections:

1. **PRIMARY**: Traditional X11 selection
   - Set by: Highlighting text with mouse
   - Paste with: Middle-click
   - OSC 52: `tea.SetPrimaryClipboard()` / `tea.ReadPrimaryClipboard()`

2. **CLIPBOARD**: Modern clipboard (Ctrl+C/V)
   - Set by: Ctrl+C or explicit copy command
   - Paste with: Ctrl+V
   - OSC 52: `tea.SetClipboard()` / `tea.ReadClipboard()`

In Crush, we write to **both** selections on copy operations to maintain
compatibility with both X11 traditions and modern expectations.

## Historical Note: atotto/clipboard Bug

The `atotto/clipboard` library has a bug in its PRIMARY selection support on X11.
In `clipboard_unix.go:97-104`, it truncates command arguments when `clipboard.Primary = true`:

```go
pasteCmdArgs = pasteCmdArgs[:1]  // BUG: Truncates to just the command name!
```

This causes xsel/xclip to hang waiting for stdin input. We bypassed this issue
by using OSC 52 instead of native clipboard access.
