package sensor

import (
	"log"
	"net"
)

func Listen(addr string) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("sensor listening on udp %s", addr)

	buf := make([]byte, 64)
	for {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("sensor read: %v", err)
			continue
		}
		log.Printf("from %s: % x", from, buf[:n])
	}
}
