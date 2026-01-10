package subagent

import (
	"sync"
	"time"
)

// debouncer prevents rapid-fire function calls by waiting for a quiet period.
type debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

// newDebouncer creates a new debouncer with the specified duration.
func newDebouncer(d time.Duration) *debouncer {
	return &debouncer{duration: d}
}

// Call schedules the function to be called after the debounce duration.
// If Call is invoked again before the duration elapses, the timer is reset.
func (d *debouncer) Call(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.duration, f)
}

// Stop cancels any pending calls.
func (d *debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
