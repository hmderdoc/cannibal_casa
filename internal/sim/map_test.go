package sim

import "testing"

func TestPlansConsistent(t *testing.T) {
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
					if len(FixturesAtSlot(l, s)) == 0 {
						t.Errorf("%s slot %q at %d,%d has no fixture", l, s, x, y)
					}
				}
			}
		}
		for _, f := range Fixtures {
			if f.Level == l && seen[f.Slot] != 1 {
				t.Errorf("fixture %s slot %q appears %d times on %s", f.ID, f.Slot, seen[f.Slot], l)
			}
		}
	}
	for a, b := range Links {
		if !walkable(a) || !walkable(b) || Links[b] != a {
			t.Errorf("link %v <-> %v is not walkable both ways", a, b)
		}
	}
	for _, r := range Recipes {
		if FixtureByID(r.Station) == nil || FixtureByID(r.Station).Kind != KindStation {
			t.Errorf("recipe %s station %q is not a station", r.ID, r.Station)
		}
		for _, n := range r.Needs {
			if FixtureByID(n) == nil {
				t.Errorf("recipe %s needs unknown %s", r.ID, n)
			}
		}
	}
	for _, f := range Fixtures {
		if f.Prereq != "" && FixtureByID(f.Prereq) == nil {
			t.Errorf("fixture %s prereq %s missing", f.ID, f.Prereq)
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

// Every fixture tile is reachable from every other: the sidewalk to the cells,
// the cells to the block, the block to the fridge, the fridge to the back step.
func TestEverythingReachable(t *testing.T) {
	var tiles []Pos
	for i := range Fixtures {
		tiles = append(tiles, SlotPos(&Fixtures[i]))
	}
	tiles = append(tiles, Pos{Ground, 0, SidewalkY}, GateTile, FrontDoor)
	for _, a := range tiles {
		for _, b := range tiles {
			if a != b && Path(a, b) == nil {
				t.Errorf("no path %v -> %v", a, b)
			}
		}
	}
	if len(Path(Pos{Basement, 2, 1}, SlotPos(FixtureByID("fridge")))) > 80 {
		t.Error("cell to fridge path is absurdly long")
	}
}
