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

func TestAlertReachesOnlyAlertSubscribers(t *testing.T) {
	h := NewHub()
	alerts, cancelAlerts := h.SubscribeAlerts()
	defer cancelAlerts()
	speech, cancelSpeech := h.SubscribeSpeech()
	defer cancelSpeech()
	changes, cancelChanges := h.Subscribe()
	defer cancelChanges()

	h.Alert("Wrong gun")

	if alert, ok := heardAlert(alerts); !ok || alert.Text != "Wrong gun" || !alert.Urgent {
		t.Fatalf("got %+v, %t", alert, ok)
	}
	if _, ok := heard(speech); ok {
		t.Fatal("an admin alert must not reach the kiosk speech")
	}
	if received(changes) {
		t.Fatal("an alert must not look like a data change")
	}
}

func TestSayDoesNotReachAlertSubscribers(t *testing.T) {
	h := NewHub()
	alerts, cancel := h.SubscribeAlerts()
	defer cancel()

	h.Say("You picked the gun from slot 3")

	if _, ok := heardAlert(alerts); ok {
		t.Fatal("kiosk speech must not reach admin alerts")
	}
}

func TestCancelStopsAlerts(t *testing.T) {
	h := NewHub()
	ch, cancel := h.SubscribeAlerts()
	cancel()

	h.Alert("late")

	if _, ok := heardAlert(ch); ok {
		t.Fatal("cancelled subscriber should not hear anything")
	}
}

func TestNoticeIsNotUrgent(t *testing.T) {
	h := NewHub()
	ch, cancel := h.SubscribeAlerts()
	defer cancel()

	h.Notice("Right gun")

	if alert, ok := heardAlert(ch); !ok || alert.Text != "Right gun" || alert.Urgent {
		t.Fatalf("got %+v, %t", alert, ok)
	}
}

func TestNilHubNoticeIsSafe(t *testing.T) {
	var h *Hub
	h.Notice("nobody")
}

func heardAlert(ch <-chan AdminAlert) (AdminAlert, bool) {
	select {
	case alert := <-ch:
		return alert, true
	case <-time.After(50 * time.Millisecond):
		return AdminAlert{}, false
	}
}

func TestNilHubAlertIsSafe(t *testing.T) {
	var h *Hub
	h.Alert("nobody")
}
