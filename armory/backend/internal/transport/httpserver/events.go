package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const heartbeat = 25 * time.Second

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	changes, cancel := s.hub.Subscribe()
	defer cancel()

	var speech <-chan string
	if r.URL.Query().Get("speech") == "1" {
		ch, cancelSpeech := s.hub.SubscribeSpeech()
		defer cancelSpeech()
		speech = ch
	}

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-changes:
			fmt.Fprint(w, "data: changed\n\n")
		case text := <-speech:
			fmt.Fprintf(w, "event: say\ndata: %s\n\n", strings.ReplaceAll(text, "\n", " "))
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		flusher.Flush()
	}
}
