package sensor

const (
	eventGunRemoved      = "gun_removed"
	eventGunReturned     = "gun_returned"
	eventSensorFault     = "sensor_fault"
	eventSensorRecovered = "sensor_recovered"
)

func eventFor(c Change) string {
	if c.To == 2 {
		return eventSensorFault
	}
	if c.From == 2 {
		return eventSensorRecovered
	}
	if c.To == 0 {
		return eventGunRemoved
	} else {
		return eventGunReturned
	}
}
