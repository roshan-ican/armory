package services

import (
	"context"
	"errors"
	"testing"

	"armory/internal/models"
)

func TestMaintenanceRequestBlockedWhileSomeoneElseIsActive(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	f.startJob(t)

	if _, err := f.svc.Request(ctx, f.other.ID, nil); !errors.Is(err, ErrMaintenanceBusy) {
		t.Fatalf("other user's request = %v, want ErrMaintenanceBusy", err)
	}
	if _, err := f.svc.Request(ctx, f.worker.ID, nil); !errors.Is(err, ErrMaintenanceOpen) {
		t.Fatalf("the active user's own request = %v, want ErrMaintenanceOpen", err)
	}
}

func TestMaintenanceRequestAllowedAfterTheActiveJobFinishes(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	f.startJob(t)

	if err := f.svc.Finish(ctx, f.worker.ID); err != nil {
		t.Fatal(err)
	}
	job, err := f.svc.Request(ctx, f.other.ID, nil)
	if err != nil || job.Status != models.MaintenanceRequested {
		t.Fatalf("request after finish = %+v, %v", job, err)
	}
}

func TestMaintenanceRequestAllowedWhileAnotherJobIsOnlyRequested(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)

	if _, err := f.svc.Request(ctx, f.worker.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Request(ctx, f.other.ID, nil); err != nil {
		t.Fatalf("second person's request with no active job = %v, want nil", err)
	}
}
