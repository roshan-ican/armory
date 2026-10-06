package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"armory/internal/services"
)

const port = 47810

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: go run ./cmd/buzz <correct|wrong|unlock> <locker-ip>")
	}
	command, ip := os.Args[1], os.Args[2]
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatalf("open udp socket: %v", err)
	}
	defer conn.Close()

	outputs := services.NewOutputs(conn, port, func(context.Context) ([]string, error) { return []string{ip}, nil }, "")
	switch command {
	case "unlock":
		outputs.PulseUnlock(ip)
	case "correct":
		outputs.PulseRight(ip)
	case "wrong":
		outputs.SetWrong(ip, true)
	default:
		log.Fatalf("unknown command %q", command)
	}

	run := func(d time.Duration) {
		ctx, cancel := context.WithTimeout(context.Background(), d)
		defer cancel()
		outputs.Run(ctx)
	}
	run(3 * time.Second)
	outputs.SetWrong(ip, false)
	run(time.Second)
}
