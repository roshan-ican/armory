package httpserver

import "net/http"

func (s *Server) kioskPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, "kiosk", "kiosk", nil)
}

func (s *Server) downloadCA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="armory-ca.crt"`)
	http.ServeFile(w, r, s.caPath)
}

func redirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}
