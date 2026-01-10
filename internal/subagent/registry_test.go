package subagent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	paths := []string{"/path1", "/path2"}
	r := NewRegistry(paths)

	require.NotNil(t, r)
	require.Equal(t, paths, r.WatchPaths())
	require.Equal(t, 0, r.Count())
}

func TestRegistryStartStop(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	// Create a test agent.
	agent := `---
name: test-agent
description: Test agent
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte(agent), 0o644))

	r := NewRegistry([]string{agentsDir})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := r.Start(ctx)
	require.NoError(t, err)

	// Verify agent was discovered.
	require.Equal(t, 1, r.Count())

	s, ok := r.Get("test-agent")
	require.True(t, ok)
	require.Equal(t, "test-agent", s.Name)

	// Stop the registry.
	err = r.Stop()
	require.NoError(t, err)
}

func TestRegistryGet(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent := `---
name: get-test
description: Get test agent
tools:
  - Glob
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "get-test.md"), []byte(agent), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	// Get existing agent.
	s, ok := r.Get("get-test")
	require.True(t, ok)
	require.Equal(t, "get-test", s.Name)
	require.Equal(t, []string{"Glob"}, s.Tools)

	// Get non-existing agent.
	_, ok = r.Get("nonexistent")
	require.False(t, ok)
}

func TestRegistryGetReturnsClone(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent := `---
name: clone-test
description: Clone test
tools:
  - Glob
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "clone.md"), []byte(agent), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	// Get and modify.
	s1, _ := r.Get("clone-test")
	s1.Tools = []string{"Modified"}

	// Get again and verify original is unchanged.
	s2, _ := r.Get("clone-test")
	require.Equal(t, []string{"Glob"}, s2.Tools)
}

func TestRegistryList(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent1 := `---
name: agent1
description: Agent 1
---
Prompt.
`
	agent2 := `---
name: agent2
description: Agent 2
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agent1.md"), []byte(agent1), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agent2.md"), []byte(agent2), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	agents := r.List()
	require.Len(t, agents, 2)

	names := make(map[string]bool)
	for _, a := range agents {
		names[a.Name] = true
	}
	require.True(t, names["agent1"])
	require.True(t, names["agent2"])
}

func TestRegistryNames(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent := `---
name: names-test
description: Names test
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "test.md"), []byte(agent), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	names := r.Names()
	require.Len(t, names, 1)
	require.Equal(t, "names-test", names[0])
}

func TestRegistryReload(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agent1 := `---
name: reload-agent
description: Original
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "reload.md"), []byte(agent1), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	// Verify initial state.
	s, _ := r.Get("reload-agent")
	require.Equal(t, "Original", s.Description)

	// Add a new agent file.
	agent2 := `---
name: new-agent
description: New agent
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "new.md"), []byte(agent2), 0o644))

	// Reload.
	err := r.Reload()
	require.NoError(t, err)

	// Verify new agent is found.
	require.Equal(t, 2, r.Count())
	_, ok := r.Get("new-agent")
	require.True(t, ok)
}

func TestRegistrySubscribe(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil)

	ch := r.Subscribe()
	require.NotNil(t, ch)

	// Send an event.
	r.publish(Event{Type: EventReloaded})

	// Receive event.
	select {
	case event := <-ch:
		require.Equal(t, EventReloaded, event.Type)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestRegistryUnsubscribe(t *testing.T) {
	t.Parallel()

	r := NewRegistry(nil)

	ch := r.Subscribe()
	r.Unsubscribe(ch)

	// Channel should be closed.
	_, ok := <-ch
	require.False(t, ok)
}

func TestRegistryFileWatching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file watching test in short mode")
	}
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	r := NewRegistry([]string{agentsDir})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	// Subscribe to events.
	events := r.Subscribe()

	// Create a new agent file.
	agent := `---
name: watched-agent
description: Watched agent
---
Prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "watched.md"), []byte(agent), 0o644))

	// Wait for the event (with debouncing delay).
	select {
	case event := <-events:
		require.Equal(t, EventAdded, event.Type)
		require.Equal(t, "watched-agent", event.Subagent.Name)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for add event")
	}

	// Verify agent is in registry.
	_, ok := r.Get("watched-agent")
	require.True(t, ok)

	// Modify the agent file.
	modifiedAgent := `---
name: watched-agent
description: Modified description
---
Modified prompt.
`
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "watched.md"), []byte(modifiedAgent), 0o644))

	// Wait for update event.
	select {
	case event := <-events:
		require.Equal(t, EventUpdated, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update event")
	}

	// Delete the agent file.
	require.NoError(t, os.Remove(filepath.Join(agentsDir, "watched.md")))

	// Wait for remove event.
	select {
	case event := <-events:
		require.Equal(t, EventRemoved, event.Type)
		require.Equal(t, "watched-agent", event.Subagent.Name)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for remove event")
	}

	// Verify agent is no longer in registry.
	_, ok = r.Get("watched-agent")
	require.False(t, ok)
}

func TestRegistryEnsureWatchDirs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "agents")

	r := NewRegistry([]string{newDir})

	// Directory doesn't exist.
	_, err := os.Stat(newDir)
	require.True(t, os.IsNotExist(err))

	// Ensure creates it.
	err = r.EnsureWatchDirs()
	require.NoError(t, err)

	// Directory now exists.
	info, err := os.Stat(newDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestRegistryGetByFilePath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	agentPath := filepath.Join(agentsDir, "path-test.md")
	agent := `---
name: path-test
description: Path test
---
Prompt.
`
	require.NoError(t, os.WriteFile(agentPath, []byte(agent), 0o644))

	r := NewRegistry([]string{agentsDir})
	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	// Get by file path.
	name, ok := r.GetByFilePath(agentPath)
	require.True(t, ok)
	require.Equal(t, "path-test", name)

	// Non-existent path.
	_, ok = r.GetByFilePath("/nonexistent/path.md")
	require.False(t, ok)
}

func TestRegistryCount(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	r := NewRegistry([]string{agentsDir})
	require.Equal(t, 0, r.Count())

	// Add agents and reload.
	for i := range 3 {
		agent := `---
name: count-agent-` + string(rune('a'+i)) + `
description: Count agent
---
Prompt.
`
		require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "agent"+string(rune('a'+i))+".md"), []byte(agent), 0o644))
	}

	ctx := context.Background()
	require.NoError(t, r.Start(ctx))
	defer r.Stop()

	require.Equal(t, 3, r.Count())
}
