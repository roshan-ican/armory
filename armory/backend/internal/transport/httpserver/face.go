package httpserver

import (
	"errors"
	"net/http"
	"time"

	"armory/internal/models"
	"armory/internal/services"
)

func (s *Server) matchRequester(w http.ResponseWriter, r *http.Request) (services.MatchResult, bool) {
	var body struct {
		Descriptor []float64 `json:"descriptor"`
	}
	if !readJSON(w, r, &body) {
		return services.MatchResult{}, false
	}
	res, err := s.face.Match(r.Context(), body.Descriptor)
	if errors.Is(err, services.ErrInvalidDescriptor) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return services.MatchResult{}, false
	}
	if err != nil {
		serverError(w, err)
		return services.MatchResult{}, false
	}
	if !res.Matched || res.User.IsAdmin() {
		writeJSON(w, map[string]any{"matched": false})
		return services.MatchResult{}, false
	}
	return res, true
}

func (s *Server) identifyFace(w http.ResponseWriter, r *http.Request) {
	res, ok := s.matchRequester(w, r)
	if !ok {
		return
	}
	writeJSON(w, map[string]any{"matched": true, "name": res.User.Name})
}

func (s *Server) matchFace(w http.ResponseWriter, r *http.Request) {
	res, ok := s.matchRequester(w, r)
	if !ok {
		return
	}
	s.startSession(w, r, s.sessions.Start(res.User))
	out := map[string]any{
		"matched":  true,
		"name":     res.User.Name,
		"role":     res.User.Role,
		"greeting": services.Greeting(time.Now()),
		"distance": res.Distance,
	}
	if job, err := s.maintain.Current(r.Context(), res.User.ID); err == nil {
		out["maintenance_status"] = job.Status
		out["maintenance"] = job.Status == models.MaintenanceActive
	}
	writeJSON(w, out)
}
