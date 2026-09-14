package sim

import (
	"errors"
	"fmt"
	"strings"
)

// Commands are the player's hand: build, switch machines, tune policy, rename,
// inherit. They are the only way the door changes the world; the clock does
// the rest.

// replaces maps an upgrade to the fixture it supersedes in place.
var replaces = map[string]string{"trapdoor_pro": "welcome_mat", "stand_big": "stand"}

// Build buys a fixture and places it. An upgrade that supersedes something
// (the trapdoor, the farm stand) takes its predecessor's tile.
func (w *World) Build(id string, at Pos) error {
	f := FixtureByID(id)
	if f == nil {
		return errors.New("no such upgrade")
	}
	if w.Has(id) {
		return fmt.Errorf("you already have the %s", f.Name)
	}
	if f.Prereq != "" && !w.Has(f.Prereq) {
		return fmt.Errorf("needs the %s first", FixtureByID(f.Prereq).Name)
	}
	if w.Cash < float64(f.Cost) {
		return fmt.Errorf("the %s costs $%d; you have $%.0f", f.Name, f.Cost, w.Cash)
	}
	if old := replaces[id]; old != "" && w.Has(old) {
		at = w.Pos(old)
		delete(w.Fixtures, old)
		delete(w.Placed, old)
	} else if err := w.CanPlace(f, at); err != nil {
		return err
	}
	w.Cash -= float64(f.Cost)
	w.Fixtures[id] = true
	w.Placed[id] = at
	w.say("good", "Installed: %s. %s", f.Name, f.Desc)
	if f.Kind == KindCell {
		w.post("info", "put in a new %s. \"Wine cellar,\" they said.", strings.ToLower(f.Name))
	}
	return nil
}

// Move relocates an owned fixture. Workers re-path on their next step, so a
// fridge dragged next to the block shortens every trip after this one.
func (w *World) Move(id string, at Pos) error {
	f := FixtureByID(id)
	if f == nil || !w.Has(id) {
		return errors.New("you don't have that")
	}
	if err := w.CanPlace(f, at); err != nil {
		return err
	}
	w.Placed[id] = at
	for _, wk := range w.Workers {
		wk.Path = nil
		if wk.Job != nil {
			for i := range wk.Job.Steps {
				if wk.Job.Steps[i].Dest == w.Pos(id) {
					wk.Job.Steps[i].Dest = at
				}
			}
		}
	}
	w.say("info", "Moved the %s.", f.Name)
	return nil
}

// ---------------------------------------------------------------------------
// errands: the cannibal leaves the house
// ---------------------------------------------------------------------------

type ErrandDef struct {
	ID    string
	Name  string
	Hours int
	Desc  string
}

var Errands = []ErrandDef{
	{"work", "Work a shift at the mall", 4, "Four hours of 'consulting' at the food court. Pays $40-60, no questions. The house runs itself."},
	{"market", "Farmers market run", 3, "Take everything on the stand and every dish marked for sale, and sell it all at once for full price."},
	{"hunt", "Go hunting", 4, "Stop waiting for the mat. Odds depend on your gear and who's out; best after dark. A cell must be free when you get back."},
	{"scrap", "Scavenge the neighborhood", 2, "Bins, curbs, an unlocked garage. Usually a little cash. Sometimes a cooler nobody will miss."},
	{"raid", "Raid a house", 5, "Break into another house on the street while its owner's away. Cash, meat, dishes. Cameras and fences fight back."},
}

func ErrandByID(id string) *ErrandDef {
	for i := range Errands {
		if Errands[i].ID == id {
			return &Errands[i]
		}
	}
	return nil
}

