# Clipboard Integration Tests

Tests for clipboard and PRIMARY selection functionality on X11 systems.

## Running Tests

### Docker (Recommended)

Run tests in Docker with X11 support:

```bash
docker build -f Dockerfile.clipboard-test -t crush-clipboard-test .
docker run --rm crush-clipboard-test sh -c 'Xvfb :99 -screen 0 1024x768x24 & sleep 3 && DISPLAY=:99 go test -v -timeout=20s ./test/integration/clipboard'
```

### Local (requires X11)

```bash
DISPLAY=:0 go test -v ./test/integration/clipboard
```

## Test Coverage

- **TestXselWorks**: Verifies xsel can read/write both selections
- **TestReadPrimary**: Tests reading from PRIMARY selection
- **TestWritePrimary**: Tests writing to PRIMARY selection
- **TestWriteBoth**: Tests writing to both CLIPBOARD and PRIMARY
- **TestPrimaryAndClipboardIndependent**: Verifies selections are independent

## Known Issues

The `atotto/clipboard` library's PRIMARY selection support is broken - it
truncates command arguments when `clipboard.Primary = true`, causing xsel/xclip
to hang waiting for input. Our `internal/clipboard` wrapper bypasses this by
calling xsel/xclip directly.
