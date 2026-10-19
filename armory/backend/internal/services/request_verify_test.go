package services

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"armory/internal/models"
)

func TestVerifyReturn(t *testing.T) {
	req := models.Request{Slots: []models.RequestSlot{{SlotNo: 1}, {SlotNo: 3}}}
	slot := func(no, reading int64) models.Slot { return models.Slot{SlotNo: no, Reading: reading} }

	tests := []struct {
		name  string
		slots []models.Slot
		wrong []int64
		want  []string
	}{
		{"all back", []models.Slot{slot(1, 1), slot(2, 1), slot(3, 1)}, nil, nil},
		{"a gun is missing", []models.Slot{slot(1, 1), slot(2, 1), slot(3, 0)}, nil, []string{"Slot 3 is empty, the gun is not back."}},
		{"a sensor is not detecting", []models.Slot{slot(1, 2), slot(3, 1)}, nil, []string{"Slot 1 is not detecting."}},
		{"no reading yet counts as missing", []models.Slot{slot(1, -1), slot(3, 1)}, nil, []string{"Slot 1 is empty, the gun is not back."}},
		{"wrong gun still out", []models.Slot{slot(1, 1), slot(2, 0), slot(3, 1)}, []int64{2}, []string{"Slot 2 is empty, a gun was taken that was not assigned."}},
		{"a requested slot is not reported twice", []models.Slot{slot(1, 0), slot(3, 1)}, []int64{1}, []string{"Slot 1 is empty, the gun is not back."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyReturn(req, tt.slots, tt.wrong)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func (f flow) returnedRequest(t *testing.T, slots ...int64) models.Request {
	t.Helper()
	ctx := context.Background()
	req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, slots, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Approve(ctx, f.admin.ID, req.ID); err != nil {
		t.Fatal(err)
	}
	for _, no := range slots {
		f.sensor(t, no, 0)
	}
	for _, no := range slots {
		f.sensor(t, no, 1)
	}
	if status, _ := f.status(t, req.ID); status != models.RequestReturned {
		t.Fatalf("setup: status %s", status)
	}
	return req
}

func TestCloseVerifiesTheReturn(t *testing.T) {
	ctx := context.Background()

	t.Run("everything back closes it", func(t *testing.T) {
		f := newFlow(t)
		req := f.returnedRequest(t, 1, 3)

		problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false)
		if err != nil || len(problems) != 0 {
			t.Fatalf("close = %v, %v", problems, err)
		}
		got, _ := f.store.GetRequest(ctx, req.ID)
		if got.ClosedAt == "" {
			t.Fatal("request was not closed")
		}
		if open, _ := f.store.HasOpenRequest(ctx, f.user.ID); open {
			t.Fatal("a closed request must not be open")
		}
		if _, err := f.svc.Close(ctx, f.user.ID, req.ID, false); !errors.Is(err, ErrRequestClosed) {
			t.Fatalf("closing twice = %v, want ErrRequestClosed", err)
		}
	})

	t.Run("a gun that left again is flagged", func(t *testing.T) {
		f := newFlow(t)
		req := f.returnedRequest(t, 1, 3)
		f.sensor(t, 3, 0)

		problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false)
		if err != nil || len(problems) != 1 || !strings.Contains(problems[0], "Slot 3") {
			t.Fatalf("close = %v, %v", problems, err)
		}
		got, _ := f.store.GetRequest(ctx, req.ID)
		if got.ClosedAt != "" || got.CloseNote == "" {
			t.Fatalf("closed %q note %q, want open with a note", got.ClosedAt, got.CloseNote)
		}

		f.sensor(t, 3, 1)
		if problems, err := f.svc.Close(ctx, f.user.ID, req.ID, false); err != nil || len(problems) != 0 {
			t.Fatalf("close after fixing = %v, %v", problems, err)
		}
	})

	t.Run("a requester cannot force or close another user's request", func(t *testing.T) {
		f := newFlow(t)
		req := f.returnedRequest(t, 1)
		f.sensor(t, 1, 0)

		if _, err := f.svc.Close(ctx, f.user.ID, req.ID, true); !errors.Is(err, ErrNotAdmin) {
			t.Fatalf("forced by requester = %v, want ErrNotAdmin", err)
		}
		if _, err := f.svc.Close(ctx, f.other.ID, req.ID, false); !errors.Is(err, ErrRequestNotFound) {
			t.Fatalf("closed by someone else = %v, want ErrRequestNotFound", err)
		}
	})

	t.Run("an admin can close anyway and it is noted", func(t *testing.T) {
		f := newFlow(t)
		req := f.returnedRequest(t, 1)
		f.sensor(t, 1, 0)

		problems, err := f.svc.Close(ctx, f.admin.ID, req.ID, true)
		if err != nil || len(problems) != 0 {
			t.Fatalf("forced close = %v, %v", problems, err)
		}
		got, _ := f.store.GetRequest(ctx, req.ID)
		if got.ClosedAt == "" || !strings.Contains(got.CloseNote, "despite") {
			t.Fatalf("closed %q note %q", got.ClosedAt, got.CloseNote)
		}
	})

	t.Run("a request that is not returned cannot be closed", func(t *testing.T) {
		f := newFlow(t)
		req, err := f.svc.CreateForLocker(ctx, f.user.ID, f.locker.ID, []int64{1}, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Close(ctx, f.user.ID, req.ID, false); !errors.Is(err, ErrRequestClosed) {
			t.Fatalf("closing a pending request = %v, want ErrRequestClosed", err)
		}
	})
}