// StartErrand sends the cannibal out. Raids on players are settled by the
// world layer and handed in through StartRaid; "raid" here is a CPU house.
func (w *World) StartErrand(kind string) error {
	p := w.Player()
	if p == nil {
		return errors.New("nobody home")
	}
	if p.Away != nil {
		return fmt.Errorf("you're already out: %s", p.Away.Label)
	}
	def := ErrandByID(kind)
	if def == nil {
		return errors.New("no such errand")
	}
	r := w.rng()
	e := &Errand{Kind: kind, Label: strings.ToLower(def.Name), Until: w.Clock + int64(def.Hours*60), TextKind: "info"}
	switch kind {
	case "work":
		e.Cash = 40 + r.Intn(21)
		e.Text = fmt.Sprintf("Back from the mall with $%d. Someone asked about your background. You said 'meat'.", e.Cash)
		e.TextKind = "cash"
	case "market":
		total := 0
		n := 0
		var keep []Item
		for _, d := range w.Stand {
			total += price(d)
			n++
		}
		w.Stand = nil
		spare := w.foodOnShelf() - w.Policy.PantryTarget
		for _, d := range w.Pantry {
			rc := RecipeByID(d.Sub)
			if rc.Price > 0 && w.Policy.Sell[rc.ID] && (!rc.Food || spare > 0) {
				total += price(d)
				n++
				if rc.Food {
					spare--
				}
				continue
			}
			keep = append(keep, d)
		}
		w.Pantry = keep
		if n == 0 {
			return errors.New("nothing marked for sale; check Policy, or cook something first")
		}
		e.Cash = total
		w.Stats.Sold += n
		e.Text = fmt.Sprintf("Sold %d item%s at the market for $%d. 'Is it organic?' 'He was.'", n, plural(n), total)
		e.TextKind = "cash"
	case "hunt":
		v := w.rollVictim(r)
		chance := w.trapChance(v) + 10
		if !w.IsNight() {
			chance -= 15
		}
		if r.Intn(100) < clamp(chance, 10, 90) {
			name := pickName(v, r)
			e.Livestock = &Livestock{ID: w.newID(), Victim: v.ID, Name: name, Marbling: float64(v.Marbling), Since: e.Until}
			e.Cash = v.Loot
			e.Heat = 2
			e.Text = fmt.Sprintf("Came home with %s the %s over one shoulder. %s", name, v.Name, v.Flavor)
			e.TextKind = "good"
		} else {
			e.Heat = 3 + float64(v.Fight)
			e.Text = fmt.Sprintf("The %s got away and yelled about it. Heat +%.0f.", strings.ToLower(v.Name), e.Heat)
			e.TextKind = "bad"
		}
	case "scrap":
		switch r.Intn(4) {
		case 0:
			e.Items = []Item{{ID: w.newID(), Kind: "cut", Sub: CutPrime, Victim: "tourist", Marbling: 3, Born: e.Until},
				{ID: w.newID(), Kind: "cut", Sub: CutPrime, Victim: "tourist", Marbling: 3, Born: e.Until}}
			e.Text = "Found a cooler behind the Hendersons' garage. Two prime cuts. Somebody else has a hobby."
			e.TextKind = "food"
		case 1:
			e.Text = "Bins, curbs, nothing. A raccoon judged you."
		default:
			e.Cash = 10 + r.Intn(21)
			e.Text = fmt.Sprintf("Cans, a lawnmower, a jar of quarters: $%d.", e.Cash)
			e.TextKind = "cash"
		}
	case "raid":
		houses := []string{"the Hendersons", "the Nguyens", "the Carlucci twins", "that Airbnb on the corner", "the model home"}
		h := houses[r.Intn(len(houses))]
		if r.Intn(100) < 55 {
			e.Cash = 30 + r.Intn(61)
			e.Items = []Item{{ID: w.newID(), Kind: "dish", Sub: "fritters", Quality: 2 + r.Intn(2), Born: e.Until}}
			e.Text = fmt.Sprintf("Hit %s while they were at soccer. $%d and something from the freezer.", h, e.Cash)
			e.TextKind = "cash"
			e.Post = fmt.Sprintf("'s street: %s came home to a jimmied side door. 'Nothing missing but the casserole.'", h)
		} else {
			e.Heat = 8
			e.Text = fmt.Sprintf("%s have a doorbell camera. Home empty-handed. Heat +8.", capitalizeFirst(h))
			e.TextKind = "bad"
		}
	}
	return w.depart(p, e)
}

// StartRaid sends the cannibal to a player's house with an outcome the world
// layer already settled.
func (w *World) StartRaid(e *Errand) error {
	p := w.Player()
	if p == nil {
		return errors.New("nobody home")
	}
	if p.Away != nil {
		return fmt.Errorf("you're already out: %s", p.Away.Label)
	}
	e.Until = w.Clock + int64(ErrandByID("raid").Hours*60)
	return w.depart(p, e)
}

// autoHunt sends a convert out for the night on their own initiative.
func (w *World) autoHunt(wk *Worker) {
	r := w.rng()
	v := w.rollVictim(r)
	e := &Errand{Kind: "hunt", Label: "out hunting, on their own", Until: w.Clock + 4*60, TextKind: "info"}
	if r.Intn(100) < clamp(w.trapChance(v)+5, 10, 85) {
		name := pickName(v, r)
		e.Livestock = &Livestock{ID: w.newID(), Victim: v.ID, Name: name, Marbling: float64(v.Marbling), Since: e.Until}
		e.Cash = v.Loot
		e.Heat = 2
		e.Text = fmt.Sprintf("%s came back with %s the %s. Didn't even ask.", wk.Name, name, v.Name)
		e.TextKind = "good"
	} else {
		e.Heat = 2 + float64(v.Fight)
		e.Text = fmt.Sprintf("%s came back empty-handed and a little too excited. Heat +%.0f.", wk.Name, e.Heat)
		e.TextKind = "bad"
	}
	w.depart(wk, e)
}

