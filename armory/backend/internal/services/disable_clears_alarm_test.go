package services

import (
	"context"
	"testing"
)

func TestDisablingTheSensorClearsItsAlarm(t *testing.T) {
	ctx := context.Background()
	f := newFlow(t)
	lockers := NewLockerService(f.store)
	lockers.OnSensorToggled(f.svc.RefreshLockerAlarm)

	f.sensor(t, 1, 0)
	if !f.alarmOn() {
		t.Fatalf("a gun taken with no request should raise the alarm, alarms = %v", f.hw.alarms)
	}

	if _, err := lockers.SetSensorDisabled(ctx, f.locker.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	if f.alarmOn() {
		t.Fatalf("disabling the sensor should clear the alarm, alarms = %v", f.hw.alarms)
	}
}
