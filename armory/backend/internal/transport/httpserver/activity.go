package httpserver

import "net/http"

func (s *Server) listActivity(w http.ResponseWriter, r *http.Request) {
	events, err := s.activity.Recent(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "activity", "layout", map[string]any{
		"Title":  "Activity",
		"Events": events,
	})
}
