package world

import (
	"testing"
	"time"

	"cannibalcasa/internal/sim"
)

func TestLocalNeighborhood(t *testing.T) {
	dir := t.TempDir()
	l := NewLocal(dir, 60)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	l.Now = func() time.Time { return now }

	a, ev, _, err := l.Join("bbs1", "Alice")
	if err != nil || a == nil || len(ev) == 0 {
		t.Fatalf("join alice: %v %v", err, ev)
	}
	if _, _, _, err := l.Join("bbs2", "Bob"); err != nil {
		t.Fatal(err)
	}
	hood, err := l.Neighborhood("bbs1", "Alice")
	if err != nil || len(hood.Neighbors) != 2 || len(hood.Feed) != 2 {
		t.Fatalf("neighborhood: %v %+v", err, hood)
	}

	// The clock runs on wall time: a real minute at scale 60 is a game hour.
	before := a.Clock
	now = now.Add(time.Minute)
	a, _ = l.Sync("bbs1", "Alice", 1)
	if a.Clock != before+60 {
		t.Fatalf("sync: clock %d -> %d", before, a.Clock)
	}
	now = now.Add(time.Minute)
	a, _ = l.Sync("bbs1", "Alice", 0) // paused
	if a.Clock != before+60 {
		t.Fatal("paused sync advanced the clock")
	}

	// A casserole reaches Bob's pantry on his next join.
	dish := a.Pantry[0]
	if _, err := l.Casserole("bbs1", "Alice", Key("bbs2", "Bob"), dish.ID); err != nil {
		t.Fatalf("casserole: %v", err)
	}
	b, ev, _, _ := l.Join("bbs2", "Bob")
	found := false
	for _, e := range ev {
		if e.Kind == "food" {
			found = true
		}
	}
	if !found || len(b.Pantry) != 3 {
		t.Fatalf("bob didn't get the casserole: %d dishes (%v)", len(b.Pantry), ev)
	}

	heat := b.Heat
	l.Tip("bbs1", "Alice", Key("bbs2", "Bob"))
	l.Leave("bbs2", "Bob")
	b, _, _, _ = l.Join("bbs2", "Bob")
	if b.Heat != heat+10 {
		t.Fatalf("tip didn't land: %.0f -> %.0f", heat, b.Heat)
	}

	// Commands persist and post.
	a.Cash = 1000
	w, err := l.Do("bbs1", "Alice", sim.Command{Kind: "build", Name: "cell_2", At: &sim.Pos{L: sim.Basement, X: 8, Y: 1}})
	if err != nil || !w.Has("cell_2") {
		t.Fatalf("do: %v", err)
	}
	hood, _ = l.Neighborhood("bbs1", "Alice")
	last := hood.Feed[len(hood.Feed)-1]
	if last.Handle != "Alice" || last.Text == "" {
		t.Fatalf("cell post missing: %+v", hood.Feed)
	}
	// A raid on Bob settles now and lands when Alice gets home.
	a.Cash = 500
	if _, err := l.Raid("bbs1", "Alice", Key("bbs2", "Bob")); err != nil {
		t.Fatalf("raid: %v", err)
	}
	if _, err := l.Raid("bbs1", "Alice", Key("bbs2", "Bob")); err == nil {
		t.Fatal("second raid the same day should be refused")
	}
	if a.Player().Away == nil {
		t.Fatal("alice should be out raiding")
	}
	// Leave saves; a fresh Local reloads the same clock.
	l.Leave("bbs1", "Alice")
	l2 := NewLocal(dir, 60)
	l2.Now = func() time.Time { return now }
	a2, _, ticks, _ := l2.Join("bbs1", "Alice")
	if a2.Clock != a.Clock || ticks != 0 || !a2.Has("cell_2") {
		t.Fatalf("reload: clock %d vs %d, ticks %d", a2.Clock, a.Clock, ticks)
	}
}
