package httpserver

import (
	"net"
	"net/http"
)

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func RedirectHTTPS(next http.Handler, tlsAddr string) http.Handler {
	_, port, err := net.SplitHostPort(tlsAddr)
	if err != nil || port == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if r.TLS != nil || isLoopback(host) || r.URL.Path == "/ca.crt" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "https://"+net.JoinHostPort(host, port)+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	})
}
