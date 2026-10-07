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

func heard(ch <-chan string) (string, bool) {
	select {
	case text := <-ch:
		return text, true
	case <-time.After(50 * time.Millisecond):
		return "", false
	}
}

func TestSayReachesAllSpeechSubscribers(t *testing.T) {
	h := NewHub()
	a, cancelA := h.SubscribeSpeech()
	defer cancelA()
	b, cancelB := h.SubscribeSpeech()
	defer cancelB()

	h.Say("pick from slot 1")

	for _, ch := range []<-chan string{a, b} {
		text, ok := heard(ch)
		if !ok || text != "pick from slot 1" {
			t.Fatalf("got %q, %t", text, ok)
		}
	}
}

func TestSayDoesNotWakeChangeSubscribers(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe()
	defer cancel()

	h.Say("hello")

	if received(ch) {
		t.Fatal("speech must not look like a data change")
	}
}

func TestSayKeepsOrderAndDropsWhenFull(t *testing.T) {
	h := NewHub()
	ch, cancel := h.SubscribeSpeech()
	defer cancel()

	for _, text := range []string{"1", "2", "3", "4", "5", "6"} {
		h.Say(text)
	}

	var got []string
	for {
		text, ok := heard(ch)
		if !ok {
			break
		}
		got = append(got, text)
	}
	if len(got) != 4 || got[0] != "1" || got[3] != "4" {
		t.Fatalf("got %v, want the first four in order", got)
	}
}

func TestCancelStopsSpeech(t *testing.T) {
	h := NewHub()
	ch, cancel := h.SubscribeSpeech()
	cancel()

	h.Say("late")

	if _, ok := heard(ch); ok {
		t.Fatal("cancelled subscriber should not hear anything")
	}
}

func TestNilHubSayIsSafe(t *testing.T) {
	var h *Hub
	h.Say("nobody")
}
