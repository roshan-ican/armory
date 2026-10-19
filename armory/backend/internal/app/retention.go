package app

import (
	"context"
	"log"
	"time"

	"armory/internal/database"
)

const keepEventsFor = 30 * 24 * time.Hour

func runRetention(ctx context.Context, store *database.Store) {
	for {
		cutoff := time.Now().UTC().Add(-keepEventsFor).Format(time.RFC3339)
		n, err := store.DeleteEventsBefore(ctx, cutoff)
		if err != nil {
			log.Printf("activity cleanup failed: %v", err)
		} else if n > 0 {
			log.Printf("activity cleanup removed %d events older than %s", n, cutoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(24 * time.Hour):
		}
	}
}
