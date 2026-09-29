package sensor

import (
	"context"
	"log"
	"net"
	"os"

	"armory/internal/services"
)

func Listen(addr string, svc *services.SensorService) error {
	conn, err := net.ListenPacket("udp", addr)

	if err != nil {
		return err
	}
	tracker := NewTracker()
	debug := os.Getenv("ARMORY_SENSOR_DEBUG") != ""

	defer conn.Close()

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
		if debug {
			log.Printf("from %s: % x slots %v", from, buf[:n], slots)
		}
		board := from.(*net.UDPAddr).IP.String()
		ctx := context.Background()

		if !tracker.Seen(board) {
			tracker.Update(board, slots)
			log.Printf("board %s connected: slots %v", board, slots)
			for i, v := range slots {
				c := Change{Slot: i, From: v, To: v}
				slotNo := int64(i) + 1
				if err := svc.Sync(ctx, board, slotNo, v, eventFor(c)); err != nil {
					log.Printf("board %s slot %d: sync failed: %v", board, slotNo, err)
				}
			}
			continue
		}

		for _, c := range tracker.Update(board, slots) {
			log.Printf("board %s slot %d: %d -> %d", board, c.Slot, c.From, c.To)
			slotNo := int64(c.Slot) + 1
			err := svc.Record(ctx, board, slotNo, c.To, eventFor(c))
			if err != nil {
				log.Printf("board %s slot %d: save failed: %v", board, slotNo, err)
			}
		}
	}

}
