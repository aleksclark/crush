package mcp

import (
	"context"
	"iter"
	"log/slog"

	"github.com/charmbracelet/crush/internal/csync"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Resource is an alias for the MCP Resource type.
type Resource = mcp.Resource

var allResources = csync.NewMap[string, []*Resource]()

// Resources returns all available MCP resources.
func Resources() iter.Seq2[string, []*Resource] {
	return allResources.Seq2()
}

// GetResources returns resources for a specific MCP server.
func GetResources(name string) []*Resource {
	resources, _ := allResources.Get(name)
	return resources
}

// RefreshResources gets the updated list of resources from the MCP and updates
// the global state.
func RefreshResources(ctx context.Context, name string) {
	entry, ok := sessions.Get(name)
	if !ok {
		slog.Warn("refresh resources: no session", "name", name)
		return
	}

	resources, err := getResources(ctx, entry.session)
	if err != nil {
		slog.Error("error listing resources", "error", err, "name", name)
		return
	}

	updateResources(name, resources)
}

func getResources(ctx context.Context, c *mcp.ClientSession) ([]*Resource, error) {
	if c.InitializeResult().Capabilities.Resources == nil {
		return nil, nil
	}
	result, err := c.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// updateResources updates the global resources map.
func updateResources(mcpName string, resources []*Resource) {
	if len(resources) == 0 {
		allResources.Del(mcpName)
		return
	}
	allResources.Set(mcpName, resources)
}
