package httpserver

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"armory/internal/live"
)

func openStream(t *testing.T, url string) <-chan string {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	lines := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	return lines
}

func waitFor(lines <-chan string, want string) bool {
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return false
			}
			if line == want {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func TestEventsDeliversSpeechOnlyWhenAsked(t *testing.T) {
	hub := live.NewHub()
	srv := &Server{hub: hub}
	ts := httptest.NewServer(http.HandlerFunc(srv.events))
	t.Cleanup(ts.Close)

	listening := openStream(t, ts.URL+"?speech=1")
	plain := openStream(t, ts.URL)
	if !waitFor(listening, ": connected") || !waitFor(plain, ": connected") {
		t.Fatal("streams did not open")
	}

	hub.Say("Roshan,\nuse slot 1")

	if !waitFor(listening, "event: say") {
		t.Fatal("kiosk should receive a say event")
	}
	if !waitFor(listening, "data: Roshan, use slot 1") {
		t.Fatal("say data should be on one line")
	}
	if waitFor(plain, "event: say") {
		t.Fatal("a client without speech=1 must not hear speech")
	}
	if strings.Contains(strings.Join(drain(plain), "\n"), "slot") {
		t.Fatal("speech text leaked to a plain client")
	}
}

func TestEventsDeliversAlertsOnlyWhenAsked(t *testing.T) {
	hub := live.NewHub()
	srv := &Server{hub: hub}
	ts := httptest.NewServer(http.HandlerFunc(srv.events))
	t.Cleanup(ts.Close)

	admin := openStream(t, ts.URL+"?alerts=1")
	kiosk := openStream(t, ts.URL+"?speech=1")
	if !waitFor(admin, ": connected") || !waitFor(kiosk, ": connected") {
		t.Fatal("streams did not open")
	}

	hub.Alert("Wrong gun.\nGun 3 was picked instead of 2.")
	hub.Say("You picked the gun from slot 3")

	if !waitFor(admin, "event: alert") {
		t.Fatal("admin should receive an alert event")
	}
	if !waitFor(admin, "data: Wrong gun. Gun 3 was picked instead of 2.") {
		t.Fatal("alert data should be on one line")
	}
	if waitFor(admin, "event: say") {
		t.Fatal("admin must not receive kiosk speech")
	}
	if waitFor(kiosk, "event: alert") {
		t.Fatal("kiosk must not receive admin alerts")
	}
}

func TestEventsSendsNoticesAsTheirOwnEvent(t *testing.T) {
	hub := live.NewHub()
	srv := &Server{hub: hub}
	ts := httptest.NewServer(http.HandlerFunc(srv.events))
	t.Cleanup(ts.Close)

	admin := openStream(t, ts.URL+"?alerts=1")
	if !waitFor(admin, ": connected") {
		t.Fatal("stream did not open")
	}

	hub.Notice("Roshan has picked the right gun. Gun 1.")

	if !waitFor(admin, "event: notice") {
		t.Fatal("a notice should arrive as a notice event")
	}
	if !waitFor(admin, "data: Roshan has picked the right gun. Gun 1.") {
		t.Fatal("notice data missing")
	}
}

func drain(lines <-chan string) []string {
	var out []string
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return out
			}
			out = append(out, line)
		default:
			return out
		}
	}
}
