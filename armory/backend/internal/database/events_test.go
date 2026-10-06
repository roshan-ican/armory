package database

import (
	"context"
	"testing"

	"armory/internal/models"
)

func TestEventsShowWhoAndFilter(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	l := newTestLocker(t, st, 3)

	for _, sl := range l.Slots {
		if err := st.RecordSensorEvent(ctx, sl.ID, 1, "gun_returned"); err != nil {
			t.Fatal(err)
		}
	}
	ann, err := st.CreateUser(ctx, models.User{Name: "Ann", ServiceNo: "ANN", Role: "requester"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := st.CreateUser(ctx, models.User{Name: "Boss", ServiceNo: "BOSS", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	reqID, err := st.CreateRequestForLocker(ctx, ann.ID, l.ID, "rifle", "", "", []int64{1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ApproveRequest(ctx, reqID, admin.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSensorEvent(ctx, l.Slots[0].ID, 0, "gun_removed"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSensorEvent(ctx, l.Slots[1].ID, 0, "gun_removed"); err != nil {
		t.Fatal(err)
	}

	all, err := st.ListEventsPage(ctx, EventFilter{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	who := map[int64]string{}
	for _, e := range all {
		if e.Type == "gun_removed" {
			who[e.SlotNo] = e.Person
		}
	}
	if who[1] != "Ann" || who[2] != "" {
		t.Fatalf("who took: %v, want slot 1 Ann and slot 2 nobody", who)
	}

	byPerson, err := st.ListEventsPage(ctx, EventFilter{Person: "Ann", Type: "gun_removed"}, 50, 0)
	if err != nil || len(byPerson) != 1 || byPerson[0].SlotNo != 1 {
		t.Fatalf("person filter: %v %v", byPerson, err)
	}
	n, err := st.CountEvents(ctx, EventFilter{Type: "gun_removed"})
	if err != nil || n != 2 {
		t.Fatalf("type filter count = %d, %v", n, err)
	}
	n, err = st.CountEvents(ctx, EventFilter{Locker: "nowhere"})
	if err != nil || n != 0 {
		t.Fatalf("locker filter count = %d, %v", n, err)
	}
	n, err = st.CountEvents(ctx, EventFilter{From: "2999-01-01"})
	if err != nil || n != 0 {
		t.Fatalf("future date count = %d, %v", n, err)
	}
	people, err := st.EventPeople(ctx)
	if err != nil || len(people) != 1 || people[0] != "Ann" {
		t.Fatalf("people = %v, %v", people, err)
	}
}
