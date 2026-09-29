package app

import (
	"log"
	"net"
	"net/http"

	"armory/internal/config"
	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/services"
	"armory/internal/transport/httpserver"
	"armory/internal/transport/sensor"
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
	lockers := services.NewLockerService(store)
	activity := services.NewActivityService(store)
	hub := live.NewHub()
	sensors := services.NewSensorService(store, hub)

	srv, err := httpserver.New(lockers, activity, hub)
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
	go func() {
		if err := sensor.Listen(cfg.SensorAddr, sensors); err != nil {
			log.Printf("sensor listener stopped: %v", err)
		}
	}()
	log.Printf("listening on http://%s", net.JoinHostPort(host, port))
	return http.ListenAndServe(cfg.Addr, srv.Routes())
}
