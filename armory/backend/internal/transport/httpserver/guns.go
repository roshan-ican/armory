package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"armory/internal/models"
	"armory/internal/services"
)

func (s *Server) listGuns(w http.ResponseWriter, r *http.Request) {
	guns, err := s.guns.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	categories, err := s.categories.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	freeSlots, err := s.guns.FreeSlots(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "guns", "layout", map[string]any{
		"Title":      "Guns",
		"Guns":       guns,
		"Categories": categories,
		"FreeSlots":  freeSlots,
	})
}

func (s *Server) createGun(w http.ResponseWriter, r *http.Request) {
	_, err := s.guns.Create(r.Context(), models.Gun{
		CategoryID: formInt(r, "category_id"),
		SlotID:     formInt(r, "slot_id"),
		Serial:     r.FormValue("serial"),
		Model:      r.FormValue("model"),
		Notes:      r.FormValue("notes"),
	})
	if errors.Is(err, services.ErrCategoryRequired) ||
		errors.Is(err, services.ErrSerialRequired) ||
		errors.Is(err, services.ErrGunExists) ||
		errors.Is(err, services.ErrSlotRequired) ||
		errors.Is(err, services.ErrSlotOccupied) {
		formError(w, err)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
}

func (s *Server) retireGun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	g, err := s.guns.Retire(r.Context(), id)
	if errors.Is(err, services.ErrCannotRetire) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "guns", "gun_row", g)
}

func formInt(r *http.Request, key string) int64 {
	n, _ := strconv.ParseInt(r.FormValue(key), 10, 64)
	return n
}

func (s *Server) gunRow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g, err := s.guns.Get(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "guns", "gun_row", g)
}

func (s *Server) editGun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g, err := s.guns.Get(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	categories, err := s.categories.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	freeSlots, err := s.guns.FreeSlots(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "guns", "gun_edit_row", map[string]any{
		"Gun":        g,
		"Categories": categories,
		"FreeSlots":  freeSlots,
	})
}

func (s *Server) updateGun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g, err := s.guns.Update(r.Context(), models.Gun{
		ID:         id,
		CategoryID: formInt(r, "category_id"),
		SlotID:     formInt(r, "slot_id"),
		Serial:     r.FormValue("serial"),
		Model:      r.FormValue("model"),
		Notes:      r.FormValue("notes"),
	})
	if errors.Is(err, services.ErrCategoryRequired) ||
		errors.Is(err, services.ErrSlotRequired) ||
		errors.Is(err, services.ErrSerialRequired) ||
		errors.Is(err, services.ErrGunExists) ||
		errors.Is(err, services.ErrSlotOccupied) ||
		errors.Is(err, services.ErrCannotEdit) {
		formError(w, err)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	s.render(w, "guns", "gun_row", g)
}
