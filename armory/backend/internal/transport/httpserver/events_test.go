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
