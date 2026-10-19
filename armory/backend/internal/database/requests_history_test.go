package database

import (
	"context"
	"testing"

	"armory/internal/models"
)

func TestRequestHistoryPaging(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	user, err := st.CreateUser(ctx, models.User{Name: "Roshan", ServiceNo: "SN-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for i := 0; i < 5; i++ {
		id, err := st.CreateRequest(ctx, user.ID, "rifle", "", "")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	for _, id := range ids[:4] {
		if _, err := st.db.Exec("UPDATE requests SET status = 'returned' WHERE id = ?", id); err != nil {
			t.Fatal(err)
		}
	}

	total, err := st.CountRequestHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4 (pending excluded)", total)
	}

	first, err := st.ListRequestHistory(ctx, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 || first[0].ID != ids[3] || first[2].ID != ids[1] {
		t.Fatalf("first page = %v", first)
	}

	second, err := st.ListRequestHistory(ctx, 3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].ID != ids[0] {
		t.Fatalf("second page = %v", second)
	}
}
