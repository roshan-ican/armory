package services

import (
	"context"
	"errors"

	"armory/internal/database"
)

func requireAdmin(ctx context.Context, store *database.Store, userID int64) error {
	u, err := store.GetUser(ctx, userID)
	if errors.Is(err, database.ErrNotFound) {
		return ErrNotAdmin
	}
	if err != nil {
		return err
	}
	if !u.IsAdmin() {
		return ErrNotAdmin
	}
	return nil
}
