package sensor

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"armory/internal/services"
)

func Listen(conn net.PacketConn, svc *services.SensorService) error {
	tracker := NewTracker()
	debug := os.Getenv("ARMORY_SENSOR_DEBUG") != ""
	lastSeen := make(map[string]time.Time)

	defer conn.Close()

	log.Printf("sensor listening on udp %s", conn.LocalAddr())

	buf := make([]byte, 64)

	for {
		n, from, err := conn.ReadFrom(buf)
		if errors.Is(err, net.ErrClosed) {
			return err
		}
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
		if time.Since(lastSeen[board]) >= 2*time.Second {
			if err := svc.Seen(ctx, board); err != nil {
				log.Printf("board %s: heartbeat failed: %v", board, err)
			}
			lastSeen[board] = time.Now()
		}

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
			slotNo := int64(c.Slot) + 1
			log.Printf("board %s slot %d: %d -> %d", board, slotNo, c.From, c.To)
			err := svc.Record(ctx, board, slotNo, c.To, eventFor(c))
			if err != nil {
				log.Printf("board %s slot %d: save failed: %v", board, slotNo, err)
			}
		}
	}

}
