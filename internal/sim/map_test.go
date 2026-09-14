package sim

import "testing"

func TestPlansConsistent(t *testing.T) {
	for l := Level(0); l < NumLevels; l++ {
		p := Plans[l]
		w := len(p.Rows[0])
		for y, row := range p.Rows {
			if len(row) != w {
				t.Errorf("%s row %d is %d wide, want %d", l, y, len(row), w)
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
	w := New("t", "b", 1)
	for _, f := range Fixtures {
		if f.Prereq != "" && FixtureByID(f.Prereq) == nil {
			t.Errorf("fixture %s prereq %s missing", f.ID, f.Prereq)
		}
		if f.Starter {
			if err := w.CanPlace(&f, f.At); err != nil {
				t.Errorf("starter %s default tile %v: %v", f.ID, f.At, err)
			}
		}
	}
	for _, v := range Victims {
		for _, tr := range v.Traits {
			if Traits[tr] == "" {
				t.Errorf("victim %s has unknown trait %q", v.ID, tr)
			}
		}
	}
	for _, f := range Fixtures {
		if f.Kind == KindLure && len(f.Draws) == 0 {
			t.Errorf("lure %s draws nobody", f.ID)
		}
		for _, d := range f.Draws {
			if Traits[d] == "" {
				t.Errorf("lure %s draws unknown trait %q", f.ID, d)
			}
		}
	}
}

// Every fixture tile is reachable from every other: the sidewalk to the cells,
// the cells to the block, the block to the fridge, the fridge to the back step.
func TestEverythingReachable(t *testing.T) {
	w := New("t", "b", 1)
	var tiles []Pos
	for id := range w.Placed {
		tiles = append(tiles, w.Pos(id))
	}
	tiles = append(tiles, Pos{Ground, 0, SidewalkY}, GateTile, FrontDoor, Pos{Attic, 1, 1}, Pos{Backyard, 44, 3})
	for _, a := range tiles {
		for _, b := range tiles {
			if a != b && Path(a, b) == nil {
				t.Errorf("no path %v -> %v", a, b)
			}
		}
	}
	if len(Path(Pos{Basement, 2, 1}, w.Pos("fridge"))) > 80 {
		t.Error("cell to fridge path is absurdly long")
	}
}
