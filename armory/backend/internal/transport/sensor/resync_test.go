package sensor

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
	"armory/internal/services"
)

func TestReenabledSensorReadsTheBoardAgain(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)

	locker, err := store.CreateLocker(ctx, models.Locker{Name: "A", IPAddress: "127.0.0.1", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	sensors := services.NewSensorService(store, live.NewHub())
	lockers := services.NewLockerService(store)
	lockers.OnSensorEnabled(sensors.RequestResync)

	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go Listen(server, sensors)

	client, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		frame := []byte{0x24, 0x01, 0x01, 0x01, 0x31, 0x23}
		for {
			select {
			case <-stop:
				return
			default:
				client.Write(frame)
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	reading := func() int64 {
		l, err := store.GetLocker(ctx, locker.ID)
		if err != nil {
			t.Fatal(err)
		}
		return l.Slots[0].Reading
	}
	waitFor := func(want int64) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if reading() == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("slot 1 reading = %d, want %d", reading(), want)
	}

	waitFor(1)
	if _, err := lockers.SetSensorDisabled(ctx, locker.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	waitFor(2)
	if _, err := lockers.SetSensorDisabled(ctx, locker.ID, 1, false); err != nil {
		t.Fatal(err)
	}
	waitFor(1)
}
