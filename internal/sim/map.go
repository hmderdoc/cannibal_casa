package sim

// Level is one floor of the house.
type Level int

const (
	Ground Level = iota
	Basement
	Backyard
	Attic
	NumLevels
)

func (l Level) String() string {
	switch l {
	case Ground:
		return "Ground Floor"
	case Basement:
		return "Basement"
	case Backyard:
		return "Backyard"
	}
	return "Attic"
}

// Pos is a tile on a level.
type Pos struct {
	L Level `json:"l"`
	X int   `json:"x"`
	Y int   `json:"y"`
}

// Plan is a floor: '#' wall, '.' floor, '+' door, '=' street, '-' sidewalk,
// '>' '<' stairs, '@' spawn, any other character a fixture slot. Everything
// but '#' and '=' is walkable; the street is for cars.
type Plan struct {
	Level Level
	Rows  []string
	Rooms []Room
}

type Room struct {
	Name string
	X, Y int
}

var Plans = [NumLevels]Plan{
	Ground: {Level: Ground, Rows: []string{
		"==============================================",
		"----------------------------------------------",
		"#########+####################################",
		"#.1.2.3.4.5.6..0.#.W....X....Y....Z....a.....#",
		"#.......7........#...........................#",
		"########+################+####################",
		"#.A..B..C..D..E..#..O.....P.....Q.....N......#",
		"#................+...........................#",
		"#.F..G..H..I..J..#...........................+",
		"#.K..L...........#...........................#",
		"########+################+############+#######",
		"#..z........y.x.#....R....U....#......>......#",
		"#...............#...............#......<.....#",
		"##############################################",
	}, Rooms: []Room{
		{"PORCH", 9, 4}, {"GARAGE", 21, 4}, {"KITCHEN", 5, 7}, {"LIVING ROOM", 25, 8},
		{"CLOSET", 19, 12}, {"HALL", 34, 11}, {"BEDROOM", 5, 12},
	}},
	Basement: {Level: Basement, Rows: []string{
		"##############################################",
		"#..A....B....C....D....E....#....F..G..H.....#",
		"#...........................+......>.........#",
		"#...........................#......K.........#",
		"##############################################",
	}, Rooms: []Room{{"CELLS", 2, 3}, {"UTILITY", 38, 3}}},
	Backyard: {Level: Backyard, Rows: []string{
		"##############################################",
		"#.A....B....C........D....E.......F..........#",
		"+............................................#",
		"#......G....H..........I..........J..........#",
		"#....................P.......................#",
		"##############################################",
	}, Rooms: []Room{{"BACKYARD", 30, 4}}},
	Attic: {Level: Attic, Rows: []string{
		"##############################################",
		"#.....A.......B.......C.......D.......E....F.#",
		"#............................................#",
		"#.....................<......................#",
		"##############################################",
	}, Rooms: []Room{{"ATTIC", 2, 2}}},
}

// Links join tiles across levels: stairs and the back door. Both directions.
var Links = map[Pos]Pos{
	{Ground, 39, 11}:  {Basement, 35, 2},
	{Basement, 35, 2}: {Ground, 39, 11},
	{Ground, 39, 12}:  {Attic, 22, 3},
	{Attic, 22, 3}:    {Ground, 39, 12},
	{Ground, 45, 8}:   {Backyard, 0, 2},
	{Backyard, 0, 2}:  {Ground, 45, 8},
}

// Named tiles the sim routes to.
var (
	GateTile     = Pos{Ground, 9, 2} // sidewalk gate
	FrontDoor    = Pos{Ground, 8, 4} // where a visitor knocks
	SidewalkY    = 1
	StreetY      = 0
	DrivewayX    = 11 // where a victim's car parks on the street row
	SidewalkEnds = [2]int{0, 45}
)

func (p Plan) At(x, y int) byte {
	if y < 0 || y >= len(p.Rows) || x < 0 || x >= len(p.Rows[y]) {
		return '#'
	}
	return p.Rows[y][x]
}

func (p Plan) Walkable(x, y int) bool {
	c := p.At(x, y)
	return c != '#' && c != '='
}

func (p Plan) SlotAt(x, y int) byte {
	switch c := p.At(x, y); c {
	case '#', '.', '+', '@', '=', '-', '>', '<':
		return 0
	default:
		return c
	}
}

func (p Plan) Spawn() Pos {
	for y, row := range p.Rows {
		for x := range row {
			if row[x] == '@' || row[x] == '>' {
				return Pos{p.Level, x, y}
			}
		}
	}
	return Pos{p.Level, 1, 1}
}

// SlotPos finds a fixture's tile.
func SlotPos(f *Fixture) Pos {
	p := Plans[f.Level]
	for y, row := range p.Rows {
		for x := range row {
			if row[x] == f.Slot {
				return Pos{f.Level, x, y}
			}
		}
	}
	return Pos{f.Level, 1, 1}
}

// FixturesAtSlot lists every fixture that can occupy a tile (an upgrade
// replaces its predecessor in the same slot, e.g. the trapdoors).
func FixturesAtSlot(level Level, slot byte) []*Fixture {
	var out []*Fixture
	for i := range Fixtures {
		f := &Fixtures[i]
		if f.Level == level && f.Slot == slot {
			out = append(out, f)
		}
	}
	return out
}

func walkable(p Pos) bool { return Plans[p.L].Walkable(p.X, p.Y) }

// Path finds a route between tiles with breadth-first search over the four
// neighbors plus the stair/door links. The house is tiny, so no caching.
func Path(from, to Pos) []Pos {
	if from == to {
		return nil
	}
	prev := map[Pos]Pos{from: from}
	queue := []Pos{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			var path []Pos
			for p := to; p != from; p = prev[p] {
				path = append(path, p)
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path
		}
		var next []Pos
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := Pos{cur.L, cur.X + d[0], cur.Y + d[1]}
			if walkable(n) {
				next = append(next, n)
			}
		}
		if l, ok := Links[cur]; ok {
			next = append(next, l)
		}
		for _, n := range next {
			if _, seen := prev[n]; !seen {
				prev[n] = cur
				queue = append(queue, n)
			}
		}
	}
	return nil
}
