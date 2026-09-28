package sensor

import (
	"context"
	"log"
	"net"

	"armory/internal/services"
)

func Listen(addr string, svc *services.SensorService) error {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	tracker := NewTracker()

	log.Printf("sensor listening on udp %s", addr)

	buf := make([]byte, 64)
	for {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			log.Printf("sensor read: %v", err)
			continue
		}
		slots, err := decodeSensorFrame(buf[:n])
		if err != nil {
			log.Printf("from %s: dropped % x: %v", from, buf[:n], err)
			continue
		}
		board := from.(*net.UDPAddr).IP.String()
		for _, c := range tracker.Update(board, slots) {
			log.Printf("board %s slot %d: %d -> %d", board, c.Slot, c.From, c.To)
			slotNo := int64(c.Slot) + 1
			err := svc.Record(context.Background(), board, slotNo, eventFor(c), statusFor(c))
			if err != nil {
				log.Printf("board %s slot %d: save failed: %v", board, slotNo, err)
			}
		}
	}
}
