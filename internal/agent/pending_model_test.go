package agent

import (
	"sync"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPendingModelUpdate(t *testing.T) {
	t.Parallel()

	t.Run("QueueModelUpdate stores pending update", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		model := config.SelectedModel{
			Provider: "test-provider",
			Model:    "test-model",
		}

		c.QueueModelUpdate(config.SelectedModelTypeLarge, model)

		assert.True(t, c.HasPendingModelUpdate())
	})

	t.Run("HasPendingModelUpdate returns false when no pending update", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		assert.False(t, c.HasPendingModelUpdate())
	})

	t.Run("ClearPendingModelUpdate removes pending update", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		model := config.SelectedModel{
			Provider: "test-provider",
			Model:    "test-model",
		}

		c.QueueModelUpdate(config.SelectedModelTypeLarge, model)
		assert.True(t, c.HasPendingModelUpdate())

		c.ClearPendingModelUpdate()
		assert.False(t, c.HasPendingModelUpdate())
	})

	t.Run("QueueModelUpdate overwrites previous pending update", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		model1 := config.SelectedModel{
			Provider: "provider-1",
			Model:    "model-1",
		}
		model2 := config.SelectedModel{
			Provider: "provider-2",
			Model:    "model-2",
		}

		c.QueueModelUpdate(config.SelectedModelTypeLarge, model1)
		c.QueueModelUpdate(config.SelectedModelTypeSmall, model2)

		// Should still have a pending update
		assert.True(t, c.HasPendingModelUpdate())

		// Verify the latest update is stored
		c.pendingModelMu.Lock()
		pending := c.pendingModelUpdate
		c.pendingModelMu.Unlock()

		require.NotNil(t, pending)
		assert.Equal(t, config.SelectedModelTypeSmall, pending.ModelType)
		assert.Equal(t, "model-2", pending.Model.Model)
	})
}

func TestPendingModelUpdateConcurrency(t *testing.T) {
	t.Parallel()

	c := &coordinator{}

	var wg sync.WaitGroup
	const numGoroutines = 20

	// Concurrently queue model updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			model := config.SelectedModel{
				Provider: "provider",
				Model:    "model-" + string(rune('a'+id)),
			}
			modelType := config.SelectedModelTypeLarge
			if id%2 == 0 {
				modelType = config.SelectedModelTypeSmall
			}
			c.QueueModelUpdate(modelType, model)
		}(i)
	}

	// Concurrently check and clear pending updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.HasPendingModelUpdate()
			if c.HasPendingModelUpdate() {
				c.ClearPendingModelUpdate()
			}
		}()
	}

	wg.Wait()

	// No panic means the concurrent access is safe
}

func TestPendingModelUpdateTypes(t *testing.T) {
	t.Parallel()

	t.Run("stores large model type", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		model := config.SelectedModel{
			Provider: "anthropic",
			Model:    "claude-opus-4-20250514",
		}

		c.QueueModelUpdate(config.SelectedModelTypeLarge, model)

		c.pendingModelMu.Lock()
		pending := c.pendingModelUpdate
		c.pendingModelMu.Unlock()

		require.NotNil(t, pending)
		assert.Equal(t, config.SelectedModelTypeLarge, pending.ModelType)
		assert.Equal(t, "claude-opus-4-20250514", pending.Model.Model)
		assert.Equal(t, "anthropic", pending.Model.Provider)
	})

	t.Run("stores small model type", func(t *testing.T) {
		t.Parallel()
		c := &coordinator{}

		model := config.SelectedModel{
			Provider: "anthropic",
			Model:    "claude-haiku-3-20240307",
		}

		c.QueueModelUpdate(config.SelectedModelTypeSmall, model)

		c.pendingModelMu.Lock()
		pending := c.pendingModelUpdate
		c.pendingModelMu.Unlock()

		require.NotNil(t, pending)
		assert.Equal(t, config.SelectedModelTypeSmall, pending.ModelType)
		assert.Equal(t, "claude-haiku-3-20240307", pending.Model.Model)
	})
}
