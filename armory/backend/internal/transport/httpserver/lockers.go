package httpserver

import (
	"armory/internal/models"
	"armory/internal/services"
	"errors"
	"net/http"
	"strconv"
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
	capacity, err := strconv.ParseInt(r.FormValue("capacity"), 10, 64)
	if err != nil {
		formError(w, services.ErrInvalidCapacity)
		return
	}
	l, err := s.lockers.Create(r.Context(),
		r.FormValue("name"), r.FormValue("location"), r.FormValue("ip_address"),
		r.FormValue("kind"), capacity)
	if errors.Is(err, services.ErrNameRequired) ||
		errors.Is(err, services.ErrLockerExists) ||
		errors.Is(err, services.ErrInvalidIP) ||
		errors.Is(err, services.ErrInvalidKind) ||
		errors.Is(err, services.ErrInvalidCapacity) {
		formError(w, err)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.hub.Publish()
	s.render(w, "lockers", "locker_card", l)
}

func (s *Server) lockerFor(w http.ResponseWriter, r *http.Request) (models.Locker, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return models.Locker{}, false
	}
	l, err := s.lockers.Get(r.Context(), id)
	if errors.Is(err, services.ErrLockerNotFound) {
		http.NotFound(w, r)
		return models.Locker{}, false
	}
	if err != nil {
		serverError(w, err)
		return models.Locker{}, false
	}
	return l, true
}

func (s *Server) lockerCard(w http.ResponseWriter, r *http.Request) {
	if l, ok := s.lockerFor(w, r); ok {
		s.render(w, "lockers", "locker_card", l)
	}
}

func (s *Server) editLocker(w http.ResponseWriter, r *http.Request) {
	if l, ok := s.lockerFor(w, r); ok {
		s.render(w, "lockers", "locker_edit_card", l)
	}
}

func (s *Server) swapSensors(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	target := "#locker-error-" + r.PathValue("id")
	from, ferr := strconv.ParseInt(r.FormValue("from"), 10, 64)
	to, terr := strconv.ParseInt(r.FormValue("to"), 10, 64)
	if ferr != nil || terr != nil {
		formErrorAt(w, services.ErrInvalidSlot, target)
		return
	}
	l, err := s.lockers.SwapSensors(r.Context(), id, from, to)
	if errors.Is(err, services.ErrInvalidSlot) || errors.Is(err, services.ErrSlotInUse) {
		formErrorAt(w, err, target)
		return
	}
	if errors.Is(err, services.ErrLockerNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.hub.Publish()
	s.render(w, "lockers", "locker_card", l)
}

func (s *Server) updateLocker(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	target := "#locker-error-" + r.PathValue("id")
	capacity, err := strconv.ParseInt(r.FormValue("capacity"), 10, 64)
	if err != nil {
		formErrorAt(w, services.ErrInvalidCapacity, target)
		return
	}
	l, err := s.lockers.Update(r.Context(), id,
		r.FormValue("name"), r.FormValue("location"), r.FormValue("ip_address"),
		r.FormValue("kind"), capacity)
	if errors.Is(err, services.ErrNameRequired) ||
		errors.Is(err, services.ErrLockerExists) ||
		errors.Is(err, services.ErrInvalidIP) ||
		errors.Is(err, services.ErrInvalidKind) ||
		errors.Is(err, services.ErrInvalidCapacity) {
		formErrorAt(w, err, target)
		return
	}
	if errors.Is(err, services.ErrLockerNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if raw := r.FormValue("sensor_start"); raw != "" {
		start, perr := strconv.ParseInt(raw, 10, 64)
		if perr != nil {
			formErrorAt(w, services.ErrInvalidSensorStart, target)
			return
		}
		l, err = s.lockers.SetSensorLayout(r.Context(), id, start, r.FormValue("sensor_reverse") == "1")
		if errors.Is(err, services.ErrInvalidSensorStart) {
			formErrorAt(w, err, target)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
	}
	s.hub.Publish()
	s.render(w, "lockers", "locker_card", l)
}
