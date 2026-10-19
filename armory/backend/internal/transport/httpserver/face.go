package httpserver

import (
	"errors"
	"net/http"
	"time"

	"armory/internal/models"
	"armory/internal/services"
)

func (s *Server) identifyFace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Descriptor []float64 `json:"descriptor"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	res, err := s.face.Match(r.Context(), body.Descriptor)
	if errors.Is(err, services.ErrInvalidDescriptor) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if !res.Matched || res.User.Role == "admin" {
		writeJSON(w, map[string]any{"matched": false})
		return
	}
	writeJSON(w, map[string]any{"matched": true, "name": res.User.Name})
}

func (s *Server) matchFace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Descriptor []float64 `json:"descriptor"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	res, err := s.face.Match(r.Context(), body.Descriptor)
	if errors.Is(err, services.ErrInvalidDescriptor) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if !res.Matched {
		writeJSON(w, map[string]any{"matched": false})
		return
	}
	if res.User.Role == "admin" {
		writeJSON(w, map[string]any{"matched": false})
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
