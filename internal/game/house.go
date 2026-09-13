package game

// Floor plans. Each level is a fixed template: '#' wall, '.' floor, '+' door,
// '@' where the player stands on arrival, and any other character a fixture
// slot (see Fixture.Slot). The door draws walls as box-drawing, floors as a
// tinted field, owned fixtures as their glyph and empty slots as a faint
// outline so the player can see where the money goes.
type Plan struct {
	Level Level
	Rows  []string
	Rooms []Room
}

// Room is a label the renderer paints over empty floor.
type Room struct {
	Name string
	X, Y int
}

var Plans = [NumLevels]Plan{
	Ground: {Level: Ground, Rows: []string{
		"##############################################",
		"#.1.2.3.4.5.6....#.W....X....Y....Z....a.....#",
		"#................#...........................#",
		"########+###########+#########################",
		"#.A..B..C..D..E..#..O.....P.....Q.....N......#",
		"#................+...........................#",
		"#.F..G..H..I..J..#...........................#",
		"#.K..L...........#...........................#",
		"########+################+############+#######",
		"#...............#....R....U....#.............#",
		"#...............#...............#.....@......#",
		"##############################################",
	}, Rooms: []Room{
		{"PORCH", 2, 2}, {"GARAGE", 21, 2}, {"KITCHEN", 5, 5}, {"LIVING ROOM", 25, 6},
		{"CLOSET", 19, 10}, {"HALL", 34, 9}, {"PANTRY", 2, 10},
	}},
	Basement: {Level: Basement, Rows: []string{
		"##############################################",
		"#..A....B....C....D....E....#....F..G..H.....#",
		"#...........................+................#",
		"#...........................#....@...........#",
		"##############################################",
	}, Rooms: []Room{{"CELLS", 2, 3}, {"UTILITY", 36, 3}}},
	Backyard: {Level: Backyard, Rows: []string{
		"##############################################",
		"#.A....B....C........D....E.......F..........#",
		"#............................................#",
		"#......G....H..........I..........J..........#",
		"#....................@.......................#",
		"##############################################",
	}, Rooms: []Room{{"BACKYARD", 2, 2}}},
	Attic: {Level: Attic, Rows: []string{
		"##############################################",
		"#.....A.......B.......C.......D.......E......#",
		"#............................................#",
		"#.....................@......................#",
		"##############################################",
	}, Rooms: []Room{{"ATTIC", 2, 2}}},
}

// Spawn is where the player appears on a level.
func (p Plan) Spawn() (x, y int) {
	for yy, row := range p.Rows {
		for xx, ch := range row {
			if ch == '@' {
				return xx, yy
			}
		}
	}
	return 1, 1
}

// Walkable reports whether a tile can be stood on (floor, door, spawn, or a
// slot: the player bumps fixtures by walking onto them).
func (p Plan) Walkable(x, y int) bool {
	if y < 0 || y >= len(p.Rows) || x < 0 || x >= len(p.Rows[y]) {
		return false
	}
	return p.Rows[y][x] != '#'
}

// SlotAt returns the slot letter at a tile, or 0.
func (p Plan) SlotAt(x, y int) byte {
	if y < 0 || y >= len(p.Rows) || x < 0 || x >= len(p.Rows[y]) {
		return 0
	}
	c := p.Rows[y][x]
	switch c {
	case '#', '.', '+', '@':
		return 0
	}
	return c
}

// FixtureAtSlot finds the fixture that lives in a slot on a level (owned or not).
func FixtureAtSlot(level Level, slot byte) *Fixture {
	for i := range Fixtures {
		f := &Fixtures[i]
		if f.Level == level && len(f.Slot) == 1 && f.Slot[0] == slot {
			return f
		}
	}
	return nil
}
