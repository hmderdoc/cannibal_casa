package game

import "testing"

// Every plan is rectangular, every fixture's slot exists exactly once on its
// level, and every slot on a plan belongs to some fixture (no dead letters).
func TestPlansAreConsistent(t *testing.T) {
	for l := Level(0); l < NumLevels; l++ {
		p := Plans[l]
		w := len(p.Rows[0])
		seen := map[byte]int{}
		for y, row := range p.Rows {
			if len(row) != w {
				t.Errorf("%s row %d is %d wide, want %d", l, y, len(row), w)
			}
			for x := range row {
				if s := p.SlotAt(x, y); s != 0 {
					seen[s]++
					if FixtureAtSlot(l, s) == nil {
						t.Errorf("%s slot %q at %d,%d has no fixture", l, s, x, y)
					}
				}
			}
		}
		for _, f := range Fixtures {
			if f.Level != l {
				continue
			}
			if len(f.Slot) != 1 || seen[f.Slot[0]] != 1 {
				t.Errorf("fixture %s slot %q appears %d times on %s", f.ID, f.Slot, seen[f.Slot[0]], l)
			}
		}
		if x, y := p.Spawn(); !p.Walkable(x, y) {
			t.Errorf("%s spawn %d,%d not walkable", l, x, y)
		}
	}
}

func TestDataTablesResolve(t *testing.T) {
	for _, f := range Fixtures {
		if f.Prereq != "" && FixtureByID(f.Prereq) == nil {
			t.Errorf("fixture %s prereq %s missing", f.ID, f.Prereq)
		}
	}
	for _, r := range Recipes {
		for _, n := range r.Needs {
			if FixtureByID(n) == nil {
				t.Errorf("recipe %s needs unknown fixture %s", r.ID, n)
			}
		}
	}
	for _, v := range Victims {
		for _, l := range v.Lures {
			if FixtureByID(l) == nil {
				t.Errorf("victim %s lure %s missing", v.ID, l)
			}
		}
	}
}
