package sim

import (
	"encoding/json"
	"testing"
	"time"
)

// A fresh house, left alone for ten days with $150 and a welcome mat: the
// machine must capture, butcher, cook and feed its owner without a single
// command. This is the "it runs whether you watch or not" guarantee.
func TestMachineRunsUnattended(t *testing.T) {
	best := 0
	for seed := int64(1); seed <= 6; seed++ {
		w := New("t", "b", seed)
		w.Advance(10 * day)
		if w.Stats.Captured > 0 && w.Stats.Butchered > 0 && w.Stats.Meals > 0 {
			best++
		}
		if !w.Alive {
			t.Logf("seed %d: died day %d: %s", seed, w.DaysAlive(), w.Cause)
		}
	}
	if best < 4 {
		t.Fatalf("only %d/6 seeds ran the whole intake->plate loop unattended", best)
	}
}

func TestWorkersNeverStrand(t *testing.T) {
	w := New("t", "b", 3)
	w.Cash = 3000
	for _, id := range []string{"lure_couch", "lure_wifi", "cell_2", "cell_3", "grinder", "stand_big", "microwave", "compost", "stockpot", "chair", "cot", "freezer", "smoker", "fence"} {
		if err := w.Build(id, w.freeTile(FixtureByID(id))); err != nil {
			t.Fatalf("build %s: %v", id, err)
		}
	}
	w.Policy.Brainwash = true
	for d := 0; d < 20; d++ {
		w.Advance(day)
		for _, wk := range w.Workers {
			if wk.Job != nil && wk.Job.Step >= len(wk.Job.Steps) {
				t.Fatalf("worker %s stuck past the end of job %s", wk.Name, wk.Job.Kind)
			}
			if !walkable(wk.Pos) {
				t.Fatalf("worker %s is inside a wall at %v", wk.Name, wk.Pos)
			}
		}
		for id, m := range w.Machines {
			if m.Reserved {
				held := false
				for _, wk := range w.Workers {
					if wk.Job != nil && wk.Job.Machine == id {
						held = true
					}
				}
				if !held {
					t.Fatalf("day %d: machine %s reserved by nobody", d, id)
				}
			}
		}
	}
	if w.Stats.Captured == 0 || w.Stats.Butchered == 0 {
		t.Fatalf("nothing happened: %+v", w.Stats)
	}
	t.Logf("20 days: %+v renfields=%d heat=%.0f cash=%.0f alive=%v", w.Stats, w.Renfields(), w.Heat, w.Cash, w.Alive)
}

func TestSaveLoadMidJobRoundTrip(t *testing.T) {
	w := New("t", "b", 5)
	w.Advance(3 * day)
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	var w2 World
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatal(err)
	}
	w.Advance(day)
	w2.Advance(day)
	a, _ := json.Marshal(w)
	b, _ := json.Marshal(&w2)
	if string(a) != string(b) {
		t.Fatal("a reloaded world diverged from the original over one day")
	}
}

func TestCatchupUsesWallClock(t *testing.T) {
	w := New("t", "b", 9)
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	w.Catchup(now)
	before := w.Clock
	if n := w.Catchup(now.Add(time.Minute)); n != 60 || w.Clock != before+60 {
		t.Fatalf("one real minute at scale 60 should be 60 ticks, got %d (clock %d -> %d)", n, before, w.Clock)
	}
}

func TestRaidAndInheritance(t *testing.T) {
	w := New("t", "b", 11)
	w.Heat = 100
	w.Advance(3 * 60)
	if w.Alive {
		t.Fatalf("expected a raid; visitors=%d heat=%.0f", len(w.Visitors), w.Heat)
	}
	if err := w.Inherit(); err != nil {
		t.Fatal(err)
	}
	if !w.Alive || w.Generation != 2 || len(w.Workers) != 1 || w.Player() == nil {
		t.Fatalf("inheritance wrong: alive=%v gen=%d workers=%d", w.Alive, w.Generation, len(w.Workers))
	}
	w.Advance(day)
}

