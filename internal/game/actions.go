package game

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// Action is what the door sends. Kind picks the verb; the other fields are
// verb-specific (ids of captives/dishes/evidence/cuts, or a table id in Name).
type Action struct {
	Kind string `json:"kind"`
	ID   int    `json:"id,omitempty"`
	IDs  []int  `json:"ids,omitempty"`
	Name string `json:"name,omitempty"`
}

// Event is one line of feedback. Kind tints it in the door: info, good, bad,
// heat, cash, food, warn.
type Event struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
}

// Result is what an Action produced. Posts are public: the world layer puts
// them on the neighborhood feed under the player's handle.
type Result struct {
	Events []Event `json:"events"`
	Posts  []Event `json:"posts,omitempty"`
	Err    string  `json:"err,omitempty"`
}

func (r *Result) say(kind, format string, a ...any) {
	r.Events = append(r.Events, Event{Text: fmt.Sprintf(format, a...), Kind: kind})
}

func (r *Result) post(kind, format string, a ...any) {
	r.Posts = append(r.Posts, Event{Text: fmt.Sprintf(format, a...), Kind: kind})
}

func (r *Result) fail(format string, a ...any) Result {
	r.Err = fmt.Sprintf(format, a...)
	return *r
}

// GrantAllowance tops up the day bank once per real day. today is YYYY-MM-DD.
func GrantAllowance(s *State, rules Rules, today string) bool {
	if s.LastGrant == today {
		return false
	}
	s.LastGrant = today
	s.Bank = clamp(s.Bank+rules.DaysPerDay, 0, rules.BankMax)
	return true
}

// apply runs one action against the state. It never panics on bad input: an
// impossible action returns Result.Err and leaves the state untouched.
func apply(s *State, a Action) Result {
	r := Result{}
	if !s.Alive && a.Kind != "inherit" && a.Kind != "set_color" {
		return r.fail("%s is no longer with us. Someone must inherit the house.", s.Cannibal)
	}
	if s.Mission != nil {
		switch a.Kind {
		case "end_phase", "rename", "set_color", "inherit":
		default:
			return r.fail("You're not home. %s is still underway.", MissionByID(s.Mission.ID).Name)
		}
	}
	switch a.Kind {
	case "end_phase":
		return s.endPhase(&r)
	case "build":
		return s.build(&r, a.Name)
	case "hunt":
		return s.hunt(&r, a.Name)
	case "butcher":
		return s.butcher(&r, a.ID)
	case "cook":
		return s.cook(&r, a.Name, a.IDs)
	case "eat":
		return s.eat(&r, a.ID)
	case "dispose":
		return s.dispose(&r, a.ID, a.Name)
	case "freeze":
		return s.freeze(&r, a.ID, true)
	case "thaw":
		return s.freeze(&r, a.ID, false)
	case "brainwash":
		return s.brainwash(&r, a.ID)
	case "mission":
		return s.startMission(&r, a.Name)
	case "dinner":
		return s.dinner(&r)
	case "rename":
		name := strings.TrimSpace(a.Name)
		if name == "" || len(name) > 28 {
			return r.fail("Pick a name between 1 and 28 characters.")
		}
		s.Cannibal = name
		r.say("info", "You are %s now. The mail will take a while to catch up.", name)
		return r
	case "set_color":
		s.ColorMode = a.Name
		return r
	case "inherit":
		return s.inherit(&r)
	}
	return r.fail("Unknown action %q.", a.Kind)
}

// Apply runs one action through the reducer and then the death check, so an
// action that pushes Heat to 100 ends the run immediately.
func Apply(s *State, a Action) Result {
	r := apply(s, a)
	if r.Err == "" {
		s.checkDeath(&r)
	}
	return r
}

// ---------------------------------------------------------------------------
// building
// ---------------------------------------------------------------------------

