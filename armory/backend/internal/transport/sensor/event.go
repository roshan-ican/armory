package sensor

const (
	eventGunRemoved      = "gun_removed"
	eventGunReturned     = "gun_returned"
	eventSensorFault     = "sensor_fault"
	eventSensorRecovered = "sensor_recovered"
)

const (
	statusIn    = "in"
	statusOut   = "out"
	statusFault = ""
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

func statusFor(c Change) string {
	if c.To == 2 {
		return statusFault
	}
	if c.To == 0 {
		return statusOut
	}
	return statusIn
}