func TestBuildReplacesTrap(t *testing.T) {
	w := New("t", "b", 1)
	w.Cash = 1000
	mat := w.Pos("welcome_mat")
	if err := w.Build("trapdoor_pro", Pos{}); err != nil {
		t.Fatal(err)
	}
	if w.Has("welcome_mat") || !w.Has("trapdoor_pro") || w.Pos("trapdoor_pro") != mat {
		t.Fatal("upgrade should replace the mat on its tile")
	}
	if err := w.Build("chest_freezer", Pos{Ground, 20, 3}); err == nil {
		t.Fatal("prereq should block")
	}
	if err := w.Build("lure_couch", Pos{Ground, 20, 7}); err == nil {
		t.Fatal("lures must go on the porch")
	}
	if err := w.Build("cell_2", Pos{Ground, 20, 7}); err == nil {
		t.Fatal("cells must go in the basement")
	}
	if err := w.Build("lure_couch", w.Pos("stand")); err == nil {
		t.Fatal("can't stack on the stand")
	}
	if err := w.Move("fridge", Pos{Ground, 30, 7}); err != nil {
		t.Fatal(err)
	}
	if w.Pos("fridge") != (Pos{Ground, 30, 7}) {
		t.Fatal("move didn't stick")
	}
}

func TestErrandsRoundTrip(t *testing.T) {
	w := New("t", "b", 4)
	cash := w.Cash
	if err := w.StartErrand("work"); err != nil {
		t.Fatal(err)
	}
	if err := w.StartErrand("scrap"); err == nil {
		t.Fatal("can't leave twice")
	}
	p := w.Player()
	if p.Away == nil {
		t.Fatal("not away")
	}
	w.Advance(4*60 + 5)
	if p.Away != nil || w.Cash <= cash {
		t.Fatalf("didn't come home paid: away=%v cash %.0f -> %.0f", p.Away, cash, w.Cash)
	}
	w.Advance(day)
	e := &Errand{Kind: "raid", Label: "raiding a neighbor", Cash: 50, Heat: 0, Text: "Got in and out.", TextKind: "cash"}
	if err := w.StartRaid(e); err != nil {
		t.Fatal(err)
	}
	w.Advance(5*60 + 5)
	if w.Stats.Raids != 1 || w.Player().Away != nil {
		t.Fatal("raid didn't resolve")
	}
}

// freeTile finds a legal tile for a fixture, for tests.
func (w *World) freeTile(f *Fixture) Pos {
	p := Plans[f.Level]
	for y, row := range p.Rows {
		for x := range row {
			pos := Pos{f.Level, x, y}
			if w.CanPlace(f, pos) == nil {
				return pos
			}
		}
	}
	return Pos{}
}

func TestDirectOrders(t *testing.T) {
	w := New("t", "b", 21)
	w.Cash = 2000
	w.Build("chair", Pos{Ground, 20, 11})
	w.Build("cot", Pos{Attic, 3, 1})
	w.Build("cell_2", Pos{Basement, 8, 1})
	w.Policy.Brainwash = false
	w.Policy.ButcherMarbling = 5
	w.Livestock = append(w.Livestock,
		&Livestock{ID: w.newID(), Victim: "crossfit", Name: "A", Cell: "cell_1", Marbling: 1, Since: w.Clock},
		&Livestock{ID: w.newID(), Victim: "mlm", Name: "B", Cell: "cell_2", Marbling: 3, Since: w.Clock})
	if err := w.Order(w.Livestock[0].ID, "chair"); err != nil {
		t.Fatal(err)
	}
	w.Advance(16 * 60)
	renfield := false
	for _, wk := range w.Workers {
		if wk.Name == "A" && wk.Kind == "renfield" {
			renfield = true
		}
	}
	if !renfield {
		t.Fatalf("the ordered guest never became a Renfield: workers=%d livestock=%d chair=%+v", len(w.Workers), len(w.Livestock), w.Machines["chair"])
	}
	if err := w.Order(w.Livestock[0].ID, "butcher"); err != nil {
		t.Fatal(err)
	}
	w.Advance(6 * 60)
	if len(w.Livestock) != 0 {
		t.Fatalf("the ordered guest was not butchered: %d left", len(w.Livestock))
	}
}

func TestReset(t *testing.T) {
	w := New("t", "b", 5)
	w.Cash = 900
	w.Build("cell_2", Pos{Basement, 8, 1})
	w.Heat = 100
	w.Advance(3 * 60)
	w.Inherit()
	w.ColorMode = "256"
	w.Reset()
	if !w.Alive || w.Generation != 1 || len(w.Ancestors) != 0 || w.Has("cell_2") || w.Cash != 150 || w.ColorMode != "256" || w.Handle != "t" {
		t.Fatalf("reset wrong: gen=%d anc=%d cell2=%v cash=%.0f color=%q", w.Generation, len(w.Ancestors), w.Has("cell_2"), w.Cash, w.ColorMode)
	}
	w.Advance(day)
}
