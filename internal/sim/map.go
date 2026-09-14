package sim

// Level is one floor of the house.
type Level int

const (
	Ground Level = iota
	Basement
	Backyard
	Attic
	Mall
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
	case Attic:
		return "Attic"
	}
	return "The Mall"
}

// Pos is a tile on a level.
type Pos struct {
	L Level `json:"l"`
	X int   `json:"x"`
	Y int   `json:"y"`
}

// Plan is a floor: '#' wall, '.' floor, '+' door, '=' street, '-' sidewalk,
// '>' '<' stairs. Fixtures are placed by the player on any floor tile of the
// right level (see Fixture.Level / Front), so the plan itself is just rooms.
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
		"#................#...........................#",
		"#................#...........................#",
		"########+################+####################",
		"#................#...........................#",
		"#................+...........................#",
		"#................#...........................+",
		"#................#...........................#",
		"########+################+############+#######",
		"#...............#..............#.............#",
		"#...............#..............#......><.....#",
		"##############################################",
	}, Rooms: []Room{
		{"PORCH", 1, 4}, {"GARAGE", 18, 4}, {"KITCHEN", 1, 9}, {"LIVING ROOM", 18, 9},
		{"CLOSET", 17, 12}, {"HALL", 33, 11}, {"BEDROOM", 1, 12},
	}},
	Basement: {Level: Basement, Rows: []string{
		"##############################################",
		"#...........................#................#",
		"#...........................+......>.........#",
		"#...........................#................#",
		"##############################################",
	}, Rooms: []Room{{"CELLS", 1, 3}, {"UTILITY", 38, 1}}},
	Backyard: {Level: Backyard, Rows: []string{
		"##############################################",
		"#............................................#",
		"+............................................#",
		"#............................................#",
		"#............................................#",
		"##############################################",
	}, Rooms: []Room{{"BACKYARD", 34, 4}}},
	Attic: {Level: Attic, Rows: []string{
		"##############################################",
		"#............................................#",
		"#............................................#",
		"#.....................<......................#",
		"##############################################",
	}, Rooms: []Room{{"ATTIC", 1, 1}}},
	// The mall: stores off a concourse, customers walking it all day. A booth
	// goes on the concourse; the food court is where the mall job is.
	Mall: {Level: Mall, Rows: []string{
		"##############################################",
		"#.................#...............#..........#",
		"#.................#...............#..........#",
		"#.................+...............+..........#",
		"#............................................#",
		"#.................+...............+..........#",
		"#.................#...............#..........#",
		"##############+###############################",
	}, Rooms: []Room{
		{"FOOD COURT", 2, 1}, {"PRETZEL TIME", 20, 1}, {"THE GAP", 36, 1},
		{"SHOE PALACE", 2, 6}, {"MOVIE 12", 20, 6}, {"ARCADE", 36, 6},
	}},
}

// Links join tiles across levels: stairs and the back door. Both directions.
var Links = map[Pos]Pos{
	{Ground, 38, 12}:  {Basement, 35, 2},
	{Basement, 35, 2}: {Ground, 38, 12},
	{Ground, 39, 12}:  {Attic, 22, 3},
	{Attic, 22, 3}:    {Ground, 39, 12},
	{Ground, 45, 8}:   {Backyard, 0, 2},
	{Backyard, 0, 2}:  {Ground, 45, 8},
	{Ground, 0, 1}:    {Mall, 14, 7}, // the end of the sidewalk is a short walk to the mall
	{Mall, 14, 7}:     {Ground, 0, 1},
}

// Named tiles and areas the sim routes to.
var (
	GateTile     = Pos{Ground, 9, 2} // sidewalk gate
	FrontDoor    = Pos{Ground, 8, 4} // where a visitor knocks and an errand returns
	SidewalkY    = 1
	StreetY      = 0
	DrivewayX    = 11 // where a victim's car parks on the street row
	SidewalkEnds = [2]int{0, 45}
	// PorchRect is the front of the house: lures, the trap and the stand go here.
	PorchRect = struct{ X0, Y0, X1, Y1 int }{1, 3, 16, 4}
	// The mall's tiles the sim routes to.
	MallEntrance = Pos{Mall, 14, 7}
	FoodCourt    = Pos{Mall, 9, 2}
	ConcourseY   = 4
	MallEnds     = [2]int{1, 44}
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

// Floor reports whether a tile is plain interior floor (where things go).
func (p Plan) Floor(x, y int) bool { return p.At(x, y) == '.' }

func (p Plan) Spawn() Pos {
	for y, row := range p.Rows {
		for x := range row {
			if row[x] == '>' {
				return Pos{p.Level, x, y}
			}
		}
	}
	return Pos{p.Level, 1, 1}
}

func walkable(p Pos) bool { return Plans[p.L].Walkable(p.X, p.Y) }

func inPorch(p Pos) bool {
	return p.L == Ground && p.X >= PorchRect.X0 && p.X <= PorchRect.X1 && p.Y >= PorchRect.Y0 && p.Y <= PorchRect.Y1
}

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
