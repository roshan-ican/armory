package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"armory/internal/models"
	"armory/internal/services"
)

func TestAdminActionKeepsValidatedSessionDuringLogout(t *testing.T) {
	s := &Server{sessions: services.NewSessions()}
	admin := models.User{ID: 23, Role: models.RoleAdmin}
	token := s.sessions.Start(admin)
	handler := s.adminOnlyWithSession(func(w http.ResponseWriter, r *http.Request, sess services.Session) {
		s.endSession(w, r)
		if sess.UserID != admin.ID || !sess.IsAdmin() {
			t.Fatalf("actor = %+v, want admin %d", sess, admin.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/enrollments/1/approve", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	if first.Code != http.StatusNoContent {
		t.Fatalf("authorized request = %d, want 204", first.Code)
	}

	afterLogout := httptest.NewRecorder()
	handler.ServeHTTP(afterLogout, req)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("request after logout = %d, want 401", afterLogout.Code)
	}
}
