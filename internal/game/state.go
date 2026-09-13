// Package game holds the pure rules of Cannibal Casa: the per-player State,
// the data tables (victims, fixtures, recipes), the Action -> Apply reducer and
// the day tick. It has no terminal or network code, so the server, the local
// door backend and the balance simulator all run exactly the same rules.
package game

import (
	"fmt"
	"math/rand"
)

// Phase is the quarter of the in-game day. Day phases are for cooking,
// building, selling and disposal; Night is for hunting.
type Phase int

const (
	Morning Phase = iota
	Afternoon
	Evening
	Night
)

func (p Phase) String() string {
	switch p {
	case Morning:
		return "Morning"
	case Afternoon:
		return "Afternoon"
	case Evening:
		return "Evening"
	}
	return "Night"
}

// Level is one floor of the house. Each has a fixed floor plan (house.go).
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

// Cut kinds a butchered victim yields. Recipes ask for kinds.
const (
	CutPrime  = "prime"
	CutOffal  = "offal"
	CutScraps = "scraps"
)

// Cut is one piece of meat in the fridge/freezer.
type Cut struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind"`
	Victim    string `json:"victim"`    // archetype id
	Marbling  int    `json:"marbling"`  // 1..5, meal quality driver
	Freshness int    `json:"freshness"` // days left before it spoils (ignored while frozen)
	Frozen    bool   `json:"frozen"`
}

// Dish is a cooked meal (or a sellable good like soap) in the pantry.
type Dish struct {
	ID      int    `json:"id"`
	Recipe  string `json:"recipe"`
	Quality int    `json:"quality"` // 1..5 stars
	Shelf   int    `json:"shelf"`   // days left
}

// Captive is livestock in the basement, or a brainwashed Renfield who roams
// free and minds the house while you're away.
type Captive struct {
	ID       int    `json:"id"`
	Victim   string `json:"victim"`
	Name     string `json:"name"`
	Days     int    `json:"days"`
	Marbling int    `json:"marbling"`
	Hungry   int    `json:"hungry"`   // consecutive days unfed
	Renfield bool   `json:"renfield"` // brainwashed helper
	Loyalty  int    `json:"loyalty"`  // Renfields only, 0..100
	Sessions int    `json:"sessions"` // brainwashing sessions so far
	LastTry  int    `json:"last_try"` // day of the last brainwashing attempt
}

// Evidence is what a meal leaves behind. Each piece radiates Heat daily until
// disposed of; a buried piece is quiet unless the neighbor's dog finds it.
type Evidence struct {
	ID     int    `json:"id"`
	Kind   string `json:"kind"` // bones | clothes | phone | car
	Victim string `json:"victim"`
	Buried bool   `json:"buried"`
	Age    int    `json:"age"`
}

// Missing tracks a captured victim until someone notices they're gone.
type Missing struct {
	Victim string `json:"victim"`
	Days   int    `json:"days"` // countdown; at 0 the heat lands
}

// Mission is an away-from-home job in progress.
type Mission struct {
	ID     string `json:"id"`
	Phases int    `json:"phases"` // remaining
}

// Inspection is a scheduled visit by whoever the current Heat tier sends.
type Inspection struct {
	Phases int    `json:"phases"` // until they knock
	Tier   int    `json:"tier"`
	Who    string `json:"who"`
}

// Ancestor is a finished cannibal in the dynasty's hall of fame.
type Ancestor struct {
	Name     string `json:"name"`
	Days     int    `json:"days"`
	Meals    int    `json:"meals"`
	Stars    int    `json:"stars"`
	Cause    string `json:"cause"`
	Captured int    `json:"captured"`
}

// Stats are lifetime counters for the current cannibal.
type Stats struct {
	Captured int `json:"captured"`
	Escaped  int `json:"escaped"`
	Dinners  int `json:"dinners"`
	Sold     int `json:"sold"`
	Buried   int `json:"buried"`
	Hunts    int `json:"hunts"`
	Missions int `json:"missions"`
}

// State is everything about one player. It is the unit of persistence and the
// value the server hands the door after every action.
type State struct {
	Version    int    `json:"version"`
	Handle     string `json:"handle"`
	BBS        string `json:"bbs"`
	Cannibal   string `json:"cannibal"`
	Generation int    `json:"generation"`

	Day   int   `json:"day"`
	Phase Phase `json:"phase"`

	Hunger int `json:"hunger"`
	Heat   int `json:"heat"`
	Cash   int `json:"cash"`
	Stars  int `json:"stars"`
	Meals  int `json:"meals"`

	Fixtures map[string]bool `json:"fixtures"`
	Recipes  map[string]bool `json:"recipes"`

	Fridge   []Cut      `json:"fridge"`
	Dishes   []Dish     `json:"dishes"`
	Captives []Captive  `json:"captives"`
	Evidence []Evidence `json:"evidence"`
	Missing  []Missing  `json:"missing"`

	Mission    *Mission    `json:"mission,omitempty"`
	Inspection *Inspection `json:"inspection,omitempty"`
	Prey       []string    `json:"prey"` // tonight's hunt options (archetype ids)

	Bank      int    `json:"bank"`               // in-game days available to advance
	LastGrant string `json:"last_grant"`         // YYYY-MM-DD of the last allowance grant
	LastTip   string `json:"last_tip,omitempty"` // YYYY-MM-DD of the last anonymous tip sent

	Alive     bool       `json:"alive"`
	Cause     string     `json:"cause,omitempty"`
	Ancestors []Ancestor `json:"ancestors"`
	Stats     Stats      `json:"stats"`
	Journal   []string   `json:"journal"`

	Compost int   `json:"compost"` // bones composted, feeds the tomato garden
	NextID  int   `json:"next_id"`
	Seed    int64 `json:"seed"`
	Step    int64 `json:"step"`

	// Player prefs carried with the save so they follow the caller across nodes.
	ColorMode string `json:"color_mode,omitempty"`
}

