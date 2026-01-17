package permission

import (
	"context"
	"slices"

	"github.com/charmbracelet/crush/internal/pubsub"
)

// PermissionMode controls how permissions are handled for an agent.
type PermissionMode string

const (
	// ModeDefault uses standard permission checking - prompts for all
	// permission requests.
	ModeDefault PermissionMode = "default"
	// ModeAcceptEdits auto-accepts file edits but prompts for bash commands.
	ModeAcceptEdits PermissionMode = "acceptEdits"
	// ModeDontAsk auto-denies all permission prompts (safe mode).
	ModeDontAsk PermissionMode = "dontAsk"
	// ModeBypassPermissions skips all permission checks (yolo mode for agents).
	ModeBypassPermissions PermissionMode = "bypassPermissions"
	// ModeYolo is an alias for ModeBypassPermissions.
	ModeYolo PermissionMode = "yolo"
	// ModePlan enables read-only exploration mode - denies all write operations.
	ModePlan PermissionMode = "plan"
)

// IsValid returns true if the permission mode is a known value.
func (p PermissionMode) IsValid() bool {
	switch p {
	case "", ModeDefault, ModeAcceptEdits, ModeDontAsk, ModeBypassPermissions, ModeYolo, ModePlan:
		return true
	}
	return false
}

// Normalize returns the canonical form of the permission mode.
// This converts aliases (like "yolo") to their canonical form ("bypassPermissions").
func (p PermissionMode) Normalize() PermissionMode {
	if p == ModeYolo {
		return ModeBypassPermissions
	}
	return p
}

// editTools are tools that modify files.
var editTools = []string{"edit", "multiedit", "write"}

// readOnlyTools are tools that only read data.
var readOnlyTools = []string{"glob", "grep", "ls", "view", "sourcegraph", "fetch", "lsp_diagnostics", "lsp_references"}

// PermissionWrapper wraps a permission service with a specific permission mode.
// It delegates to the underlying service but may auto-approve or auto-deny
// based on the configured mode.
type PermissionWrapper struct {
	underlying Service
	mode       PermissionMode
}

// WrapWithMode creates a new permission wrapper with the given mode.
// If mode is empty or default, returns the underlying service directly.
func WrapWithMode(svc Service, mode PermissionMode) Service {
	if mode == "" || mode == ModeDefault {
		return svc
	}
	return &PermissionWrapper{
		underlying: svc,
		mode:       mode,
	}
}

// Request implements Service.Request with mode-specific behavior.
func (w *PermissionWrapper) Request(ctx context.Context, opts CreatePermissionRequest) (bool, error) {
	switch w.mode {
	case ModeBypassPermissions, ModeYolo:
		// Skip all permission checks.
		return true, nil

	case ModeDontAsk:
		// Auto-deny all permission requests.
		return false, ErrorPermissionDenied

	case ModePlan:
		// Allow only read-only tools.
		if slices.Contains(readOnlyTools, opts.ToolName) {
			return true, nil
		}
		return false, ErrorPermissionDenied

	case ModeAcceptEdits:
		// Auto-accept file edits, prompt for everything else.
		if slices.Contains(editTools, opts.ToolName) {
			return true, nil
		}
		// Fall through to underlying service for bash, etc.
		return w.underlying.Request(ctx, opts)

	default:
		// Unknown mode, use underlying service.
		return w.underlying.Request(ctx, opts)
	}
}

// Subscribe implements Service.
func (w *PermissionWrapper) Subscribe(ctx context.Context) <-chan pubsub.Event[PermissionRequest] {
	return w.underlying.Subscribe(ctx)
}

// GrantPersistent implements Service.
func (w *PermissionWrapper) GrantPersistent(permission PermissionRequest) {
	w.underlying.GrantPersistent(permission)
}

// Grant implements Service.
func (w *PermissionWrapper) Grant(permission PermissionRequest) {
	w.underlying.Grant(permission)
}

// Deny implements Service.
func (w *PermissionWrapper) Deny(permission PermissionRequest) {
	w.underlying.Deny(permission)
}

// AutoApproveSession implements Service.
func (w *PermissionWrapper) AutoApproveSession(sessionID string) {
	w.underlying.AutoApproveSession(sessionID)
}

// SetSkipRequests implements Service.
func (w *PermissionWrapper) SetSkipRequests(skip bool) {
	w.underlying.SetSkipRequests(skip)
}

// SkipRequests implements Service.
func (w *PermissionWrapper) SkipRequests() bool {
	// For bypass/yolo mode, report as skipping requests.
	if w.mode == ModeBypassPermissions || w.mode == ModeYolo {
		return true
	}
	return w.underlying.SkipRequests()
}

// SubscribeNotifications implements Service.
func (w *PermissionWrapper) SubscribeNotifications(ctx context.Context) <-chan pubsub.Event[PermissionNotification] {
	return w.underlying.SubscribeNotifications(ctx)
}

// DescribeEffectivePermissions returns a human-readable description of what
// permissions will be granted for the given mode.
func DescribeEffectivePermissions(mode PermissionMode) string {
	switch mode {
	case ModeBypassPermissions, ModeYolo:
		return "All operations auto-approved (yolo mode)"
	case ModeDontAsk:
		return "All operations auto-denied (safe mode)"
	case ModePlan:
		return "Read-only operations allowed, writes denied"
	case ModeAcceptEdits:
		return "File edits auto-approved, bash prompts for approval"
	case ModeDefault, "":
		return "All operations prompt for approval"
	default:
		return "Unknown mode: prompts for approval"
	}
}
