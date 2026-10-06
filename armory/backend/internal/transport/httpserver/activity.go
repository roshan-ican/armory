package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"armory/internal/services"
)

func dateParam(r *http.Request, key string) string {
	v := r.URL.Query().Get(key)
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return ""
	}
	return v
}

func (s *Server) listActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	filter := services.ActivityFilter{
		Person: q.Get("person"),
		Locker: q.Get("locker"),
		Type:   q.Get("type"),
		From:   dateParam(r, "from"),
		To:     dateParam(r, "to"),
	}
	view, err := s.activity.Page(r.Context(), page, filter)
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "activity", "layout", map[string]any{
		"Title": "Activity",
		"View":  view,
	})
}
