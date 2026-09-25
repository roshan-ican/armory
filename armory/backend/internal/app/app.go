package app

import (
	"log"
	"net"
	"net/http"

	"armory/internal/config"
	"armory/internal/database"
	"armory/internal/services"
	"armory/internal/transport/httpserver"
)

func Run() error {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Printf("database ready at %s", cfg.DBPath)

	store := database.NewStore(db)
	categories := services.NewCategoryService(store)
	lockers := services.NewLockerService(store)

	srv, err := httpserver.New(categories, lockers)
	if err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return err
	}
	if host == "" {
		host = "localhost"
	}
	log.Printf("listening on http://%s", net.JoinHostPort(host, port))
	return http.ListenAndServe(cfg.Addr, srv.Routes())
}
