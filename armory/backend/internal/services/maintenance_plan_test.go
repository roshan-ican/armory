package services

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"armory/internal/database"
	"armory/internal/live"
	"armory/internal/models"
)

type unlockRecorder struct {
	mu      sync.Mutex
	doors   int
	unlocks []string
}

func (h *unlockRecorder) Unlock(ctx context.Context, lockerIP string, slotNo int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unlocks = append(h.unlocks, lockerIP)
	return nil
}

func (h *unlockRecorder) Signal(ctx context.Context, lockerIP string, signal Signal) error {
	return nil
}

func (h *unlockRecorder) Alarm(ctx context.Context, lockerIP string, on bool) error { return nil }

func (h *unlockRecorder) OpenDoor(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.doors++
	return nil
}

func (h *unlockRecorder) seen() (int, []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.doors, append([]string(nil), h.unlocks...)
}

type planFixture struct {
	svc      *MaintenanceService
	requests *RequestService
	store    *database.Store
	hub      *live.Hub
	hw       *unlockRecorder
	worker   models.User
	other    models.User
	admin    models.User
	l1, l2   models.Locker
}

func newPlanFixture(t *testing.T) planFixture {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := database.NewStore(db)
	hub := live.NewHub()
	hw := &unlockRecorder{}

	worker, err := store.CreateUser(ctx, models.User{Name: "Fixer", ServiceNo: "MT-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateUser(ctx, models.User{Name: "Other", ServiceNo: "OT-1", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := store.CreateUser(ctx, models.User{Name: "Admin", ServiceNo: "AD-1", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	l1, err := store.CreateLocker(ctx, models.Locker{Name: "Locker 1", IPAddress: "10.0.0.1", Kind: "rifle", Capacity: 3})
	if err != nil {
		t.Fatal(err)
	}
	l2, err := store.CreateLocker(ctx, models.Locker{Name: "Locker 2", IPAddress: "10.0.0.2", Kind: "rifle", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	return planFixture{
		svc: NewMaintenanceService(store, hw, hub), requests: NewRequestService(store, hw, hub),
		store: store, hub: hub, hw: hw, worker: worker, other: other, admin: admin, l1: l1, l2: l2,
	}
}

func (f planFixture) readings(t *testing.T, lockerID int64, values ...byte) {
	t.Helper()
	l, err := f.store.GetLocker(context.Background(), lockerID)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range values {
		if err := f.store.RecordSensorEvent(context.Background(), l.Slots[i].ID, v, "test_reading"); err != nil {
			t.Fatal(err)
		}
	}
}

func (f planFixture) startJob(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	job, err := f.svc.Request(ctx, f.worker.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Approve(ctx, f.admin.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	if began, err := f.svc.Begin(ctx, f.worker.ID); err != nil || !began {
		t.Fatalf("begin = %v, %v", began, err)
	}
}

func TestMaintenanceRequestNeedsNoPlan(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)

	job, err := f.svc.Request(ctx, f.worker.ID, nil)
	if err != nil || job.Status != models.MaintenanceRequested {
		t.Fatalf("job = %+v, %v", job, err)
	}
	if _, err := f.svc.Request(ctx, f.worker.ID, nil); !errors.Is(err, ErrMaintenanceOpen) {
		t.Fatalf("second request = %v, want ErrMaintenanceOpen", err)
	}
	if _, err := f.svc.Request(ctx, f.admin.ID, nil); !errors.Is(err, ErrNotRequester) {
		t.Fatalf("admin request = %v, want ErrNotRequester", err)
	}
	if _, err := f.svc.Request(ctx, 9999, nil); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown user = %v, want ErrUserNotFound", err)
	}
}

func TestMaintenanceRequestStillRejectsABadOptionalPlan(t *testing.T) {
	ctx := context.Background()
	cases := map[string][]models.MaintenanceChange{
		"unknown locker":  {{LockerID: 999, Action: models.ChangeAddSensor}},
		"unknown action":  {{LockerID: 1, Action: "explode"}},
		"replace no slot": {{LockerID: 2, Action: models.ChangeReplaceSensor}},
		"add beyond five": {{LockerID: 1, Action: models.ChangeAddSensor}, {LockerID: 1, Action: models.ChangeAddSensor}, {LockerID: 1, Action: models.ChangeAddSensor}},
	}
	for name, changes := range cases {
		t.Run(name, func(t *testing.T) {
			f := newPlanFixture(t)
			if _, err := f.svc.Request(ctx, f.worker.ID, changes); err == nil {
				t.Fatal("a bad plan was accepted")
			}
			if _, err := f.store.CurrentMaintenanceFor(ctx, f.worker.ID); !errors.Is(err, database.ErrNotFound) {
				t.Fatalf("a refused request left a job behind: %v", err)
			}
		})
	}
}

func TestMaintenanceApproveNeedsAnAdmin(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	job, err := f.svc.Request(ctx, f.worker.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Approve(ctx, f.worker.ID, job.ID); !errors.Is(err, ErrNotAdmin) {
		t.Fatalf("approving own request = %v, want ErrNotAdmin", err)
	}
	if began, err := f.svc.Begin(ctx, f.worker.ID); err != nil || began {
		t.Fatalf("begin before approval = %v, %v", began, err)
	}
	if err := f.svc.Approve(ctx, f.admin.ID, 999); !errors.Is(err, ErrMaintenanceNotFound) {
		t.Fatalf("approve unknown = %v", err)
	}
	if err := f.svc.Approve(ctx, f.admin.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	if began, err := f.svc.Begin(ctx, f.worker.ID); err != nil || !began {
		t.Fatalf("begin after approval = %v, %v", began, err)
	}
}

func TestMaintenanceOpensTheDoorAndUnlocksEveryLocker(t *testing.T) {
	f := newPlanFixture(t)
	f.startJob(t)
	doors, unlocks := f.hw.seen()
	if doors != 1 || len(unlocks) != 2 {
		t.Fatalf("doors = %d, unlocks = %v", doors, unlocks)
	}
	got := map[string]bool{}
	for _, ip := range unlocks {
		got[ip] = true
	}
	if !got["10.0.0.1"] || !got["10.0.0.2"] {
		t.Fatalf("unlocked %v, want both lockers", unlocks)
	}

	ctx := context.Background()
	if err := f.svc.UnlockLocker(ctx, f.worker.ID, f.l2.ID); err != nil {
		t.Fatal(err)
	}
	if _, unlocks := f.hw.seen(); len(unlocks) != 3 || unlocks[2] != "10.0.0.2" {
		t.Fatalf("unlock again = %v", unlocks)
	}
	if err := f.svc.UnlockLocker(ctx, f.worker.ID, 999); !errors.Is(err, ErrLockerNotFound) {
		t.Fatalf("unknown locker = %v", err)
	}
	if err := f.svc.UnlockLocker(ctx, f.other.ID, f.l1.ID); !errors.Is(err, ErrMaintenanceNotFound) {
		t.Fatalf("someone not in maintenance = %v", err)
	}
}

func TestMaintenanceFinishKeepsWhateverWasDone(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	f.readings(t, f.l1.ID, 1, 1, 1)
	f.readings(t, f.l2.ID, 1, 1)
	f.startJob(t)

	spoken, cancel := f.hub.SubscribeSpeech()
	defer cancel()

	if err := f.svc.SetCapacity(ctx, f.worker.ID, f.l2.ID, 5); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetCapacity(ctx, f.worker.ID, f.l1.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetCapacity(ctx, f.worker.ID, f.l1.ID, 9); !errors.Is(err, ErrBadChange) {
		t.Fatalf("capacity 9 = %v", err)
	}
	if err := f.svc.SetCapacity(ctx, f.other.ID, f.l1.ID, 2); !errors.Is(err, ErrMaintenanceNotFound) {
		t.Fatalf("someone not in maintenance changing capacity = %v", err)
	}
	f.readings(t, f.l2.ID, 0, 0)

	if err := f.svc.Finish(ctx, f.worker.ID); err != nil {
		t.Fatalf("finish = %v", err)
	}
	select {
	case said := <-spoken:
		if said != MaintenanceDoneMessage() {
			t.Fatalf("said %q", said)
		}
	case <-time.After(time.Second):
		t.Fatal("nothing was spoken")
	}

	l1, _ := f.store.GetLocker(ctx, f.l1.ID)
	l2, _ := f.store.GetLocker(ctx, f.l2.ID)
	if l1.Capacity != 1 || l2.Capacity != 5 {
		t.Fatalf("capacities after finishing = %d and %d, want 1 and 5", l1.Capacity, l2.Capacity)
	}
	if _, err := f.svc.Current(ctx, f.worker.ID); !errors.Is(err, ErrMaintenanceNotFound) {
		t.Fatalf("job still open: %v", err)
	}
	if err := f.svc.Finish(ctx, f.worker.ID); !errors.Is(err, ErrMaintenanceNotFound) {
		t.Fatalf("finishing twice = %v", err)
	}
}

func TestGunRequestsAreBlockedWhileSomeoneIsInMaintenance(t *testing.T) {
	ctx := context.Background()
	f := newPlanFixture(t)
	f.readings(t, f.l1.ID, 1, 1, 1)

	job, err := f.svc.Request(ctx, f.worker.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.requests.Catalog(ctx); err != nil {
		t.Fatalf("catalog while only requested = %v", err)
	}
	if err := f.svc.Approve(ctx, f.admin.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	early, err := f.requests.Create(ctx, f.other.ID, "rifle", "")
	if err != nil {
		t.Fatalf("approved but not started must not block: %v", err)
	}
	if err := f.requests.Cancel(ctx, f.other.ID, early.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Begin(ctx, f.worker.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := f.requests.Create(ctx, f.other.ID, "rifle", ""); !errors.Is(err, ErrMaintenanceActive) {
		t.Fatalf("Create = %v, want ErrMaintenanceActive", err)
	}
	if _, err := f.requests.CreateForLocker(ctx, f.other.ID, f.l1.ID, []int64{1}, ""); !errors.Is(err, ErrMaintenanceActive) {
		t.Fatalf("CreateForLocker = %v, want ErrMaintenanceActive", err)
	}
	if _, err := f.requests.Catalog(ctx); !errors.Is(err, ErrMaintenanceActive) {
		t.Fatalf("Catalog = %v, want ErrMaintenanceActive", err)
	}
	if _, err := f.requests.CreateForLocker(ctx, f.worker.ID, f.l1.ID, []int64{1}, ""); !errors.Is(err, ErrMaintenanceActive) {
		t.Fatalf("the maintainer asking for a gun = %v, want ErrMaintenanceActive", err)
	}

	if err := f.svc.Finish(ctx, f.worker.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.requests.CreateForLocker(ctx, f.other.ID, f.l1.ID, []int64{1}, ""); err != nil {
		t.Fatalf("after finishing, requests must work again: %v", err)
	}
}
