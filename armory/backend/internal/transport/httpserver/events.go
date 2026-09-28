package httpserver

import (
	"fmt"
	"net/http"
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
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		flusher.Flush()
	}
}
