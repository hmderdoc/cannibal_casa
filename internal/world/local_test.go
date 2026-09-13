package world

import (
	"testing"
	"time"

	"cannibalcasa/internal/game"
)

func TestLocalNeighborhood(t *testing.T) {
	dir := t.TempDir()
	l := NewLocal(dir, game.DefaultRules)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	l.Now = func() time.Time { return now }

	a, ev, err := l.Join("bbs1", "Alice")
	if err != nil || a == nil || len(ev) < 2 {
		t.Fatalf("join alice: %v %v", err, ev)
	}
	if a.Bank != 3 {
		t.Fatalf("allowance not granted on first join: %d", a.Bank)
	}
	b, _, err := l.Join("bbs2", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	hood, err := l.Neighborhood("bbs1", "Alice")
	if err != nil || len(hood.Neighbors) != 2 || len(hood.Feed) != 2 {
		t.Fatalf("neighborhood: %v %+v", err, hood)
	}

	// Alice cooks nothing yet, so a casserole must fail; give her a dish.
	a.Dishes = append(a.Dishes, game.Dish{ID: 999, Recipe: "fritters", Quality: 4, Shelf: 3})
	l.Store.Save(Key("bbs1", "Alice"), a)
	_, r, err := l.Casserole("bbs1", "Alice", Key("bbs2", "Bob"), 999)
	if err != nil || r.Err != "" {
		t.Fatalf("casserole: %v %s", err, r.Err)
	}
	b, ev, _ = l.Join("bbs2", "Bob")
	if len(b.Dishes) != 1 || b.Dishes[0].Recipe != "fritters" {
		t.Fatalf("bob didn't get the casserole: %+v (%v)", b.Dishes, ev)
	}

	heat := b.Heat
	_, r, _ = l.Tip("bbs1", "Alice", Key("bbs2", "Bob"))
	if r.Err != "" {
		t.Fatalf("tip: %s", r.Err)
	}
	if _, r, _ = l.Tip("bbs1", "Alice", Key("bbs2", "Bob")); r.Err == "" {
		t.Fatal("second tip the same day should be refused")
	}
	b, _, _ = l.Join("bbs2", "Bob")
	if b.Heat != heat+10 {
		t.Fatalf("tip didn't land: %d -> %d", heat, b.Heat)
	}

	// Do persists and posts.
	a.Cash = 1000
	l.Store.Save(Key("bbs1", "Alice"), a)
	s, r, err := l.Do("bbs1", "Alice", game.Action{Kind: "build", Name: "cell_2"})
	if err != nil || r.Err != "" || !s.Has("cell_2") {
		t.Fatalf("do: %v %s", err, r.Err)
	}
	hood, _ = l.Neighborhood("bbs1", "Alice")
	if hood.Feed[len(hood.Feed)-1].Text == "" || hood.Feed[len(hood.Feed)-1].Handle != "Alice" {
		t.Fatalf("cell post missing: %+v", hood.Feed)
	}
	// Next real day: allowance again.
	now = now.Add(24 * time.Hour)
	a, _, _ = l.Join("bbs1", "Alice")
	if a.Bank != 6 {
		t.Fatalf("second-day allowance: %d", a.Bank)
	}
}