func (s *State) build(r *Result, id string) Result {
	f := FixtureByID(id)
	if f == nil {
		return r.fail("No such upgrade.")
	}
	if s.Has(id) {
		return r.fail("You already have the %s.", f.Name)
	}
	if f.Prereq != "" && !s.Has(f.Prereq) {
		return r.fail("Needs the %s first.", FixtureByID(f.Prereq).Name)
	}
	if s.Cash < f.Cost {
		return r.fail("The %s costs $%d. You have $%d.", f.Name, f.Cost, s.Cash)
	}
	s.Cash -= f.Cost
	s.Fixtures[id] = true
	r.say("good", "Installed: %s. %s", f.Name, f.Desc)
	for _, rc := range Recipes {
		if !s.Recipes[rc.ID] && s.CanCook(&rc) && contains(rc.Needs, id) {
			s.Recipes[rc.ID] = true
			r.say("food", "New recipe: %s.", rc.Name)
		}
	}
	if f.Cell {
		r.post("info", "put in a new %s. \"Wine cellar,\" they said.", strings.ToLower(f.Name))
	}
	s.log("Built %s.", f.Name)
	return *r
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// hunting and livestock
// ---------------------------------------------------------------------------

// rollPrey picks who's out on the street tonight: three archetypes weighted by
// rarity, lures, and the current Heat tier (podcasters only show up once
// you're a story). High Heat thins the street: people stop walking past.
func rollPrey(s *State, rng *rand.Rand) []string {
	type cand struct {
		id string
		w  int
	}
	var pool []cand
	for _, v := range Victims {
		if v.Weight == 0 || s.Tier() < v.MinTier {
			continue
		}
		w := v.Weight
		for _, l := range v.Lures {
			if s.Has(l) {
				w += 6
			}
		}
		pool = append(pool, cand{v.ID, w})
	}
	n := 3
	if s.Tier() >= 4 {
		n = 2
	}
	var out []string
	for len(out) < n && len(pool) > 0 {
		total := 0
		for _, c := range pool {
			total += c.w
		}
		x := rng.Intn(total)
		for i, c := range pool {
			x -= c.w
			if x < 0 {
				out = append(out, c.id)
				pool = append(pool[:i], pool[i+1:]...)
				break
			}
		}
	}
	return out
}

func (s *State) huntChance(v *Victim) int {
	c := 55 - v.Fight*8
	if s.Has("van") {
		c += 12
	}
	if s.Has("chemistry") {
		c += 12
	}
	if s.Has("telescope") {
		c += 6
	}
	if s.Has("gnome") {
		c += 3
	}
	if s.Has("shed") {
		c += 3
	}
	return clamp(c, 10, 95)
}

// HuntChance is exposed so the door can show the odds on the prey menu.
func (s *State) HuntChance(id string) int { return s.huntChance(VictimByID(id)) }

func (s *State) hunt(r *Result, id string) Result {
	if s.Phase != Night {
		return r.fail("Hunting is a night thing. Wait for dark.")
	}
	if !contains(s.Prey, id) {
		return r.fail("Nobody like that is out tonight.")
	}
	v := VictimByID(id)
	rng := s.rng()
	s.Stats.Hunts++
	s.Prey = nil // one hunt per night, however it goes
	if rng.Intn(100) >= s.huntChance(v) {
		if rng.Intn(2) == 0 {
			s.Heat = clamp(s.Heat+5+v.Fight, 0, 120)
			r.say("bad", "The %s got away. There was some yelling. Heat rises.", strings.ToLower(v.Name))
		} else {
			s.Hunger = clamp(s.Hunger-8, 0, 100)
			s.Heat = clamp(s.Heat+2, 0, 120)
			r.say("bad", "The %s fought back and you ate pavement instead. Hungry and bruised.", strings.ToLower(v.Name))
		}
		s.log("Failed hunt: %s.", v.Name)
		return *r
	}
	s.Heat = clamp(s.Heat+2, 0, 120)
	s.capture(r, v, rng)
	if v.Pair {
		r.say("good", "There were two of them. There are always two of them.")
		s.capture(r, v, rng)
	}
	if v.Drives {
		if s.Has("van") {
			r.say("info", "Their car is on the van's hitch. It'll be parts by morning.")
		} else {
			s.Evidence = append(s.Evidence, Evidence{ID: s.newID(), Kind: "car", Victim: v.ID})
			r.say("heat", "Their car is in your driveway. That's evidence with a license plate.")
		}
	}
	return *r
}

// capture adds a victim as livestock, or butchers them on the doorstep when
// the cells are full.
func (s *State) capture(r *Result, v *Victim, rng *rand.Rand) {
	s.Stats.Captured++
	s.Missing = append(s.Missing, Missing{Victim: v.ID, Days: v.Notice})
	name := captiveNames[rng.Intn(len(captiveNames))]
	if s.Livestock() < s.CellCount() {
		c := Captive{ID: s.newID(), Victim: v.ID, Name: name, Marbling: v.Marbling}
		s.Captives = append(s.Captives, c)
		r.say("good", "Captured: %s the %s. %s", name, v.Name, v.Flavor)
		s.log("Captured %s (%s).", name, v.Name)
		return
	}
	r.say("warn", "No cell free for %s the %s. Straight to the kitchen, then.", name, v.Name)
	s.yieldCuts(r, v, v.Marbling)
}

func (s *State) butcher(r *Result, id int) Result {
	i := s.findCaptive(id)
	if i < 0 {
		return r.fail("No such guest.")
	}
	c := s.Captives[i]
	v := VictimByID(c.Victim)
	s.Captives = append(s.Captives[:i], s.Captives[i+1:]...)
	if c.Renfield {
		r.say("warn", "%s trusted you. Loyalty is delicious.", c.Name)
	} else {
		r.say("info", "%s the %s has been processed. Hum something.", c.Name, v.Name)
	}
	s.yieldCuts(r, v, c.Marbling)
	s.log("Butchered %s.", c.Name)
	return *r
}

// yieldCuts turns a victim into fridge contents plus the evidence they leave.
func (s *State) yieldCuts(r *Result, v *Victim, marbling int) {
	prime := 2
	if s.Has("knife_set") {
		prime++
	}
	scraps := 1
	if marbling >= 4 {
		scraps++
	}
	var cuts []Cut
	for i := 0; i < prime; i++ {
		cuts = append(cuts, Cut{Kind: CutPrime, Victim: v.ID, Marbling: marbling, Freshness: 3})
	}
	cuts = append(cuts, Cut{Kind: CutOffal, Victim: v.ID, Marbling: marbling, Freshness: 2})
	for i := 0; i < scraps; i++ {
		cuts = append(cuts, Cut{Kind: CutScraps, Victim: v.ID, Marbling: marbling, Freshness: 3})
	}
	stored, wasted := 0, 0
	fresh, _ := s.fridgeUsed()
	for _, c := range cuts {
		if fresh < s.FridgeCap() {
			c.ID = s.newID()
			s.Fridge = append(s.Fridge, c)
			fresh++
			stored++
		} else {
			wasted++
		}
	}
	r.say("food", "%d cuts in the fridge (%d prime, 1 offal, %d scraps).", stored, minInt(prime, stored), scraps)
	if wasted > 0 {
		r.say("bad", "%d cuts wouldn't fit. Wasted. Buy a bigger fridge.", wasted)
	}
	for _, k := range []string{"bones", "clothes", "phone"} {
		s.Evidence = append(s.Evidence, Evidence{ID: s.newID(), Kind: k, Victim: v.ID})
	}
	r.say("heat", "Left behind: bones, clothes, a phone that keeps lighting up.")
}

func (s *State) findCaptive(id int) int {
	for i, c := range s.Captives {
		if c.ID == id {
			return i
		}
	}
	return -1
}

func (s *State) brainwash(r *Result, id int) Result {
	if !s.Has("chair") {
		return r.fail("You need The Chair for that.")
	}
	i := s.findCaptive(id)
	if i < 0 {
		return r.fail("No such guest.")
	}
	c := &s.Captives[i]
	if c.Renfield {
		return r.fail("%s is already yours, body and soul.", c.Name)
	}
	if c.LastTry == s.Day {
		return r.fail("%s has had enough tapes for one day.", c.Name)
	}
	if s.Renfields() >= s.RenfieldCap() {
		return r.fail("Nowhere for another Renfield to sleep. Build a cot in the attic.")
	}
	c.LastTry = s.Day
	chance := 25 + 20*c.Sessions
	if s.Has("vhs") {
		chance += 20
	}
	if s.rng().Intn(100) < chance {
		c.Renfield = true
		c.Loyalty = 70
		if s.Has("big_tv") {
			c.Loyalty = 85
		}
		r.say("good", "%s blinks twice and asks what needs doing. A Renfield. The cell is free.", c.Name)
		r.post("info", "has a new 'roommate'. Very quiet. Very helpful.")
		s.log("%s became a Renfield.", c.Name)
	} else {
		c.Sessions++
		r.say("info", "Session %d. %s wept through tape %d. Progress.", c.Sessions, c.Name, c.Sessions*3)
	}
	return *r
}

// ---------------------------------------------------------------------------
// kitchen
// ---------------------------------------------------------------------------

func (s *State) cook(r *Result, id string, ids []int) Result {
	rc := RecipeByID(id)
	if rc == nil {
		return r.fail("No such recipe.")
	}
	if !s.CanCook(rc) {
		var need []string
		for _, n := range rc.Needs {
			if !s.Has(n) {
				need = append(need, FixtureByID(n).Name)
			}
		}
		return r.fail("%s needs: %s.", rc.Name, strings.Join(need, ", "))
	}
	quality := rc.Base
	if rc.Bones {
		bi := -1
		for i, e := range s.Evidence {
			if e.Kind == "bones" && !e.Buried {
				bi = i
				break
			}
		}
		if bi < 0 {
			return r.fail("No bones in the house. Broth needs bones.")
		}
		s.Evidence = append(s.Evidence[:bi], s.Evidence[bi+1:]...)
		r.say("good", "The bones simmered down to nothing. Evidence, gone. Collagen, acquired.")
	} else {
		picked, err := s.pickCuts(rc, ids)
		if err != "" {
			return r.fail("%s", err)
		}
		marb, worst, frozen := 0, 9, false
		for _, i := range picked {
			c := s.Fridge[i]
			marb += c.Marbling
			if c.Frozen {
				frozen = true
			} else if c.Freshness < worst {
				worst = c.Freshness
			}
		}
		marb /= len(picked)
		switch {
		case marb <= 2:
			quality--
		case marb >= 4:
			quality++
		}
		if worst <= 1 {
			quality--
		}
		if frozen {
			quality--
		}
		sort.Sort(sort.Reverse(sort.IntSlice(picked)))
		for _, i := range picked {
			s.Fridge = append(s.Fridge[:i], s.Fridge[i+1:]...)
		}
	}
	quality = clamp(quality, 1, 5)
	d := Dish{ID: s.newID(), Recipe: rc.ID, Quality: quality, Shelf: rc.Shelf}
	s.Dishes = append(s.Dishes, d)
	if !s.Recipes[rc.ID] {
		s.Recipes[rc.ID] = true
		r.say("food", "First time making %s. %s", rc.Name, rc.Flavor)
	}
	r.say("food", "Made %s (%s).", rc.Name, starsText(quality))
	if quality == 5 {
		r.post("good", "plated a five-star %s. The smell reached three houses.", rc.Name)
	}
	s.log("Cooked %s (%d star).", rc.Name, quality)
	return *r
}

// pickCuts resolves which fridge entries a recipe consumes: the caller's ids
// if given, else the freshest unfrozen cuts of each kind (frozen as fallback).
func (s *State) pickCuts(rc *Recipe, ids []int) ([]int, string) {
	need := map[string]int{}
	for k, n := range rc.Cuts {
		need[k] = n
	}
	var picked []int
	used := map[int]bool{}
	if len(ids) > 0 {
		for _, id := range ids {
			for i, c := range s.Fridge {
				if c.ID == id && !used[i] && need[c.Kind] > 0 {
					picked = append(picked, i)
					used[i] = true
					need[c.Kind]--
				}
			}
		}
	}
	for k, n := range need {
		for n > 0 {
			best := -1
			for i, c := range s.Fridge {
				if used[i] || c.Kind != k {
					continue
				}
				if best < 0 || better(c, s.Fridge[best]) {
					best = i
				}
			}
			if best < 0 {
				return nil, fmt.Sprintf("Not enough %s in the fridge for %s.", k, rc.Name)
			}
			picked = append(picked, best)
			used[best] = true
			n--
		}
	}
	return picked, ""
}

// better prefers unfrozen over frozen, then the cut closest to spoiling
// (use it or lose it), then higher marbling.
func better(a, b Cut) bool {
	if a.Frozen != b.Frozen {
		return !a.Frozen
	}
	if !a.Frozen && a.Freshness != b.Freshness {
		return a.Freshness < b.Freshness
	}
	return a.Marbling > b.Marbling
}

func starsText(q int) string {
	return strings.Repeat("*", q) + strings.Repeat("-", 5-q)
}

func (s *State) eat(r *Result, id int) Result {
	for i, d := range s.Dishes {
		if d.ID != id {
			continue
		}
		rc := RecipeByID(d.Recipe)
		if !rc.Food {
			return r.fail("You don't eat the %s. You sell the %s.", rc.Name, rc.Name)
		}
		s.Dishes = append(s.Dishes[:i], s.Dishes[i+1:]...)
		gain := rc.Restore + 5*(d.Quality-3)
		s.Hunger = clamp(s.Hunger+gain, 0, 100)
		s.Meals++
		stars := d.Quality - 2
		if stars > 0 {
			s.Stars += stars
			r.say("good", "Ate %s. +%d hunger, +%d star%s.", rc.Name, gain, stars, plural(stars))
		} else {
			r.say("food", "Ate %s. +%d hunger. Not your best work.", rc.Name, gain)
		}
		s.log("Ate %s.", rc.Name)
		return *r
	}
	return r.fail("No such dish.")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (s *State) freeze(r *Result, id int, freeze bool) Result {
	for i := range s.Fridge {
		c := &s.Fridge[i]
		if c.ID != id {
			continue
		}
		if freeze {
			if c.Frozen {
				return r.fail("Already frozen.")
			}
			if _, fz := s.fridgeUsed(); fz >= s.FreezerCap() {
				return r.fail("The freezer is full.")
			}
			c.Frozen = true
			r.say("info", "Frozen. It'll keep, at the cost of a star.")
		} else {
			if !c.Frozen {
				return r.fail("Not frozen.")
			}
			if fr, _ := s.fridgeUsed(); fr >= s.FridgeCap() {
				return r.fail("The fridge is full.")
			}
			c.Frozen = false
			c.Freshness = 2
			r.say("info", "Thawing. Use it in the next couple of days.")
		}
		return *r
	}
	return r.fail("No such cut.")
}

// ---------------------------------------------------------------------------
// evidence
// ---------------------------------------------------------------------------

// DisposalMethods lists what can be done with a piece of evidence right now,
// so the door can offer only the real options.
func (s *State) DisposalMethods(e Evidence) []string {
	var m []string
	switch e.Kind {
	case "car":
		m = append(m, "abandon")
		if s.Has("van") {
			m = append(m, "van")
		}
		if s.Has("chipper") {
			m = append(m, "chipper")
		}
		return m
	case "phone":
		if s.Has("microwave") {
			m = append(m, "microwave")
		}
	case "clothes":
		m = append(m, "donate")
	case "bones":
		if s.Has("compost") {
			m = append(m, "compost")
		}
	}
	if !e.Buried && s.PlotCap() > s.Buried() {
		m = append(m, "bury")
	}
	if s.Has("chipper") {
		m = append(m, "chipper")
	}
	if s.Has("lye_tub") && (e.Kind == "bones" || e.Kind == "clothes") {
		m = append(m, "lye")
	}
	return m
}

func (s *State) dispose(r *Result, id int, method string) Result {
	idx := -1
	for i, e := range s.Evidence {
		if e.ID == id {
			idx = i
		}
	}
	if idx < 0 {
		return r.fail("No such evidence.")
	}
	e := s.Evidence[idx]
	if !contains(s.DisposalMethods(e), method) {
		return r.fail("You can't %s the %s. Not with what you own.", method, e.Kind)
	}
	remove := true
	rng := s.rng()
	switch method {
	case "bury":
		s.Evidence[idx].Buried = true
		s.Stats.Buried++
		remove = false
		if s.Has("shed") {
			r.say("good", "Buried under the flowers. The shed's shovel is quiet.")
		} else {
			s.Heat = clamp(s.Heat+1, 0, 120)
			r.say("info", "Buried under the flowers. Digging at 2 A.M. draws a look.")
		}
	case "chipper":
		s.Heat = clamp(s.Heat+4, 0, 120)
		r.say("heat", "The chipper roared. Gone in seconds. The whole street heard it.")
	case "lye":
		if rng.Intn(100) < 10 {
			s.Heat = clamp(s.Heat+8, 0, 120)
			r.say("bad", "The tub dissolved. So did the floor. Gladys heard the crash.")
		} else {
			r.say("good", "Dissolved. Yeah, science.")
		}
	case "microwave":
		r.say("good", "Thirty seconds on high. The phone stopped pinging. Sparks were pretty.")
	case "donate":
		if rng.Intn(100) < 30 {
			s.Heat = clamp(s.Heat+6, 0, 120)
			r.say("bad", "Goodwill took the clothes. Then someone recognized the jacket.")
		} else {
			r.say("good", "Donated. Tax deductible.")
		}
	case "compost":
		s.Compost++
		r.say("good", "Bones into the compost. The tomatoes will be enormous.")
	case "van":
		r.say("good", "Drove the car to a lot two counties over. Walked home. Nice night.")
	case "abandon":
		if rng.Intn(100) < 30 {
			s.Heat = clamp(s.Heat+8, 0, 120)
			r.say("bad", "Left the car at the mall. A traffic cam got you walking away from it. Heat +8.")
		} else {
			r.say("good", "Left the car at the mall with the keys in it. Somebody else's problem by noon.")
		}
	}
	if remove {
		s.Evidence = append(s.Evidence[:idx], s.Evidence[idx+1:]...)
	}
	s.log("Disposed of %s (%s).", e.Kind, method)
	return *r
}

// ---------------------------------------------------------------------------
// social
// ---------------------------------------------------------------------------

func (s *State) dinner(r *Result) Result {
	if !s.Has("dining_set") {
		return r.fail("You need a Dining Set to host anyone.")
	}
	if s.Phase != Evening {
		return r.fail("Dinner parties happen in the Evening.")
	}
	best := -1
	for i, d := range s.Dishes {
		if RecipeByID(d.Recipe).Food && d.Quality >= 3 && (best < 0 || d.Quality > s.Dishes[best].Quality) {
			best = i
		}
	}
	if best < 0 {
		return r.fail("Nothing worth serving. Guests expect three stars or better.")
	}
	d := s.Dishes[best]
	rc := RecipeByID(d.Recipe)
	s.Dishes = append(s.Dishes[:best], s.Dishes[best+1:]...)
	drop := 5 * d.Quality
	if s.Has("wine_rack") {
		drop += 3
	}
	s.Heat = clamp(s.Heat-drop, 0, 120)
	s.Stars += d.Quality
	s.Stats.Dinners++
	r.say("good", "The neighbors came for %s. They asked for the recipe. Heat -%d, +%d stars.", rc.Name, drop, d.Quality)
	r.post("good", "hosted a dinner party. \"Best %s on the street,\" said Gladys, licking the plate.", strings.ToLower(rc.Name))
	s.log("Dinner party: %s.", rc.Name)
	s.Phase = Night
	s.Prey = rollPrey(s, s.rng())
	r.say("info", "The guests left late. It's Night.")
	return *r
}

func (s *State) startMission(r *Result, id string) Result {
	m := MissionByID(id)
	if m == nil {
		return r.fail("No such mission.")
	}
	if s.Tier() < m.MinTier {
		return r.fail("Nobody's asking for a hero yet.")
	}
	if id == "market" {
		if s.Phase == Night {
			return r.fail("The market is closed at night.")
		}
		if !s.Has("garden") && s.sellable() == 0 {
			return r.fail("Nothing to sell. Bake some pies first.")
		}
	}
	if id == "day_job" && s.Phase != Morning {
		return r.fail("Work starts in the Morning. Consulting has standards.")
	}
	if s.Phase == Night && id != "city_hunt" && id != "tinder" {
		return r.fail("Not at night. Sleep, or hunt.")
	}
	s.Mission = &Mission{ID: id, Phases: m.Phases}
	s.Stats.Missions++
	r.say("info", "%s: you lock up and go. The house is on its own for %d phase%s.", m.Name, m.Phases, plural(m.Phases))
	if s.Livestock() > 0 && s.Renfields() == 0 && !s.Has("cameras") {
		r.say("warn", "Nobody is watching the basement.")
	}
	s.log("Mission: %s.", m.Name)
	return *r
}

func (s *State) sellable() int {
	n := 0
	for _, d := range s.Dishes {
		if RecipeByID(d.Recipe).Price > 0 {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// death and inheritance
// ---------------------------------------------------------------------------

func (s *State) inherit(r *Result) Result {
	if s.Alive {
		return r.fail("%s is alive and well-fed.", s.Cannibal)
	}
	s.Ancestors = append(s.Ancestors, Ancestor{
		Name: s.Cannibal, Days: s.Day, Meals: s.Meals, Stars: s.Stars, Cause: s.Cause, Captured: s.Stats.Captured,
	})
	old := s.Cannibal
	s.Generation++
	s.Cannibal = randomCannibalName(s.rng())
	s.Day, s.Phase = 1, Morning
	s.Hunger, s.Heat = 60, 15
	s.Cash /= 2
	s.Stars, s.Meals = 0, 0
	s.Fridge, s.Dishes, s.Captives, s.Evidence, s.Missing, s.Prey = nil, nil, nil, nil, nil, nil
	s.Mission, s.Inspection = nil, nil
	s.Stats = Stats{}
	s.Compost = 0
	s.Alive, s.Cause = true, ""
	r.say("info", "%s inherits the house from %s. The freezer came empty. The neighbors came curious.", s.Cannibal, old)
	r.post("info", "has a new owner: %s. The estate sale was mostly tupperware.", s.Cannibal)
	s.log("%s inherited the house.", s.Cannibal)
	return *r
}

// DynastyStars is the score across every cannibal who has held the house.
func (s *State) DynastyStars() int {
	n := s.Stars
	for _, a := range s.Ancestors {
		n += a.Stars
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
