package game

import (
	"strings"
	"testing"
)

func must(t *testing.T, r Result) Result {
	t.Helper()
	if r.Err != "" {
		t.Fatalf("unexpected error: %s", r.Err)
	}
	return r
}

func toNight(t *testing.T, s *State) {
	for s.Phase != Night {
		must(t, Apply(s, Action{Kind: "end_phase"}))
	}
}

// A full first night: hunt, capture (or fail), butcher, cook, eat, tidy up.
func TestFirstNightLoop(t *testing.T) {
	s := NewState("tester", "bbs", 7)
	if !GrantAllowance(s, DefaultRules, "2026-09-13") || s.Bank != 3 {
		t.Fatalf("allowance not granted: bank=%d", s.Bank)
	}
	if GrantAllowance(s, DefaultRules, "2026-09-13") {
		t.Fatal("double grant on the same day")
	}
	if r := Apply(s, Action{Kind: "hunt", Name: "solar"}); r.Err == "" {
		t.Fatal("hunting in the morning should fail")
	}
	toNight(t, s)
	if len(s.Prey) == 0 {
		t.Fatal("no prey rolled at night")
	}
	// Try seeds until a hunt succeeds so the rest of the loop is exercised.
	var captured bool
	for seed := int64(1); seed < 40 && !captured; seed++ {
		s = NewState("tester", "bbs", seed)
		GrantAllowance(s, DefaultRules, "2026-09-13")
		toNight(t, s)
		must(t, Apply(s, Action{Kind: "hunt", Name: s.Prey[0]}))
		captured = len(s.Captives) > 0
	}
	if !captured {
		t.Fatal("no successful hunt in 40 seeds; chance table is broken")
	}
	if len(s.Prey) != 0 {
		t.Fatal("prey should clear after a hunt")
	}
	c := s.Captives[0]
	must(t, Apply(s, Action{Kind: "butcher", ID: c.ID}))
	if len(s.Fridge) < 4 {
		t.Fatalf("expected cuts in the fridge, got %d", len(s.Fridge))
	}
	if len(s.Evidence) < 3 {
		t.Fatalf("expected evidence, got %d", len(s.Evidence))
	}
	must(t, Apply(s, Action{Kind: "cook", Name: "pan_sear"}))
	if len(s.Dishes) != 1 {
		t.Fatalf("expected a dish, got %d", len(s.Dishes))
	}
	h := s.Hunger
	must(t, Apply(s, Action{Kind: "eat", ID: s.Dishes[0].ID}))
	if s.Hunger <= h {
		t.Fatal("eating did not restore hunger")
	}
	// Bury bones (plot_1 holds 2), donate clothes, leave the phone (no microwave).
	for _, e := range append([]Evidence(nil), s.Evidence...) {
		switch e.Kind {
		case "bones":
			must(t, Apply(s, Action{Kind: "dispose", ID: e.ID, Name: "bury"}))
		case "clothes":
			must(t, Apply(s, Action{Kind: "dispose", ID: e.ID, Name: "donate"}))
		}
	}
	if r := Apply(s, Action{Kind: "dispose", ID: s.Evidence[0].ID, Name: "microwave"}); r.Err == "" {
		t.Fatal("microwaving without a microwave should fail")
	}
	// Roll the day; the bank spends.
	bank := s.Bank
	must(t, Apply(s, Action{Kind: "end_phase"}))
	if s.Day != 2 || s.Bank != bank-1 || s.Phase != Morning {
		t.Fatalf("day roll wrong: day=%d bank=%d phase=%v", s.Day, s.Bank, s.Phase)
	}
}

func TestBankRunsOut(t *testing.T) {
	s := NewState("tester", "bbs", 3)
	GrantAllowance(s, Rules{DaysPerDay: 1, BankMax: 1}, "2026-09-13")
	toNight(t, s)
	must(t, Apply(s, Action{Kind: "end_phase"}))
	toNight(t, s)
	if r := Apply(s, Action{Kind: "end_phase"}); r.Err == "" || !strings.Contains(r.Err, "out of days") {
		t.Fatalf("expected out-of-days error, got %q", r.Err)
	}
}

func TestBuildAndRecipeUnlock(t *testing.T) {
	s := NewState("tester", "bbs", 1)
	s.Cash = 1000
	if r := Apply(s, Action{Kind: "build", Name: "chest_freezer"}); r.Err == "" {
		t.Fatal("prereq should block chest freezer")
	}
	r := must(t, Apply(s, Action{Kind: "build", Name: "grinder"}))
	found := false
	for _, e := range r.Events {
		if strings.Contains(e.Text, "New recipe") {
			found = true
		}
	}
	if !found || !s.Recipes["meat_pie"] {
		t.Fatal("grinder should unlock meat pies")
	}
	if s.Cash != 1000-80 {
		t.Fatalf("cash not charged: %d", s.Cash)
	}
}

func TestMissionBlocksAndResolves(t *testing.T) {
	s := NewState("tester", "bbs", 5)
	GrantAllowance(s, DefaultRules, "2026-09-13")
	s.Cash = 500
	must(t, Apply(s, Action{Kind: "mission", Name: "burglary"}))
	if r := Apply(s, Action{Kind: "build", Name: "blender"}); r.Err == "" {
		t.Fatal("building while away should fail")
	}
	must(t, Apply(s, Action{Kind: "end_phase"}))
	if s.Mission != nil {
		t.Fatal("one-phase mission should resolve")
	}
	if s.Cash <= 500 {
		t.Fatal("burglary should pay")
	}
}

func TestDeathAndInheritance(t *testing.T) {
	s := NewState("tester", "bbs", 9)
	GrantAllowance(s, DefaultRules, "2026-09-13")
	s.Cash = 400
	s.Heat = 99
	for i := 0; i < 3; i++ {
		s.Evidence = append(s.Evidence, Evidence{ID: s.newID(), Kind: "car"})
	}
	toNight(t, s)
	Apply(s, Action{Kind: "end_phase"})
	if s.Alive {
		t.Fatalf("expected a raid at heat %d", s.Heat)
	}
	if r := Apply(s, Action{Kind: "hunt", Name: "solar"}); r.Err == "" {
		t.Fatal("the dead don't hunt")
	}
	must(t, Apply(s, Action{Kind: "inherit"}))
	if !s.Alive || s.Generation != 2 || len(s.Ancestors) != 1 || s.Cash != 200 || s.Day != 1 {
		t.Fatalf("inheritance wrong: alive=%v gen=%d anc=%d cash=%d day=%d", s.Alive, s.Generation, len(s.Ancestors), s.Cash, s.Day)
	}
	if !s.Has("stove") || !s.Has("cell_1") {
		t.Fatal("fixtures should persist across generations")
	}
}

func TestDeterministic(t *testing.T) {
	run := func() string {
		s := NewState("d", "b", 42)
		GrantAllowance(s, DefaultRules, "2026-09-13")
		var out []string
		for i := 0; i < 12; i++ {
			if s.Phase == Night && len(s.Prey) > 0 {
				r := Apply(s, Action{Kind: "hunt", Name: s.Prey[0]})
				for _, e := range r.Events {
					out = append(out, e.Text)
				}
			}
			r := Apply(s, Action{Kind: "end_phase"})
			for _, e := range r.Events {
				out = append(out, e.Text)
			}
		}
		return strings.Join(out, "\n")
	}
	if run() != run() {
		t.Fatal("same seed and actions produced different outcomes")
	}
}
