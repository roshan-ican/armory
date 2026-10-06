package app

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"armory/internal/certs"
	"armory/internal/config"
	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/services"
	"armory/internal/transport/httpserver"
	"armory/internal/transport/sensor"
)

const defaultBoardPort = 47810

func Run() error {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o700); err != nil {
		return err
	}
	if !certs.Exists(cfg.CertDir) {
		if err := certs.Generate(cfg.CertDir, nil); err != nil {
			return err
		}
		log.Printf("created local HTTPS certificates in %s", cfg.CertDir)
	}
	return serve(context.Background(), cfg)
}

func serve(ctx context.Context, cfg config.Config) error {
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Printf("database ready at %s", cfg.DBPath)

	go runBackups(ctx, db, filepath.Dir(cfg.DBPath))

	store := database.NewStore(db)
	lockers := services.NewLockerService(store)
	activity := services.NewActivityService(store)
	users := services.NewUserService(store)
	face := services.NewFaceService(store)
	hub := live.NewHub()
	sensorConn, err := net.ListenPacket("udp", cfg.SensorAddr)
	if err != nil {
		return err
	}
	defer sensorConn.Close()
	sensors := services.NewSensorService(store, hub)

	outputs := services.NewOutputs(sensorConn, boardPort(cfg.SensorAddr), func(ctx context.Context) ([]string, error) {
		all, err := store.ListLockers(ctx)
		if err != nil {
			return nil, err
		}
		var ips []string
		for _, l := range all {
			if l.IPAddress != "" {
				ips = append(ips, l.IPAddress)
			}
		}
		if cfg.DoorIP != "" {
			ips = append(ips, cfg.DoorIP)
		}
		return ips, nil
	}, cfg.DoorIP)
	go outputs.Run(ctx)
	requests := services.NewRequestService(store, outputs, hub)
	sensors.OnSlotChange(requests.SlotChanged)
	sessions := services.NewSessions()

	srv, err := httpserver.New(lockers, activity, users, face, requests, sessions, hub)
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
		if err := sensor.Listen(sensorConn, sensors); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("sensor listener stopped: %v", err)
		}
	}()
	hasCerts := certs.Exists(cfg.CertDir)
	if hasCerts {
		srv.ServeCA(filepath.Join(cfg.CertDir, certs.CAFile))
	}
	handler := srv.Routes()

	errc := make(chan error, 2)
	var servers []*http.Server
	if hasCerts {
		tlsSrv := &http.Server{Addr: cfg.TLSAddr, Handler: handler}
		servers = append(servers, tlsSrv)
		go func() {
			log.Printf("https on %s (CA download at /ca.crt)", cfg.TLSAddr)
			err := tlsSrv.ListenAndServeTLS(filepath.Join(cfg.CertDir, certs.CertFile), filepath.Join(cfg.CertDir, certs.KeyFile))
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			log.Printf("https stopped: %v", err)
		}()
	} else {
		log.Printf("no certificates in %s: run `go run ./cmd/certgen` to enable https", cfg.CertDir)
	}
	plain := handler
	if hasCerts {
		plain = httpserver.RedirectHTTPS(handler, cfg.TLSAddr)
	}
	plainSrv := &http.Server{Addr: cfg.Addr, Handler: plain}
	servers = append(servers, plainSrv)
	go func() { errc <- plainSrv.ListenAndServe() }()

	announce(cfg.TLSAddr)
	log.Printf("listening on http://%s", net.JoinHostPort(host, port))

	select {
	case <-ctx.Done():
	case err = <-errc:
	}
	shutdown(servers)
	return err
}

func shutdown(servers []*http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range servers {
		if err := s.Shutdown(ctx); err != nil {
			s.Close()
		}
	}
}

func boardPort(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return defaultBoardPort
	}
	n, err := strconv.Atoi(port)
	if err != nil || n == 0 {
		return defaultBoardPort
	}
	return n
}

func announce(tlsAddr string) {
	_, port, err := net.SplitHostPort(tlsAddr)
	if err != nil {
		port = "8443"
	}
	for _, ip := range certs.LocalIPs() {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		log.Printf("armory is running: https://%s/admin/login", net.JoinHostPort(ip.String(), port))
	}
}
