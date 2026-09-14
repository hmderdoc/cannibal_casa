package sim

import (
	"errors"
	"fmt"
	"math/rand"
)

// Item is anything that sits in a container or a worker's hands: a cut of
// meat, a finished dish, a bag of evidence, a body in transit.
type Item struct {
	ID       int    `json:"id"`
	Kind     string `json:"kind"` // cut | dish | evidence | livestock
	Sub      string `json:"sub"`  // cut kind, recipe id, evidence kind
	Victim   string `json:"victim,omitempty"`
	Marbling int    `json:"marbling,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	Born     int64  `json:"born"`
	Frozen   bool   `json:"frozen,omitempty"`
	Ref      int    `json:"ref,omitempty"` // livestock id when Kind == livestock
}

// Livestock is a person in a cell.
type Livestock struct {
	ID       int     `json:"id"`
	Victim   string  `json:"victim"`
	Name     string  `json:"name"`
	Cell     string  `json:"cell"`   // fixture id, "" while in transit or in the chair
	Hunger   float64 `json:"hunger"` // 0 fed .. 100 starving
	Marbling float64 `json:"marbling"`
	Since    int64   `json:"since"`
	Starve   int     `json:"starve"`          // minutes at 100 hunger
	Order    string  `json:"order,omitempty"` // a direct order: chair | butcher
}

// Step is one leg of a Job: walk to Dest, work for Work minutes, then Do.
type Step struct {
	Dest Pos    `json:"dest"`
	Work int    `json:"work"`
	Do   string `json:"do"`
}

// Job is what a worker is doing. It is data so a save mid-job resumes cleanly.
type Job struct {
	Kind    string `json:"kind"`
	Steps   []Step `json:"steps"`
	Step    int    `json:"step"`
	Timer   int    `json:"timer"`
	Machine string `json:"machine,omitempty"`
	Target  int    `json:"target,omitempty"`
	Recipe  string `json:"recipe,omitempty"`
	Label   string `json:"label"`
}

// Worker is the cannibal or a Renfield: an entity that walks the house doing
// jobs, and eats and sleeps like anyone.
type Worker struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"` // cannibal | renfield
	Victim   string  `json:"victim,omitempty"`
	Pos      Pos     `json:"pos"`
	Path     []Pos   `json:"path,omitempty"`
	Hunger   float64 `json:"hunger"` // 100 full .. 0 starving
	Energy   float64 `json:"energy"`
	Loyalty  float64 `json:"loyalty,omitempty"`
	Asleep   bool    `json:"asleep"`
	Starving int     `json:"starving"`
	Carry    []Item  `json:"carry,omitempty"`
	Job      *Job    `json:"job,omitempty"`
	Away     *Errand `json:"away,omitempty"`
	Role     string  `json:"role,omitempty"`   // Renfields: hand | cook | guard | shill | gimp
	Meals    int     `json:"meals,omitempty"`  // human dishes eaten; enough of them and a Renfield converts
	Active   int64   `json:"active,omitempty"` // clock of the last real job (not idling or sleeping)
}

// Busy reports whether a worker is doing something worth watching.
func (wk *Worker) Busy() bool {
	if wk.Asleep || wk.Job == nil {
		return false
	}
	return wk.Job.Kind != "idle" && wk.Job.Kind != "sleep"
}

// ConvertMeals is how many dishes it takes before a Renfield asks for
// seconds, knows, and is in: a fellow cannibal ("convert") who never wavers,
// eats like one of the family, and goes hunting on their own at night.
const ConvertMeals = 6

// Renfield roles: forms of forced servitude, each with a job.
const (
	RoleHand  = "hand"  // does whatever needs doing
	RoleCook  = "cook"  // kitchen first
	RoleGuard = "guard" // patrols the cells: escapes rare, raiders repelled
	RoleShill = "shill" // works the porch: the lures pull harder
	RoleGimp  = "gimp"  // lives in the basement: livestock stays quiet and calm
)

