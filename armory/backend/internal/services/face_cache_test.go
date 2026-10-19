package services

import (
	"context"
	"testing"
)

func TestFaceCacheNeverServesStaleData(t *testing.T) {
	ctx := context.Background()
	users, face := newFaceFixture(t)
	alice, err := users.Create(ctx, "Alice", "A-1", "requester")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := users.Create(ctx, "Bob", "B-1", "requester")
	if err != nil {
		t.Fatal(err)
	}

	match := func(at float64) MatchResult {
		res, err := face.Match(ctx, descriptorAt(at))
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	if match(0).Matched {
		t.Fatal("nobody is enrolled yet, so nobody should match")
	}
	if err := face.Enroll(ctx, alice.ID, [][]float64{descriptorAt(0)}); err != nil {
		t.Fatal(err)
	}
	if got := match(0); !got.Matched || got.User.Name != "Alice" {
		t.Fatalf("after enrolling Alice: %+v", got)
	}
	if err := face.Enroll(ctx, bob.ID, [][]float64{descriptorAt(5)}); err != nil {
		t.Fatal(err)
	}
	if got := match(5); !got.Matched || got.User.Name != "Bob" {
		t.Fatalf("a face enrolled after the cache was warm was not found: %+v", got)
	}

	if err := users.Rename(ctx, alice.ID, "Alicia"); err != nil {
		t.Fatal(err)
	}
	if got := match(0); !got.Matched || got.User.Name != "Alicia" {
		t.Fatalf("a rename was not picked up: %+v", got)
	}

	if err := users.Remove(ctx, bob.ID); err != nil {
		t.Fatal(err)
	}
	if got := match(5); got.Matched {
		t.Fatalf("a removed person still matches: %+v", got)
	}
	if got := match(0); !got.Matched || got.User.Name != "Alicia" {
		t.Fatalf("removing Bob broke Alicia: %+v", got)
	}

	if err := face.Enroll(ctx, alice.ID, [][]float64{descriptorAt(9)}); err != nil {
		t.Fatal(err)
	}
	if match(0).Matched {
		t.Fatal("Alice's old face still matches after she re-enrolled")
	}
	if got := match(9); !got.Matched || got.User.Name != "Alicia" {
		t.Fatalf("re-enrolled face not found: %+v", got)
	}
}

func TestFaceCacheSurvivesManyMatches(t *testing.T) {
	ctx := context.Background()
	users, face := newFaceFixture(t)
	u, err := users.Create(ctx, "Alice", "A-1", "requester")
	if err != nil {
		t.Fatal(err)
	}
	if err := face.Enroll(ctx, u.ID, [][]float64{descriptorAt(1)}); err != nil {
		t.Fatal(err)
	}
	face.Warm(ctx)
	for i := 0; i < 200; i++ {
		if res, err := face.Match(ctx, descriptorAt(1)); err != nil || !res.Matched {
			t.Fatalf("match %d: %+v %v", i, res, err)
		}
	}
}