func (w *World) depart(p *Worker, e *Errand) error {
	if p.Asleep {
		p.Asleep = false
	}
	w.dropBack(p)
	w.unreserve(jobMachine(p))
	p.Job = nil
	p.Path = nil
	p.Away = e
	if p.Kind == "cannibal" {
		w.Stats.Errands++
		if e.Kind == "raid" {
			w.Stats.Raids++
		}
	}
	w.say("info", "%s left: %s.", p.Name, e.Label)
	return nil
}

func (w *World) returnHome(p *Worker) {
	e := p.Away
	p.Away = nil
	p.Pos = FrontDoor
	w.Cash += float64(e.Cash)
	if e.Kind != "work" && e.Kind != "market" {
		w.Stats.Looted += e.Cash
	}
	w.Heat = clampF(w.Heat+e.Heat, 0, 120)
	for _, it := range e.Items {
		it.Born = w.Clock
		switch it.Kind {
		case "cut":
			if !w.storeCut(it) {
				w.say("bad", "No cold space for what you brought home. Wasted.")
			}
		case "dish":
			if len(w.Pantry) < w.PantryCap() {
				w.Pantry = append(w.Pantry, it)
			}
		}
	}
	if e.Livestock != nil {
		l := e.Livestock
		if cell := w.freeCell(); cell != "" {
			l.Cell = cell
			w.Livestock = append(w.Livestock, l)
			w.Missing = append(w.Missing, Missing{Victim: l.Victim, Due: w.Clock + int64(VictimByID(l.Victim).NoticeDays*day)})
			w.Stats.Captured++
		} else {
			w.say("warn", "No cell for %s. Straight to the block.", l.Name)
			p.Carry = append(p.Carry, Item{ID: w.newID(), Kind: "livestock", Ref: l.ID, Victim: l.Victim})
			w.Livestock = append(w.Livestock, l)
			w.dropBack(p)
		}
	}
	w.say(e.TextKind, "%s", e.Text)
	if e.Post != "" {
		w.post("info", "%s", e.Post)
	}
}

func price(d Item) int { return RecipeByID(d.Sub).Price + 5*(d.Quality-3) }

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Toggle switches a machine off or on. Off machines take no jobs.
func (w *World) Toggle(id string) error {
	f := FixtureByID(id)
	if f == nil || !w.Has(id) {
		return errors.New("you don't have that")
	}
	m := w.machine(id)
	m.Off = !m.Off
	if m.Off {
		w.say("info", "%s switched off.", f.Name)
	} else {
		w.say("info", "%s switched on.", f.Name)
	}
	return nil
}

func (w *World) SetPolicy(p Policy) {
	if p.Sell == nil {
		p.Sell = map[string]bool{}
	}
	p.ButcherMarbling = clamp(p.ButcherMarbling, 1, 5)
	p.KeepLivestock = clamp(p.KeepLivestock, 0, 5)
	p.PantryTarget = clamp(p.PantryTarget, 0, 16)
	p.FeedReserve = clamp(p.FeedReserve, 0, 500)
	w.Policy = p
}

// Order gives one guest a direct instruction: "chair", "butcher", or "" to
// go back to policy. The next free worker carries it out.
func (w *World) Order(id int, order string) error {
	switch order {
	case "", "chair", "butcher":
	default:
		return errors.New("no such order")
	}
	if order == "chair" && !w.Has("chair") {
		return errors.New("you don't have The Chair")
	}
	l := w.livestockByID(id)
	if l == nil || l.Cell == "" {
		return errors.New("no such guest, or they're already being moved")
	}
	l.Order = order
	switch order {
	case "chair":
		if w.Renfields() >= w.CotCount() {
			w.say("warn", "%s is marked for The Chair, but every cot is taken. Build one in the attic.", l.Name)
		} else {
			w.say("info", "%s is next for The Chair.", l.Name)
		}
	case "butcher":
		w.say("info", "%s is next for the block.", l.Name)
	default:
		w.say("info", "%s goes back to being livestock like anybody else.", l.Name)
	}
	return nil
}

// SetRole assigns a Renfield a form of servitude.
func (w *World) SetRole(id int, role string) error {
	if RoleName(role) == "Hand" && role != RoleHand {
		return errors.New("no such role")
	}
	for _, wk := range w.Workers {
		if wk.ID == id && wk.Kind != "cannibal" {
			wk.Role = role
			w.dropBack(wk) // put down whatever they were carrying before the job goes away
			w.unreserve(jobMachine(wk))
			wk.Job = nil
			wk.Path = nil
			w.say("info", "%s is now the %s.", wk.Name, strings.ToLower(RoleName(role)))
			return nil
		}
	}
	return errors.New("no such Renfield")
}

