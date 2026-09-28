package sensor

type Change struct {
	Slot int
	From byte
	To   byte
}

const stableCount = 10

type slotState struct {
	stable  byte
	pending byte
	count   int
}

type Tracker struct {
	last map[string][]slotState
}

func NewTracker() *Tracker {
	return &Tracker{last: make(map[string][]slotState)}
}

func (t *Tracker) Update(board string, slots []byte) []Change {
	states, ok := t.last[board]
	if !ok {
		states = make([]slotState, len(slots))
		for i, v := range slots {
			states[i].stable = v
		}
		t.last[board] = states
		return nil
	}
	var changes []Change
	for i, v := range slots {
		s := &states[i]

		if v == s.stable {
			s.count = 0
		} else if v == s.pending {
			s.count++
			if s.count >= stableCount {
				changes = append(changes, Change{Slot: i, From: s.stable, To: v})
				s.stable = v
				s.count = 0
			}
		} else {
			s.pending = v
			s.count = 1
		}
	}
	return changes

}

func (t *Tracker) Seen(board string) bool {
	_, ok := t.last[board]
	return ok
}
