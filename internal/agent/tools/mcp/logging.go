// Package mcp provides logging functionality for MCP tool invocations.
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogEntry represents a single MCP invocation log entry.
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	SessionID   string    `json:"session_id"`
	ServerName  string    `json:"server_name"`
	ToolName    string    `json:"tool_name"`
	Input       string    `json:"input"`
	Output      string    `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	Success     bool      `json:"success"`
	ContentType string    `json:"content_type,omitempty"`
}

// LogSession represents a unique session of MCP invocations.
type LogSession struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
}

var (
	loggers   = make(map[string]*lumberjack.Logger)
	loggersMu sync.RWMutex

	currentSessionID string
	sessionMu        sync.RWMutex
)

// SetSessionID sets the current session ID for logging.
func SetSessionID(sessionID string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	currentSessionID = sessionID
}

// GetSessionID returns the current session ID.
func GetSessionID() string {
	sessionMu.RLock()
	defer sessionMu.RUnlock()
	return currentSessionID
}

// getLogger returns or creates a logger for the specified MCP server.
func getLogger(serverName string) *lumberjack.Logger {
	loggersMu.Lock()
	defer loggersMu.Unlock()

	if logger, ok := loggers[serverName]; ok {
		return logger
	}

	cfg := config.Get()
	logsDir := filepath.Join(cfg.Options.DataDirectory, "logs", "mcp")

	// Ensure the directory exists.
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil
	}

	logger := &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, fmt.Sprintf("%s.log", sanitizeFilename(serverName))),
		MaxSize:    5, // Max size in MB
		MaxBackups: 3,
		MaxAge:     7,    // Days
		Compress:   true, // Enable compression
	}

	loggers[serverName] = logger
	return logger
}

// LogInvocation logs an MCP tool invocation.
func LogInvocation(entry LogEntry) {
	if entry.SessionID == "" {
		entry.SessionID = GetSessionID()
	}

	logger := getLogger(entry.ServerName)
	if logger == nil {
		return
	}

	// Truncate large outputs to prevent log bloat.
	const maxOutputLen = 10000
	if len(entry.Output) > maxOutputLen {
		entry.Output = entry.Output[:maxOutputLen] + "... (truncated)"
	}
	if len(entry.Input) > maxOutputLen {
		entry.Input = entry.Input[:maxOutputLen] + "... (truncated)"
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = logger.Write(append(data, '\n'))
}

// GetLogEntries reads log entries for a specific MCP server, optionally filtered
// by session ID.
func GetLogEntries(serverName, sessionID string, limit int) ([]LogEntry, error) {
	cfg := config.Get()
	logFile := filepath.Join(
		cfg.Options.DataDirectory,
		"logs",
		"mcp",
		fmt.Sprintf("%s.log", sanitizeFilename(serverName)),
	)

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var entries []LogEntry
	lines := strings.Split(string(data), "\n")

	// Read from end to get most recent entries first.
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Filter by session ID if specified.
		if sessionID != "" && entry.SessionID != sessionID {
			continue
		}

		entries = append(entries, entry)

		if limit > 0 && len(entries) >= limit {
			break
		}
	}

	return entries, nil
}

// GetSessions returns a list of unique sessions from the log file for a server.
func GetSessions(serverName string) ([]LogSession, error) {
	cfg := config.Get()
	logFile := filepath.Join(
		cfg.Options.DataDirectory,
		"logs",
		"mcp",
		fmt.Sprintf("%s.log", sanitizeFilename(serverName)),
	)

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogSession{}, nil
		}
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	sessionMap := make(map[string]*LogSession)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.SessionID == "" {
			continue
		}

		if session, ok := sessionMap[entry.SessionID]; ok {
			// Update end time.
			if entry.Timestamp.After(session.EndTime) {
				session.EndTime = entry.Timestamp
			}
		} else {
			sessionMap[entry.SessionID] = &LogSession{
				ID:        entry.SessionID,
				StartTime: entry.Timestamp,
				EndTime:   entry.Timestamp,
			}
		}
	}

	sessions := make([]LogSession, 0, len(sessionMap))
	for _, session := range sessionMap {
		sessions = append(sessions, *session)
	}

	// Sort by start time (most recent first).
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].StartTime.After(sessions[i].StartTime) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	return sessions, nil
}

// GetAllServerLogs returns the names of all MCP servers that have log files.
func GetAllServerLogs() ([]string, error) {
	cfg := config.Get()
	logsDir := filepath.Join(cfg.Options.DataDirectory, "logs", "mcp")

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	var servers []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".log") {
			servers = append(servers, strings.TrimSuffix(name, ".log"))
		}
	}

	return servers, nil
}

// CloseLoggers closes all active loggers. Should be called during shutdown.
func CloseLoggers() {
	loggersMu.Lock()
	defer loggersMu.Unlock()

	for _, logger := range loggers {
		if closer, ok := any(logger).(io.Closer); ok {
			_ = closer.Close()
		}
	}
	loggers = make(map[string]*lumberjack.Logger)
}

// sanitizeFilename removes or replaces characters that are not safe for filenames.
func sanitizeFilename(name string) string {
	// Replace problematic characters with underscores.
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}
