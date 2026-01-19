// Package agentstatus implements the Agent Status Reporting Standard for
// filesystem-based status reporting. Agents write JSON files to a shared
// directory that can be read by monitors to display agent activity.
package agentstatus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Status represents the current state of the agent.
type Status string

const (
	StatusIdle     Status = "idle"     // Waiting for user input.
	StatusThinking Status = "thinking" // Processing/reasoning (no tool calls yet).
	StatusWorking  Status = "working"  // Actively executing tools.
	StatusWaiting  Status = "waiting"  // Waiting for external resource.
	StatusError    Status = "error"    // Agent encountered an error.
	StatusDone     Status = "done"     // Agent completed its task.
	StatusPaused   Status = "paused"   // Agent is paused by user.
)

// Tools contains tool usage information.
type Tools struct {
	Active *string          `json:"active"` // Currently executing tool (null if none).
	Recent []string         `json:"recent"` // Last N tools used (most recent last), max 10.
	Counts map[string]int64 `json:"counts"` // Map of tool name to invocation count.
}

// Tokens contains token usage counters.
type Tokens struct {
	Input      int64 `json:"input"`       // Total input tokens consumed.
	Output     int64 `json:"output"`      // Total output tokens generated.
	CacheRead  int64 `json:"cache_read"`  // Tokens read from cache (Anthropic).
	CacheWrite int64 `json:"cache_write"` // Tokens written to cache (Anthropic).
}

// AgentStatus represents the JSON structure written to the status file.
type AgentStatus struct {
	Version  int     `json:"v"`                  // Schema version (currently 1).
	Agent    string  `json:"agent"`              // Agent type identifier.
	Instance string  `json:"instance"`           // Unique instance identifier.
	PID      int     `json:"pid,omitempty"`      // Process ID of the agent.
	Project  string  `json:"project,omitempty"`  // Project/repo name.
	CWD      string  `json:"cwd,omitempty"`      // Current working directory.
	Status   Status  `json:"status"`             // Current status.
	Task     string  `json:"task,omitempty"`     // Human-readable current task.
	Model    string  `json:"model,omitempty"`    // AI model identifier.
	Provider string  `json:"provider,omitempty"` // API provider.
	Tools    *Tools  `json:"tools,omitempty"`    // Tool usage information.
	Tokens   *Tokens `json:"tokens,omitempty"`   // Token usage counters.
	CostUSD  float64 `json:"cost_usd,omitempty"` // Estimated cost in USD.
	Started  int64   `json:"started,omitempty"`  // Unix timestamp when session started.
	Updated  int64   `json:"updated"`            // Unix timestamp of last update.
	Error    string  `json:"error,omitempty"`    // Error message (when status is error).
}

// Reporter handles writing agent status to the filesystem.
type Reporter struct {
	mu       sync.Mutex
	dir      string
	instance string
	filePath string
	status   AgentStatus
	closed   bool
}

// NewReporter creates a new status reporter. If dir is empty, status reporting
// is disabled. The dir can use $AGENT_STATUS_DIR or default to ~/.agent-status.
func NewReporter(dir string) (*Reporter, error) {
	if dir == "" {
		return &Reporter{closed: true}, nil
	}

	// Expand ~ to home directory.
	if len(dir) > 0 && dir[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, dir[1:])
	}

	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	// Generate a unique instance ID (6 random hex chars).
	instanceBytes := make([]byte, 3)
	if _, err := rand.Read(instanceBytes); err != nil {
		return nil, err
	}
	instance := hex.EncodeToString(instanceBytes)

	r := &Reporter{
		dir:      dir,
		instance: instance,
		filePath: filepath.Join(dir, "crush-"+instance+".json"),
		status: AgentStatus{
			Version:  1,
			Agent:    "crush",
			Instance: instance,
			PID:      os.Getpid(),
			Status:   StatusIdle,
			Started:  time.Now().Unix(),
			Updated:  time.Now().Unix(),
			Tools: &Tools{
				Recent: []string{},
				Counts: make(map[string]int64),
			},
			Tokens: &Tokens{},
		},
	}

	// Write initial status.
	if err := r.write(); err != nil {
		return nil, err
	}

	return r, nil
}

// SetProject sets the project name and working directory.
func (r *Reporter) SetProject(project, cwd string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Project = project
	r.status.CWD = cwd
	r.write()
}

// SetModel sets the model and provider.
func (r *Reporter) SetModel(model, provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Model = model
	r.status.Provider = provider
	r.write()
}

// SetStatus updates the agent status.
func (r *Reporter) SetStatus(status Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Status = status
	r.write()
}

// SetTask updates the current task description.
func (r *Reporter) SetTask(task string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Task = task
	r.write()
}

// SetError sets the error status with a message.
func (r *Reporter) SetError(errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Status = StatusError
	r.status.Error = errMsg
	r.write()
}

// ToolStart marks a tool as currently active.
func (r *Reporter) ToolStart(toolName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Status = StatusWorking
	r.status.Tools.Active = &toolName
	r.write()
}

// ToolEnd marks a tool as completed and updates counts.
func (r *Reporter) ToolEnd(toolName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Tools.Active = nil
	r.status.Tools.Counts[toolName]++

	// Add to recent, keeping max 10.
	r.status.Tools.Recent = append(r.status.Tools.Recent, toolName)
	if len(r.status.Tools.Recent) > 10 {
		r.status.Tools.Recent = r.status.Tools.Recent[len(r.status.Tools.Recent)-10:]
	}
	r.write()
}

// UpdateTokens updates token usage counters.
func (r *Reporter) UpdateTokens(input, output, cacheRead, cacheWrite int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.Tokens.Input = input
	r.status.Tokens.Output = output
	r.status.Tokens.CacheRead = cacheRead
	r.status.Tokens.CacheWrite = cacheWrite
	r.write()
}

// UpdateCost updates the cost estimate.
func (r *Reporter) UpdateCost(cost float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.status.CostUSD = cost
	r.write()
}

// Close removes the status file and marks the reporter as closed.
func (r *Reporter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return os.Remove(r.filePath)
}

// write atomically writes the status to the file.
func (r *Reporter) write() error {
	r.status.Updated = time.Now().Unix()

	data, err := json.Marshal(r.status)
	if err != nil {
		return err
	}

	// Write to temp file first, then rename for atomicity.
	tmpFile := filepath.Join(r.dir, ".tmp.crush-"+r.instance+".json")
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpFile, r.filePath)
}

// IsEnabled returns true if status reporting is enabled.
func (r *Reporter) IsEnabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.closed
}