var Roles = []struct{ ID, Name, Desc string }{
	{RoleHand, "Hand", "General help: feeds, butchers, cooks, carries, disposes. The default."},
	{RoleCook, "Cook", "Kitchen first. Plates and cooks before anything else; a stocked shelf, always."},
	{RoleGuard, "Guard", "Patrols the cells and the doors. Escapes drop to a third; raiders on your house face them."},
	{RoleShill, "Shill", "Stands on the porch being friendly. Every lure pulls harder. Also minds the stand."},
	{RoleGimp, "Gimp", "Lives in the basement, in the suit. The livestock stops screaming and stops trying the door."},
}

func RoleName(id string) string {
	for _, r := range Roles {
		if r.ID == id {
			return r.Name
		}
	}
	return "Hand"
}

// Errand is the cannibal out of the house: working, selling, hunting,
// scavenging, or raiding another house. The outcome is settled when it
// starts (the world layer settles raids on players); it lands on return.
type Errand struct {
	Kind      string     `json:"kind"`
	Label     string     `json:"label"`
	Until     int64      `json:"until"`
	Cash      int        `json:"cash"`
	Items     []Item     `json:"items,omitempty"`
	Livestock *Livestock `json:"livestock,omitempty"`
	Heat      float64    `json:"heat"`
	Text      string     `json:"text"`
	TextKind  string     `json:"text_kind"`
	Post      string     `json:"post,omitempty"`
}

// Passerby walks the sidewalk; a lure may pull them onto the porch.
type Passerby struct {
	ID     int    `json:"id"`
	Victim string `json:"victim"`
	Pos    Pos    `json:"pos"`
	Dir    int    `json:"dir"`
	State  string `json:"state"` // walk | lured | flee
	Path   []Pos  `json:"path,omitempty"`
	Pair   bool   `json:"pair,omitempty"`
}

// Visitor is someone the street sends: an inspector, the dog, the police.
type Visitor struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	Tier  int    `json:"tier,omitempty"`
	Name  string `json:"name"`
	Pos   Pos    `json:"pos"`
	Path  []Pos  `json:"path,omitempty"`
	State string `json:"state"`
	Timer int    `json:"timer"`
}

// Machine is the runtime side of a fixture that processes something.
type Machine struct {
	Busy     int    `json:"busy"`               // minutes remaining
	Recipe   string `json:"recipe,omitempty"`   // station: what's cooking
	Item     string `json:"item,omitempty"`     // disposal: evidence kind inside
	Output   *Item  `json:"output,omitempty"`   // station: finished dish awaiting the shelf
	Load     int    `json:"load,omitempty"`     // chair: livestock inside
	Reserved bool   `json:"reserved,omitempty"` // a worker is on the way
	Off      bool   `json:"off,omitempty"`      // switched off by the player
	Uses     int    `json:"uses"`
}

// Policy is how the player tunes the machine without touching it.
type Policy struct {
	ButcherMarbling int             `json:"butcher_marbling"`  // butcher when marbling reaches this
	ButcherWhenFull bool            `json:"butcher_when_full"` // ...or when every cell is taken
	KeepLivestock   int             `json:"keep_livestock"`    // never butcher below this many
	PantryTarget    int             `json:"pantry_target"`     // keep this many food dishes on the shelf
	Sell            map[string]bool `json:"sell"`              // recipes cooked for the stand once the shelf is stocked
	Brainwash       bool            `json:"brainwash"`
	FeedReserve     int             `json:"feed_reserve"` // don't spend feed money below this cash
	Freeze          bool            `json:"freeze"`       // overflow cuts into the freezer
}

func DefaultPolicy() Policy {
	return Policy{ButcherMarbling: 4, ButcherWhenFull: true, PantryTarget: 3,
		Sell:        map[string]bool{"meat_pie": true, "jerky": true, "broth": true, "soap": true, "fritters": true},
		FeedReserve: 20, Freeze: true}
}

