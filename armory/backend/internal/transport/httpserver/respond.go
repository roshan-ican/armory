package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"armory/internal/services"
)

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func errorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, services.ErrInvalidKind),
		errors.Is(err, services.ErrInvalidReturn),
		errors.Is(err, services.ErrInvalidSlot),
		errors.Is(err, services.ErrEmptyPlan),
		errors.Is(err, services.ErrBadChange),
		errors.Is(err, services.ErrLockerNotFound):
		return http.StatusBadRequest, true
	case errors.Is(err, services.ErrRequestNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, services.ErrRequestOpen),
		errors.Is(err, services.ErrRequestClosed),
		errors.Is(err, services.ErrMaintenanceActive),
		errors.Is(err, services.ErrNoneAvailable),
		errors.Is(err, services.ErrGunUnavailable),
		errors.Is(err, services.ErrMaintenanceOpen),
		errors.Is(err, services.ErrMaintenanceBusy),
		errors.Is(err, services.ErrMaintenanceNotFound):
		return http.StatusConflict, true
	case errors.Is(err, services.ErrNotAdmin),
		errors.Is(err, services.ErrNotRequester):
		return http.StatusForbidden, true
	}
	return 0, false
}

func (s *Server) apiError(w http.ResponseWriter, err error) {
	if status, ok := errorStatus(err); ok {
		writeJSONError(w, status, err.Error())
		return
	}
	serverError(w, err)
}
