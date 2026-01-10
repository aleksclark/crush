package pubsub

import "context"

const (
	CreatedEvent EventType = "created"
	UpdatedEvent EventType = "updated"
	DeletedEvent EventType = "deleted"
)

type Subscriber[T any] interface {
	Subscribe(context.Context) <-chan Event[T]
}

type (
	// EventType identifies the type of event
	EventType string

	// Event represents an event in the lifecycle of a resource
	Event[T any] struct {
		Type    EventType
		Payload T
	}

	Publisher[T any] interface {
		Publish(EventType, T)
	}
)

// UpdateAvailableMsg is sent when a new version is available.
type UpdateAvailableMsg struct {
	CurrentVersion string
	LatestVersion  string
	IsDevelopment  bool
}

// SubagentEventMsg is sent when a subagent is added, updated, or removed.
type SubagentEventMsg struct {
	Type string // "added", "updated", "removed", "reloaded", "error"
	Name string // Name of the affected subagent (empty for "reloaded").
	Err  error  // Error details (for "error" type).
}

// SubagentReloadedMsg is sent when subagents have been reloaded.
type SubagentReloadedMsg struct {
	Count int // Number of subagents loaded.
}