// Line is a ticker entry.
type Line struct {
	Tick int64  `json:"t"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Event is a feed post waiting for the world layer.
type Event struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type Missing struct {
	Victim string `json:"victim"`
	Due    int64  `json:"due"`
}

type Ancestor struct {
	Name     string `json:"name"`
	Days     int    `json:"days"`
	Meals    int    `json:"meals"`
	Stars    int    `json:"stars"`
	Cause    string `json:"cause"`
	Captured int    `json:"captured"`
}

type Stats struct {
	Captured    int `json:"captured"`
	Escaped     int `json:"escaped"`
	Butchered   int `json:"butchered"`
	Meals       int `json:"meals"`
	Sold        int `json:"sold"`
	Errands     int `json:"errands"`
	Raids       int `json:"raids"`
	Looted      int `json:"looted"` // cash taken from wallets and houses
	Buried      int `json:"buried"`
	Disposed    int `json:"disposed"`
	Inspections int `json:"inspections"`
	Dinners     int `json:"dinners"`
	Fled        int `json:"fled"` // failed traps
}

// World is one player's house and everything in it.
type World struct {
	Version    int    `json:"version"`
	Handle     string `json:"handle"`
	BBS        string `json:"bbs"`
	Cannibal   string `json:"cannibal"`
	Generation int    `json:"generation"`

	Clock    int64 `json:"clock"`     // game minutes since the dynasty began
	Born     int64 `json:"born"`      // clock when this cannibal took over
	LastWall int64 `json:"last_wall"` // unix seconds at last save, for catch-up
	Scale    int   `json:"scale"`     // game minutes per real minute

	Cash  float64 `json:"cash"`
	Heat  float64 `json:"heat"`
	Stars int     `json:"stars"`

	Fixtures map[string]bool     `json:"fixtures"`
	Placed   map[string]Pos      `json:"placed"` // where each owned fixture sits
	Machines map[string]*Machine `json:"machines"`
	Policy   Policy              `json:"policy"`

	Workers   []*Worker    `json:"workers"`
	Livestock []*Livestock `json:"livestock"`
	Passers   []*Passerby  `json:"passers"`
	Visitors  []*Visitor   `json:"visitors"`

	Fridge   []Item `json:"fridge"`
	Pantry   []Item `json:"pantry"`
	Pile     []Item `json:"pile"`
	Driveway []Item `json:"driveway"`
	Stand    []Item `json:"stand"`
	Buried   int    `json:"buried"`
	Compost  int    `json:"compost"`

	Missing      []Missing `json:"missing"`
	LastInspect  int64     `json:"last_inspect"`
	InspectTier  int       `json:"inspect_tier"` // highest tier that has already sent someone
	RaidCalled   bool      `json:"raid_called"`
	NextPasserby int64     `json:"next_passerby"`
	TrapReady    int64     `json:"trap_ready"`

	LastRaid map[string]int64 `json:"last_raid,omitempty"` // target key -> unix day of the last raid

	// Running totals since the last morning report, so small drains get a line.
	SpentFeed float64 `json:"spent_feed"`
	SpentUtil float64 `json:"spent_util"`
	FedCount  int     `json:"fed_count"`

	Alive     bool       `json:"alive"`
	Cause     string     `json:"cause,omitempty"`
	Ancestors []Ancestor `json:"ancestors"`
	Stats     Stats      `json:"stats"`
	Ticker    []Line     `json:"ticker"`
	Posts     []Event    `json:"posts,omitempty"`

	NextID int   `json:"next_id"`
	Seed   int64 `json:"seed"`
	Step   int64 `json:"step"`

	ColorMode    string `json:"color_mode,omitempty"`
	JournalOrder string `json:"journal_order,omitempty"` // "" or "newest" = newest first; "oldest" = reading order
}

const DefaultScale = 60

// New starts a dynasty: a starter kitchen, one cell, a block, a bed, a mat.
func New(handle, bbs string, seed int64) *World {
	w := &World{Version: 2, Handle: handle, BBS: bbs, Generation: 1, Scale: DefaultScale,
		Cash: 150, Heat: 8, Fixtures: map[string]bool{}, Placed: map[string]Pos{}, Machines: map[string]*Machine{},
		Policy: DefaultPolicy(), Alive: true, Seed: seed}
	w.Clock = 8 * 60 // move in at 8 A.M.
	w.Born = w.Clock
	w.Cannibal = randomCannibalName(w.rng())
	for _, f := range Fixtures {
		if f.Starter {
			w.Fixtures[f.ID] = true
			w.Placed[f.ID] = f.At
		}
	}
	w.Workers = append(w.Workers, &Worker{ID: w.newID(), Name: w.Cannibal, Kind: "cannibal",
		Pos: w.Pos("bed"), Hunger: 70, Energy: 90})
	w.NextPasserby = w.Clock + 5
	// Gladys's welcome casseroles: two days of grace before the machine has to work.
	for i := 0; i < 2; i++ {
		w.Pantry = append(w.Pantry, Item{ID: w.newID(), Kind: "dish", Sub: "fritters", Quality: 3, Born: w.Clock})
	}
	w.say("info", "Welcome home, %s. The fridge is empty and the neighbors are plump.", w.Cannibal)
	return w
}

func (w *World) rng() *rand.Rand {
	w.Step++
	return rand.New(rand.NewSource(w.Seed*1000003 + w.Step))
}

func (w *World) newID() int {
	w.NextID++
	return w.NextID
}

func (w *World) say(kind, format string, a ...any) {
	w.Ticker = append(w.Ticker, Line{Tick: w.Clock, Kind: kind, Text: fmt.Sprintf(format, a...)})
	if len(w.Ticker) > 400 {
		w.Ticker = w.Ticker[len(w.Ticker)-400:]
	}
}

func (w *World) post(kind, format string, a ...any) {
	w.Posts = append(w.Posts, Event{Kind: kind, Text: fmt.Sprintf(format, a...)})
}

func (w *World) Has(id string) bool { return w.Fixtures[id] }

// Pos is where an owned fixture sits (its default when unplaced).
func (w *World) Pos(id string) Pos {
	if p, ok := w.Placed[id]; ok {
		return p
	}
	if f := FixtureByID(id); f != nil {
		return f.At
	}
	return Pos{}
}

// FixtureAt returns the owned fixture on a tile, or nil.
func (w *World) FixtureAt(p Pos) *Fixture {
	for id, at := range w.Placed {
		if at == p && w.Fixtures[id] {
			return FixtureByID(id)
		}
	}
	return nil
}

// CanPlace reports whether a fixture may go on a tile: plain floor on the
// fixture's level (the porch for lures, trap and stand), not already taken.
func (w *World) CanPlace(f *Fixture, p Pos) error {
	if !f.AllowedOn(p.L) {
		return fmt.Errorf("the %s goes on the %s", f.Name, f.Levels())
	}
	if !Plans[p.L].Floor(p.X, p.Y) {
		return errors.New("that's not floor")
	}
	if f.Front && !inPorch(p) {
		return fmt.Errorf("the %s has to be out front, on the porch", f.Name)
	}
	if o := w.FixtureAt(p); o != nil && o.ID != f.ID {
		return fmt.Errorf("the %s is already there", o.Name)
	}
	return nil
}

// TrapPos is where the mat (or the trapdoor that replaced it) sits.
func (w *World) TrapPos() Pos {
	for _, f := range w.ownedOfKind(KindTrap) {
		return w.Pos(f.ID)
	}
	return FrontDoor
}

// Day, Hour, Minute of the game clock (day 1 is move-in day).
func (w *World) Day() int    { return int(w.Clock/day) + 1 }
func (w *World) Hour() int   { return int(w.Clock%day) / 60 }
func (w *World) Minute() int { return int(w.Clock % 60) }
func (w *World) DaysAlive() int {
	return int((w.Clock-w.Born)/day) + 1
}

// TimeString is "Day 3 14:05".
func (w *World) TimeString() string {
	return fmt.Sprintf("Day %d %02d:%02d", w.Day(), w.Hour(), w.Minute())
}

func (w *World) IsNight() bool { h := w.Hour(); return h >= 21 || h < 6 }

func (w *World) machine(id string) *Machine {
	m := w.Machines[id]
	if m == nil {
		m = &Machine{}
		w.Machines[id] = m
	}
	return m
}

// Owned fixtures of a kind.
func (w *World) ownedOfKind(kind string) []*Fixture {
	var out []*Fixture
	for i := range Fixtures {
		f := &Fixtures[i]
		if f.Kind == kind && w.Fixtures[f.ID] {
			out = append(out, f)
		}
	}
	return out
}

func (w *World) capacity(kind string, ids ...string) int {
	n := 0
	for _, f := range Fixtures {
		if f.Kind != kind || !w.Fixtures[f.ID] {
			continue
		}
		if len(ids) > 0 && !contains(ids, f.ID) {
			continue
		}
		n += f.Capacity
	}
	return n
}

func (w *World) FridgeCap() int  { return w.capacity(KindContainer, "fridge", "fridge_big") }
func (w *World) FreezerCap() int { return w.capacity(KindContainer, "freezer", "chest_freezer") }
func (w *World) PantryCap() int  { return w.capacity(KindContainer, "pantry", "pantry_big") }
func (w *World) PlotCap() int    { return w.capacity(KindPlot) }
func (w *World) StandCap() int   { return w.capacity(KindStand) }
func (w *World) Cells() []*Fixture {
	return w.ownedOfKind(KindCell)
}
func (w *World) CotCount() int {
	n := 0
	for _, f := range w.ownedOfKind(KindBed) {
		if f.ID != "bed" {
			n++
		}
	}
	return n
}

func (w *World) fridgeUsed() (fresh, frozen int) {
	for _, c := range w.Fridge {
		if c.Frozen {
			frozen++
		} else {
			fresh++
		}
	}
	return
}

func (w *World) freeCell() string {
	for _, f := range w.Cells() {
		taken := false
		for _, l := range w.Livestock {
			if l.Cell == f.ID {
				taken = true
				break
			}
		}
		if !taken {
			return f.ID
		}
	}
	return ""
}

func (w *World) livestockByID(id int) *Livestock {
	for _, l := range w.Livestock {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func (w *World) Renfields() int {
	n := 0
	for _, wk := range w.Workers {
		if wk.Kind == "renfield" {
			n++
		}
	}
	return n
}

// RoleCount is how many Renfields hold a role and are home and awake.
func (w *World) RoleCount(role string) int {
	n := 0
	for _, wk := range w.Workers {
		if wk.Kind != "cannibal" && wk.Role == role && wk.Away == nil {
			n++
		}
	}
	return n
}

func (w *World) Player() *Worker {
	for _, wk := range w.Workers {
		if wk.Kind == "cannibal" {
			return wk
		}
	}
	return nil
}

func (w *World) foodOnShelf() int {
	n := 0
	for _, d := range w.Pantry {
		if RecipeByID(d.Sub).Food {
			n++
		}
	}
	return n
}

func (w *World) Tier() int { return HeatTier(w.Heat) }

func HeatTier(heat float64) int {
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

// VisibleEvidence is what a visitor at the front door notices: strange cars
// at the curb, bags on the back step when the yard isn't fenced, screaming
// from the basement when it isn't soundproofed. Curtains hide a little.
func (w *World) VisibleEvidence() int {
	v := 3 * len(w.Driveway)
	if !w.Has("fence") {
		v += len(w.Pile)
	}
	if !w.Has("soundproof") {
		for _, l := range w.Livestock {
			if l.Hunger > 50 {
				v++
			}
		}
	}
	if w.Has("curtains") {
		v -= 2
	}
	if v < 0 {
		v = 0
	}
	return v
}

func InspectionThreshold(tier int) int {
	return []int{6, 4, 4, 3, 3, 2}[clamp(tier, 0, 5)]
}

// Summary is the street's view of a player.
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

func (w *World) Summary() Summary {
	return Summary{Handle: w.Handle, BBS: w.BBS, Cannibal: w.Cannibal, Generation: w.Generation,
		Day: w.DaysAlive(), Stars: w.Stars, Meals: w.Stats.Meals, Heat: int(w.Heat), Alive: w.Alive,
		Renfields: w.Renfields(), Livestock: len(w.Livestock)}
}

func (w *World) DynastyStars() int {
	n := w.Stars
	for _, a := range w.Ancestors {
		n += a.Stars
	}
	return n
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

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func randomCannibalName(r *rand.Rand) string {
	return relations[r.Intn(len(relations))] + " " + cannibalNames[r.Intn(len(cannibalNames))]
}
