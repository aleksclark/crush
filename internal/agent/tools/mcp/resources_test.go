package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResourceListNotificationsAreRateLimited(t *testing.T) {
	const name = "resource-notification"
	resourceNotificationMu.Lock()
	delete(resourceNotificationUntil, name)
	resourceNotificationMu.Unlock()
	t.Cleanup(func() {
		resourceNotificationMu.Lock()
		delete(resourceNotificationUntil, name)
		resourceNotificationMu.Unlock()
	})

	require.True(t, shouldHandleResourceListNotification(name))
	require.False(t, shouldHandleResourceListNotification(name))
}

func TestResourceListNotificationRateLimitExpires(t *testing.T) {
	const name = "expired-resource-notification"
	resourceNotificationMu.Lock()
	resourceNotificationUntil[name] = time.Now().Add(-resourceListNotificationWindow)
	resourceNotificationMu.Unlock()
	t.Cleanup(func() {
		resourceNotificationMu.Lock()
		delete(resourceNotificationUntil, name)
		resourceNotificationMu.Unlock()
	})

	require.True(t, shouldHandleResourceListNotification(name))
}
