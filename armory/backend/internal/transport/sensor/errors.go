package sensor

import "errors"

var (
	errFrameLength = errors.New("sensor frame: wrong length")
	errFrameBytes  = errors.New("sensor frame: bad header or tail")
	errSlotState   = errors.New("sensor frame: invalid slot state")
)