func (w *World) Rename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 28 {
		return errors.New("pick a name between 1 and 28 characters")
	}
	w.Cannibal = name
	if p := w.Player(); p != nil {
		p.Name = name
	}
	w.say("info", "You are %s now. The mail will take a while to catch up.", name)
	return nil
}

// Inherit hands the house to a relative after a death. Fixtures and recipes
// stay; cash halves; everything alive, edible or incriminating is gone.
func (w *World) Inherit() error {
	if w.Alive {
		return fmt.Errorf("%s is alive and well-fed", w.Cannibal)
	}
	w.Ancestors = append(w.Ancestors, Ancestor{Name: w.Cannibal, Days: w.DaysAlive(), Meals: w.Stats.Meals,
		Stars: w.Stars, Cause: w.Cause, Captured: w.Stats.Captured})
	old := w.Cannibal
	w.Generation++
	w.Cannibal = randomCannibalName(w.rng())
	w.Born = w.Clock
	w.Cash = float64(int(w.Cash) / 2)
	if w.Cash < 60 {
		w.Cash = 60
	}
	w.Heat = 15
	w.Stars = 0
	w.Stats = Stats{}
	w.Livestock, w.Passers, w.Visitors, w.Missing = nil, nil, nil, nil
	w.Fridge, w.Pantry, w.Pile, w.Driveway, w.Stand = nil, nil, nil, nil, nil
	w.Buried, w.Compost = 0, 0
	w.Machines = map[string]*Machine{}
	w.RaidCalled = false
	w.InspectTier = 0
	w.Workers = []*Worker{{ID: w.newID(), Name: w.Cannibal, Kind: "cannibal", Pos: w.Pos("bed"), Hunger: 70, Energy: 90}}
	w.Alive, w.Cause = true, ""
	w.say("info", "%s inherits the house from %s. The freezer came empty. The neighbors came curious.", w.Cannibal, old)
	w.post("info", "has a new owner: %s. The estate sale was mostly tupperware.", w.Cannibal)
	return nil
}

// Reset burns the dynasty down and starts a brand-new house: new cannibal,
// starter kitchen, no ancestors, nothing carried over but the clock scale
// and the color preference.
func (w *World) Reset() {
	fresh := New(w.Handle, w.BBS, w.Seed*31+w.Clock+7)
	fresh.Scale = w.Scale
	fresh.ColorMode = w.ColorMode
	fresh.JournalOrder = w.JournalOrder
	fresh.LastWall = w.LastWall
	old := w.Cannibal
	*w = *fresh
	w.say("info", "The old place burned to the foundation. %s moves in and starts over.", w.Cannibal)
	w.post("info", "'s house changed hands. Whatever happened to %s, the new owner seems nice. Waves a lot.", old)
}

// Command is the wire form of a player action.
type Command struct {
	Kind   string  `json:"kind"` // build | move | toggle | policy | rename | inherit | color | errand | advance
	Name   string  `json:"name,omitempty"`
	At     *Pos    `json:"at,omitempty"`
	ID     int     `json:"id,omitempty"` // role: which Renfield
	To     string  `json:"to,omitempty"` // errand raid: a neighbor's key ("" = a CPU house)
	Policy *Policy `json:"policy,omitempty"`
	Ticks  int     `json:"ticks,omitempty"`
}

// Apply runs a command. "advance" is how a connected door drives the clock
// forward (the server ticks worlds itself; the local backend uses this).
func (w *World) Apply(c Command) error {
	switch c.Kind {
	case "build":
		at := Pos{}
		if c.At != nil {
			at = *c.At
		} else if replaces[c.Name] == "" || !w.Has(replaces[c.Name]) {
			return errors.New("build needs a tile")
		}
		return w.Build(c.Name, at)
	case "move":
		if c.At == nil {
			return errors.New("move needs a tile")
		}
		return w.Move(c.Name, *c.At)
	case "errand":
		return w.StartErrand(c.Name)
	case "role":
		return w.SetRole(c.ID, c.Name)
	case "order":
		return w.Order(c.ID, c.Name)
	case "toggle":
		return w.Toggle(c.Name)
	case "policy":
		if c.Policy == nil {
			return errors.New("no policy")
		}
		w.SetPolicy(*c.Policy)
		return nil
	case "rename":
		return w.Rename(c.Name)
	case "inherit":
		return w.Inherit()
	case "reset":
		w.Reset()
		return nil
	case "color":
		w.ColorMode = c.Name
		return nil
	case "journal":
		w.JournalOrder = c.Name
		return nil
	case "advance":
		if c.Ticks < 0 || c.Ticks > 4*day {
			return errors.New("advance out of range")
		}
		w.Advance(c.Ticks)
		return nil
	}
	return fmt.Errorf("unknown command %q", c.Kind)
}
