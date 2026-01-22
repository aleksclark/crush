// Package skills provides logging functionality for skill usage.
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/tracing"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogEntry represents a single skill usage log entry.
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id"`
	SkillName  string    `json:"skill_name"`
	Action     string    `json:"action"` // "read", "activated", etc.
	FilePath   string    `json:"file_path,omitempty"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
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

// getLogger returns or creates a logger for the specified skill.
func getLogger(skillName string) *lumberjack.Logger {
	loggersMu.Lock()
	defer loggersMu.Unlock()

	if logger, ok := loggers[skillName]; ok {
		return logger
	}

	cfg := config.Get()
	logsDir := filepath.Join(cfg.Options.DataDirectory, "logs", "skills")

	// Ensure the directory exists.
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil
	}

	logger := &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, fmt.Sprintf("%s.log", sanitizeFilename(skillName))),
		MaxSize:    5, // Max size in MB
		MaxBackups: 3,
		MaxAge:     7,    // Days
		Compress:   true, // Enable compression
	}

	loggers[skillName] = logger
	return logger
}

// LogUsage logs a skill usage event.
func LogUsage(entry LogEntry) {
	if entry.SessionID == "" {
		entry.SessionID = GetSessionID()
	}

	// Emit tracing span for skill usage.
	skillSpan := tracing.StartSkillUsage(context.Background(), entry.SkillName, entry.Action, entry.SessionID)
	skillSpan.SetResult(entry.Success, entry.FilePath, entry.DurationMs)
	if !entry.Success && entry.Error != "" {
		skillSpan.SetError(fmt.Errorf("%s", entry.Error))
	}
	skillSpan.End()

	logger := getLogger(entry.SkillName)
	if logger == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = logger.Write(append(data, '\n'))
}

// GetLogEntries reads log entries for a specific skill, optionally filtered by session ID.
func GetLogEntries(skillName, sessionID string, limit int) ([]LogEntry, error) {
	cfg := config.Get()
	logFile := filepath.Join(
		cfg.Options.DataDirectory,
		"logs",
		"skills",
		fmt.Sprintf("%s.log", sanitizeFilename(skillName)),
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

// GetAllSkillLogs returns the names of all skills that have log files.
func GetAllSkillLogs() ([]string, error) {
	cfg := config.Get()
	logsDir := filepath.Join(cfg.Options.DataDirectory, "logs", "skills")

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".log") {
			skills = append(skills, strings.TrimSuffix(name, ".log"))
		}
	}

	return skills, nil
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
