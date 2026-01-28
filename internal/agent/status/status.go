// Package status provides agent status reporting functionality.
// It writes the current agent state to a JSON file that external tools can read
// to understand what the agent is doing.
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State represents the current state of the agent.
type State string

const (
	// StateIdle indicates the agent is not processing any request.
	StateIdle State = "idle"
	// StateThinking indicates the agent is waiting for model response.
	StateThinking State = "thinking"
	// StateToolRunning indicates the agent is executing a tool.
	StateToolRunning State = "tool_running"
	// StateStreaming indicates the agent is streaming a response.
	StateStreaming State = "streaming"
)

// Status represents the current status of the agent, written to disk as JSON.
type Status struct {
	// State is the current agent state.
	State State `json:"state"`
	// SessionID is the ID of the current session being processed.
	SessionID string `json:"session_id,omitempty"`
	// Tool contains information about the currently running tool.
	Tool *ToolStatus `json:"tool,omitempty"`
	// UpdatedAt is when this status was last updated.
	UpdatedAt time.Time `json:"updated_at"`
	// StartedAt is when the current operation started.
	StartedAt *time.Time `json:"started_at,omitempty"`
}

// ToolStatus contains information about a running tool.
type ToolStatus struct {
	// Name is the name of the tool being executed.
	Name string `json:"name"`
	// ID is the tool call ID.
	ID string `json:"id"`
	// Description is an optional description of what the tool is doing.
	Description string `json:"description,omitempty"`
}

// Reporter writes agent status to a file.
type Reporter struct {
	mu       sync.Mutex
	filePath string
	enabled  bool
	current  Status
}

// NewReporter creates a new status reporter.
// If filePath is empty, status reporting is disabled.
func NewReporter(filePath string) *Reporter {
	return &Reporter{
		filePath: filePath,
		enabled:  filePath != "",
		current: Status{
			State:     StateIdle,
			UpdatedAt: time.Now(),
		},
	}
}

// NewReporterInDir creates a new status reporter that writes to a unique file
// in the given directory. The filename is based on the process ID.
func NewReporterInDir(dir string) *Reporter {
	if dir == "" {
		return &Reporter{enabled: false}
	}
	filePath := filepath.Join(dir, fmt.Sprintf("status-%d.json", os.Getpid()))
	return NewReporter(filePath)
}

// DefaultStatusPath returns the default path for the status file.
func DefaultStatusPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".crush", "status.json")
}

// SetIdle sets the agent state to idle.
func (r *Reporter) SetIdle() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.current = Status{
		State:     StateIdle,
		UpdatedAt: time.Now(),
	}
	return r.write()
}

// SetThinking sets the agent state to thinking.
func (r *Reporter) SetThinking(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.current = Status{
		State:     StateThinking,
		SessionID: sessionID,
		UpdatedAt: now,
		StartedAt: &now,
	}
	return r.write()
}

// SetStreaming sets the agent state to streaming.
func (r *Reporter) SetStreaming(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Preserve StartedAt if transitioning from thinking.
	startedAt := r.current.StartedAt
	if startedAt == nil {
		now := time.Now()
		startedAt = &now
	}

	r.current = Status{
		State:     StateStreaming,
		SessionID: sessionID,
		UpdatedAt: time.Now(),
		StartedAt: startedAt,
	}
	return r.write()
}

// SetToolRunning sets the agent state to tool running.
func (r *Reporter) SetToolRunning(sessionID, toolName, toolID, description string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Preserve StartedAt from the thinking state.
	startedAt := r.current.StartedAt
	if startedAt == nil {
		now := time.Now()
		startedAt = &now
	}

	r.current = Status{
		State:     StateToolRunning,
		SessionID: sessionID,
		Tool: &ToolStatus{
			Name:        toolName,
			ID:          toolID,
			Description: description,
		},
		UpdatedAt: time.Now(),
		StartedAt: startedAt,
	}
	return r.write()
}

// Current returns a copy of the current status.
func (r *Reporter) Current() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current
}

// write writes the current status to the file.
func (r *Reporter) write() error {
	if !r.enabled {
		return nil
	}

	// Ensure directory exists.
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.current, "", "  ")
	if err != nil {
		return err
	}

	// Write atomically by writing to a temp file and renaming.
	tmpPath := r.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, r.filePath)
}

// Close cleans up the status file by setting state to idle.
func (r *Reporter) Close() error {
	return r.SetIdle()
}
