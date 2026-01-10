package subagent

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType identifies the type of registry event.
type EventType string

const (
	// EventAdded is emitted when a new subagent is discovered.
	EventAdded EventType = "added"
	// EventUpdated is emitted when an existing subagent is modified.
	EventUpdated EventType = "updated"
	// EventRemoved is emitted when a subagent is deleted.
	EventRemoved EventType = "removed"
	// EventReloaded is emitted when all subagents are reloaded.
	EventReloaded EventType = "reloaded"
	// EventError is emitted when an error occurs.
	EventError EventType = "error"
)

// Event represents a change in the subagent registry.
type Event struct {
	Type     EventType
	Subagent *Subagent
	Error    error
}

// Registry manages subagent definitions with support for hot-reloading.
type Registry struct {
	mu         sync.RWMutex
	subagents  map[string]*Subagent
	byFilePath map[string]string // filePath -> name mapping.

	watchPaths []string
	watcher    *fsnotify.Watcher
	debouncer  *debouncer

	subscribers    []chan Event
	subscribersMap map[<-chan Event]chan Event // For unsubscribe lookup.
	subMu          sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewRegistry creates a new subagent registry.
func NewRegistry(watchPaths []string) *Registry {
	return &Registry{
		subagents:  make(map[string]*Subagent),
		byFilePath: make(map[string]string),
		watchPaths: watchPaths,
		debouncer:  newDebouncer(300 * time.Millisecond),
	}
}

// Start initializes the registry and begins watching for changes.
func (r *Registry) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Initial discovery.
	if err := r.Reload(); err != nil {
		slog.Warn("Initial subagent discovery failed", "error", err)
	}

	// Setup file watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	r.watcher = watcher

	// Watch directories.
	for _, path := range r.watchPaths {
		if err := r.watchPath(path); err != nil {
			slog.Debug("Failed to watch path", "path", path, "error", err)
		}
	}

	// Start watch loop.
	go r.watchLoop()

	return nil
}

// Stop shuts down the registry and stops watching for changes.
func (r *Registry) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.watcher != nil {
		return r.watcher.Close()
	}
	return nil
}

// watchPath adds a path to the watcher, creating it if necessary.
func (r *Registry) watchPath(path string) error {
	// Create directory if it doesn't exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Don't create, just skip.
		return nil
	}

	if err := r.watcher.Add(path); err != nil {
		return err
	}

	slog.Debug("Watching path for subagents", "path", path)
	return nil
}

// watchLoop handles file system events.
func (r *Registry) watchLoop() {
	for {
		select {
		case <-r.ctx.Done():
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// Only process .md files.
			if !strings.HasSuffix(event.Name, ".md") {
				continue
			}

			// Debounce rapid changes (e.g., editor save operations).
			r.debouncer.Call(func() {
				r.handleFileChange(event)
			})

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "error", err)
			r.publish(Event{Type: EventError, Error: err})
		}
	}
}

// handleFileChange processes a file system change event.
func (r *Registry) handleFileChange(event fsnotify.Event) {
	path := event.Name
	slog.Debug("File change detected", "path", path, "op", event.Op.String())

	switch {
	case event.Op&fsnotify.Create != 0 || event.Op&fsnotify.Write != 0:
		r.reloadFile(path)
	case event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0:
		r.removeFile(path)
	}
}

// reloadFile parses and updates a single subagent file.
func (r *Registry) reloadFile(path string) {
	subagent, err := Parse(path)
	if err != nil {
		slog.Warn("Failed to parse subagent file", "path", path, "error", err)
		r.publish(Event{Type: EventError, Error: err})
		return
	}

	if err := subagent.Validate(); err != nil {
		slog.Warn("Subagent validation failed", "path", path, "error", err)
		r.publish(Event{Type: EventError, Error: err})
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if this is an update or add.
	existing, isUpdate := r.subagents[subagent.Name]

	// Update mappings.
	r.subagents[subagent.Name] = subagent
	r.byFilePath[path] = subagent.Name

	if isUpdate {
		slog.Info("Subagent updated", "name", subagent.Name, "path", path)
		r.publish(Event{Type: EventUpdated, Subagent: existing})
	} else {
		slog.Info("Subagent added", "name", subagent.Name, "path", path)
		r.publish(Event{Type: EventAdded, Subagent: subagent})
	}
}

// removeFile removes a subagent when its file is deleted.
func (r *Registry) removeFile(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name, ok := r.byFilePath[path]
	if !ok {
		return
	}

	subagent := r.subagents[name]
	delete(r.subagents, name)
	delete(r.byFilePath, path)

	slog.Info("Subagent removed", "name", name, "path", path)
	r.publish(Event{Type: EventRemoved, Subagent: subagent})
}

// Reload rediscovers all subagents from the configured paths.
func (r *Registry) Reload() error {
	subagents := Discover(r.watchPaths)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing.
	r.subagents = make(map[string]*Subagent, len(subagents))
	r.byFilePath = make(map[string]string, len(subagents))

	// Add discovered subagents.
	for _, s := range subagents {
		r.subagents[s.Name] = s
		r.byFilePath[s.FilePath] = s.Name
	}

	slog.Info("Subagents reloaded", "count", len(subagents))
	r.publish(Event{Type: EventReloaded})

	return nil
}

// Get returns a subagent by name.
func (r *Registry) Get(name string) (*Subagent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.subagents[name]
	if !ok {
		return nil, false
	}
	return s.Clone(), true
}

// List returns all registered subagents.
func (r *Registry) List() []*Subagent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Subagent, 0, len(r.subagents))
	for _, s := range r.subagents {
		result = append(result, s.Clone())
	}
	return result
}

// Names returns the names of all registered subagents.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.subagents))
	for name := range r.subagents {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered subagents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.subagents)
}

// Subscribe returns a channel that receives registry events.
func (r *Registry) Subscribe() <-chan Event {
	ch := make(chan Event, 16)
	r.subMu.Lock()
	r.subscribers = append(r.subscribers, ch)
	if r.subscribersMap == nil {
		r.subscribersMap = make(map[<-chan Event]chan Event)
	}
	r.subscribersMap[ch] = ch
	r.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (r *Registry) Unsubscribe(ch <-chan Event) {
	r.subMu.Lock()
	defer r.subMu.Unlock()

	original, ok := r.subscribersMap[ch]
	if !ok {
		return
	}

	for i, sub := range r.subscribers {
		if sub == original {
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			close(sub)
			delete(r.subscribersMap, ch)
			return
		}
	}
}

// publish sends an event to all subscribers.
func (r *Registry) publish(event Event) {
	r.subMu.RLock()
	defer r.subMu.RUnlock()

	for _, ch := range r.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip.
			slog.Debug("Dropping event, subscriber channel full")
		}
	}
}

// WatchPaths returns the configured watch paths.
func (r *Registry) WatchPaths() []string {
	return r.watchPaths
}

// EnsureWatchDirs creates the watch directories if they don't exist.
func (r *Registry) EnsureWatchDirs() error {
	for _, path := range r.watchPaths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}

		// Add to watcher if running.
		if r.watcher != nil {
			if err := r.watchPath(path); err != nil {
				slog.Warn("Failed to watch new path", "path", path, "error", err)
			}
		}
	}
	return nil
}

// GetByFilePath returns the subagent name for a given file path.
func (r *Registry) GetByFilePath(filePath string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalize path.
	filePath = filepath.Clean(filePath)

	name, ok := r.byFilePath[filePath]
	return name, ok
}
