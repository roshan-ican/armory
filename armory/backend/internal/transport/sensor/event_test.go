package sensor

import "testing"

func TestEventFor(t *testing.T) {
	tests := []struct {
		from, to byte
		want     string
	}{
		{1, 0, eventGunRemoved},
		{0, 1, eventGunReturned},
		{1, 2, eventSensorFault},
		{0, 2, eventSensorFault},
		{2, 0, eventSensorRecovered},
		{2, 1, eventSensorRecovered},
	}

	for _, tc := range tests {
		got := eventFor(Change{Slot: 0, From: tc.from, To: tc.to})
		if got != tc.want {
			t.Errorf("%d -> %d: got %q, want %q", tc.from, tc.to, got, tc.want)
		}
	}
}
