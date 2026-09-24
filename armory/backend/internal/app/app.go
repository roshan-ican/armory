package app

import (
	"log"
	"net/http"

	"armory/internal/config"
	"armory/internal/database"
)

func Run() error {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Printf("database ready at %s", cfg.DBPath)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Printf("listening on %s", cfg.Addr)
	return http.ListenAndServe(cfg.Addr, mux)
}