// Rules are the tunable pacing numbers a sysop sets in the ini.
type Rules struct {
	DaysPerDay int // in-game days granted per real day
	BankMax    int // ceiling on banked days
}

// DefaultRules matches cannibalcasa.example.ini.
var DefaultRules = Rules{DaysPerDay: 3, BankMax: 9}

// NewState builds a fresh dynasty: a starter kitchen, one cell, one plot, a
// little cash and a hungry belly.
func NewState(handle, bbs string, seed int64) *State {
	s := &State{
		Version:    1,
		Handle:     handle,
		BBS:        bbs,
		Generation: 1,
		Day:        1,
		Phase:      Morning,
		Hunger:     60,
		Heat:       10,
		Cash:       120,
		Fixtures:   map[string]bool{},
		Recipes:    map[string]bool{},
		Alive:      true,
		Seed:       seed,
	}
	s.Cannibal = randomCannibalName(rand.New(rand.NewSource(seed)))
	for _, f := range Fixtures {
		if f.Starter {
			s.Fixtures[f.ID] = true
		}
	}
	for _, r := range Recipes {
		if r.Starter {
			s.Recipes[r.ID] = true
		}
	}
	s.log("Welcome home, %s. The fridge is empty and the neighbors are plump.", s.Cannibal)
	return s
}

// rng returns a fresh deterministic generator for the next random decision.
// Each call advances Step, so a save replayed with the same seed and action
// sequence produces the same outcomes (which is what the balance sim relies on).
func (s *State) rng() *rand.Rand {
	s.Step++
	return rand.New(rand.NewSource(s.Seed*1000003 + s.Step))
}

func (s *State) newID() int {
	s.NextID++
	return s.NextID
}

func (s *State) log(format string, a ...any) {
	s.Journal = append(s.Journal, fmt.Sprintf(format, a...))
	if len(s.Journal) > 60 {
		s.Journal = s.Journal[len(s.Journal)-60:]
	}
}

func (s *State) Has(fixture string) bool { return s.Fixtures[fixture] }

// Tier is the antagonist ladder rung for the current Heat.
func (s *State) Tier() int { return HeatTier(s.Heat) }

// HeatTier maps Heat to its ladder rung: 0 quiet .. 5 FBI.
func HeatTier(heat int) int {
	switch {
	case heat >= 90:
		return 5
	case heat >= 75:
		return 4
	case heat >= 60:
		return 3
	case heat >= 40:
		return 2
	case heat >= 20:
		return 1
	}
	return 0
}

// TierName is who's watching at each rung.
func TierName(t int) string {
	switch t {
	case 1:
		return "Nosy Neighbor"
	case 2:
		return "The HOA"
	case 3:
		return "Local News"
	case 4:
		return "True Crime Podcast"
	case 5:
		return "The FBI"
	}
	return "Quiet Street"
}

// CellCount is basement livestock capacity (Renfields don't take a cell).
func (s *State) CellCount() int {
	n := 0
	for _, f := range Fixtures {
		if f.Cell && s.Fixtures[f.ID] {
			n++
		}
	}
	return n
}

// Livestock are the captives still in cells.
func (s *State) Livestock() int {
	n := 0
	for _, c := range s.Captives {
		if !c.Renfield {
			n++
		}
	}
	return n
}

// Renfields are brainwashed helpers.
func (s *State) Renfields() int {
	n := 0
	for _, c := range s.Captives {
		if c.Renfield {
			n++
		}
	}
	return n
}

// FridgeCap / FreezerCap are storage limits from owned fixtures.
func (s *State) FridgeCap() int {
	n := 0
	if s.Has("fridge") {
		n += 6
	}
	if s.Has("fridge_big") {
		n += 6
	}
	return n
}

func (s *State) FreezerCap() int {
	n := 0
	if s.Has("freezer") {
		n += 8
	}
	if s.Has("chest_freezer") {
		n += 16
	}
	return n
}

// PlotCap is how many pieces of evidence the backyard can hold underground.
func (s *State) PlotCap() int {
	n := 0
	for _, f := range Fixtures {
		if f.Plot > 0 && s.Fixtures[f.ID] {
			n += f.Plot
		}
	}
	return n
}

func (s *State) Buried() int {
	n := 0
	for _, e := range s.Evidence {
		if e.Buried {
			n++
		}
	}
	return n
}

func (s *State) fridgeUsed() (fresh, frozen int) {
	for _, c := range s.Fridge {
		if c.Frozen {
			frozen++
		} else {
			fresh++
		}
	}
	return
}

// Summary is the neighbor-list view of a player.
type Summary struct {
	Handle     string `json:"handle"`
	BBS        string `json:"bbs"`
	Cannibal   string `json:"cannibal"`
	Generation int    `json:"generation"`
	Day        int    `json:"day"`
	Stars      int    `json:"stars"`
	Meals      int    `json:"meals"`
	Heat       int    `json:"heat"`
	Alive      bool   `json:"alive"`
	Renfields  int    `json:"renfields"`
	Livestock  int    `json:"livestock"`
}

func (s *State) Summary() Summary {
	return Summary{
		Handle: s.Handle, BBS: s.BBS, Cannibal: s.Cannibal, Generation: s.Generation,
		Day: s.Day, Stars: s.Stars, Meals: s.Meals, Heat: s.Heat, Alive: s.Alive,
		Renfields: s.Renfields(), Livestock: s.Livestock(),
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
