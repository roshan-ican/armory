package sensor

import "testing"

const board = "10.252.176.50"

func feed(tr *Tracker, slots []byte, times int) []Change {
	var all []Change
	for i := 0; i < times; i++ {
		all = append(all, tr.Update(board, slots)...)
	}
	return all
}

func TestTrackerFirstPacketReportsNothing(t *testing.T) {
	tr := NewTracker()
	if got := tr.Update(board, []byte{2, 0, 1}); got != nil {
		t.Fatalf("first packet: got %v, want nil", got)
	}
}

func TestTrackerSamePacketReportsNothing(t *testing.T) {
	tr := NewTracker()
	if got := feed(tr, []byte{2, 0, 1}, 50); len(got) != 0 {
		t.Fatalf("same packet: got %v, want no changes", got)
	}
}

func TestTrackerChangeNeedsStableCountPackets(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})

	if got := feed(tr, []byte{1, 0, 1}, stableCount-1); len(got) != 0 {
		t.Fatalf("reported after %d packets, want %d: %v", stableCount-1, stableCount, got)
	}

	got := tr.Update(board, []byte{1, 0, 1})
	want := Change{Slot: 1, From: 1, To: 0}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("on packet %d: got %+v, want [%+v]", stableCount, got, want)
	}
}

func TestTrackerReportsChangeOnlyOnce(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})
	if got := feed(tr, []byte{1, 0, 1}, stableCount*5); len(got) != 1 {
		t.Fatalf("got %d changes, want 1: %v", len(got), got)
	}
}

func TestTrackerIgnoresJitter(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{2, 2, 1})
	jitter := [][]byte{
		{2, 2, 0}, {2, 2, 1}, {2, 2, 0}, {2, 2, 1}, {2, 2, 0},
		{2, 2, 1}, {2, 2, 0}, {2, 2, 0}, {2, 2, 1}, {2, 2, 0},
		{2, 2, 1}, {2, 2, 1}, {2, 2, 0}, {2, 2, 1},
	}
	for i, p := range jitter {
		if got := tr.Update(board, p); len(got) != 0 {
			t.Fatalf("packet %d reported jitter as change: %v", i, got)
		}
	}
}

func TestTrackerJitterResetsCount(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})
	feed(tr, []byte{1, 1, 0}, stableCount-1)
	tr.Update(board, []byte{1, 1, 1})

	if got := feed(tr, []byte{1, 1, 0}, stableCount-1); len(got) != 0 {
		t.Fatalf("count was not reset by jitter: %v", got)
	}
}

func TestTrackerNewSuspectRestartsCount(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})
	feed(tr, []byte{1, 1, 0}, stableCount-1)

	if got := feed(tr, []byte{1, 1, 2}, stableCount-1); len(got) != 0 {
		t.Fatalf("switching suspect kept old count: %v", got)
	}
	got := tr.Update(board, []byte{1, 1, 2})
	want := Change{Slot: 2, From: 1, To: 2}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want [%+v]", got, want)
	}
}

func TestTrackerTwoSlotsChangeTogether(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})
	got := feed(tr, []byte{0, 1, 2}, stableCount)

	want := []Change{{Slot: 0, From: 1, To: 0}, {Slot: 2, From: 1, To: 2}}
	if len(got) != len(want) {
		t.Fatalf("got %d changes, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("change %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestTrackerBoardsAreSeparate(t *testing.T) {
	tr := NewTracker()
	tr.Update(board, []byte{1, 1, 1})
	if got := tr.Update("10.252.176.51", []byte{0, 0, 0}); got != nil {
		t.Fatalf("new board's first packet: got %v, want nil", got)
	}
	if got := feed(tr, []byte{1, 1, 1}, stableCount); len(got) != 0 {
		t.Fatalf("board A affected by board B: %v", got)
	}
}

func TestTrackerSeen(t *testing.T) {
	tr := NewTracker()
	if tr.Seen(board) {
		t.Fatal("new tracker should not have seen the board")
	}
	tr.Update(board, []byte{1, 1, 1})
	if !tr.Seen(board) {
		t.Fatal("board should be seen after first packet")
	}
}
