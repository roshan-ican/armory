package live

import (
	"testing"
	"time"
)

func received(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(50 * time.Millisecond):
		return false
	}
}

func TestPublishReachesAllSubscribers(t *testing.T) {
	h := NewHub()
	a, cancelA := h.Subscribe()
	defer cancelA()
	b, cancelB := h.Subscribe()
	defer cancelB()

	h.Publish()

	if !received(a) || !received(b) {
		t.Fatal("every subscriber should be notified")
	}
}

func TestPublishCoalescesBursts(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	defer cancel()

	h.Publish()
	h.Publish()
	h.Publish()

	if !received(ch) {
		t.Fatal("want one notification")
	}
	if received(ch) {
		t.Fatal("a burst should collapse into a single pending notification")
	}
}

func TestCancelStopsNotifications(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	cancel()

	h.Publish()

	if received(ch) {
		t.Fatal("cancelled subscriber should not be notified")
	}
}

func TestNilHubPublishIsSafe(t *testing.T) {
	var h *Hub
	h.Publish()
}
