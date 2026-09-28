package sensor

const (
	frameLen    = 6
	frameHeader = 0x24
	frameTail   = 0x23
	slotCount   = 3
	maxState    = 2
)

func decodeSensorFrame(b []byte) ([]byte, error) {
	if len(b) != frameLen {
		return nil, errFrameLength
	}
	if b[0] != frameHeader || b[frameLen-1] != frameTail {
		return nil, errFrameBytes
	}
	slots := b[1 : 1+slotCount]

	for _, s := range slots {
		if s > maxState {
			return nil, errSlotState
		}
	}

	out := make([]byte, slotCount)
	copy(out, slots)
	return out, nil
}
