package httpserver

import (
	"errors"
	"net/http"

	"armory/internal/services"
)

func (s *Server) listLockers(w http.ResponseWriter, r *http.Request) {
	lockers, err := s.lockers.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "lockers", "layout", map[string]any{
		"Title":   "Lockers",
		"Lockers": lockers,
	})
}

func (s *Server) createLocker(w http.ResponseWriter, r *http.Request) {
	l, err := s.lockers.Create(r.Context(),
		r.FormValue("name"), r.FormValue("location"), r.FormValue("ip_address"))
	if errors.Is(err, services.ErrNameRequired) ||
		errors.Is(err, services.ErrLockerExists) ||
		errors.Is(err, services.ErrInvalidIP) {
		formError(w, err)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "lockers", "locker_card", l)
}
