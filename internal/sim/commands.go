package sim

import (
	"errors"
	"fmt"
	"strings"
)

// Commands are the player's hand: build, switch machines, tune policy, rename,
// inherit. They are the only way the door changes the world; the clock does
// the rest.

// Build buys a fixture. An upgrade sharing a slot (the trapdoors) replaces its
// predecessor.
func (w *World) Build(id string) error {
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
	w.Cash -= float64(f.Cost)
	for _, o := range FixturesAtSlot(f.Level, f.Slot) {
		if o.ID != id && w.Fixtures[o.ID] && o.Kind == f.Kind {
			delete(w.Fixtures, o.ID)
		}
	}
	w.Fixtures[id] = true
	w.say("good", "Installed: %s. %s", f.Name, f.Desc)
	if f.Kind == KindCell {
		w.post("info", "put in a new %s. \"Wine cellar,\" they said.", strings.ToLower(f.Name))
	}
	return nil
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
	w.Workers = []*Worker{{ID: w.newID(), Name: w.Cannibal, Kind: "cannibal", Pos: SlotPos(FixtureByID("bed")), Hunger: 70, Energy: 90}}
	w.Alive, w.Cause = true, ""
	w.say("info", "%s inherits the house from %s. The freezer came empty. The neighbors came curious.", w.Cannibal, old)
	w.post("info", "has a new owner: %s. The estate sale was mostly tupperware.", w.Cannibal)
	return nil
}

// Command is the wire form of a player action.
type Command struct {
	Kind   string  `json:"kind"` // build | toggle | policy | rename | inherit | color | advance
	Name   string  `json:"name,omitempty"`
	Policy *Policy `json:"policy,omitempty"`
	Ticks  int     `json:"ticks,omitempty"`
}

// Apply runs a command. "advance" is how a connected door drives the clock
// forward (the server ticks worlds itself; the local backend uses this).
func (w *World) Apply(c Command) error {
	switch c.Kind {
	case "build":
		return w.Build(c.Name)
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
	case "color":
		w.ColorMode = c.Name
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
