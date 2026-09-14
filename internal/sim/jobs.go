package sim

import (
	"fmt"
	"sort"
)

// Job selection: a worker with nothing to do asks, in priority order, whether
// there's an eat, sleep, feed, collect, dispose, butcher, cook, stock, or
// brainwash job available. Each job is a list of steps (walk here, work this
// long, then do this), so a half-finished job survives a save.

func (w *World) pickJob(wk *Worker) *Job {
	if wk.Hunger < 35 {
		if j := w.jobEat(wk); j != nil {
			return j
		}
	}
	if wk.Energy < 15 {
		if j := w.jobSleep(wk); j != nil {
			return j
		}
	}
	if wk.Hunger < 20 {
		if j := w.jobCook(wk, true); j != nil { // starving: cook anything edible first
			return j
		}
	}
	// A direct order from the player comes before routine work.
	if w.hasOrder("chair") {
		if j := w.jobBrainwash(wk); j != nil {
			return j
		}
	}
	if w.hasOrder("butcher") {
		if j := w.jobButcher(wk); j != nil {
			return j
		}
	}
	order := []func(*Worker) *Job{w.jobFeed, w.jobCollect, w.jobDispose, w.jobButcher, w.jobCookAny, w.jobStock, w.jobBrainwash}
	switch wk.Role {
	case RoleCook:
		order = []func(*Worker) *Job{w.jobCollect, w.jobCookAny, w.jobFeed, w.jobStock, w.jobButcher, w.jobDispose, w.jobBrainwash}
	case RoleGuard, RoleGimp:
		order = []func(*Worker) *Job{w.jobFeed, w.jobButcher} // stays near the cells; the rest is not their job
	case RoleShill:
		order = []func(*Worker) *Job{w.jobStock} // lives on the porch, minds the stand
	}
	for _, f := range order {
		if j := f(wk); j != nil {
			return j
		}
	}
	return w.jobIdle(wk)
}

func (w *World) stepTo(f string, work int, do string) Step {
	return Step{Dest: w.Pos(f), Work: work, Do: do}
}

func (w *World) jobEat(wk *Worker) *Job {
	best := -1
	for i, d := range w.Pantry {
		if RecipeByID(d.Sub).Food && (best < 0 || d.Quality > w.Pantry[best].Quality) {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	return &Job{Kind: "eat", Label: "getting a plate", Target: w.Pantry[best].ID,
		Steps: []Step{w.stepTo("pantry", 0, "take_dish"), w.stepTo("pantry", 15, "eat")}}
}

func (w *World) jobSleep(wk *Worker) *Job {
	bed := "bed"
	if wk.Kind != "cannibal" {
		bed = ""
		for _, f := range []string{"cot", "cot_2", "cot_3", "cot_4"} {
			if w.Has(f) && !w.bedTaken(f, wk) {
				bed = f
				break
			}
		}
		if bed == "" {
			label := "dozing in a corner"
			if wk.Kind == "convert" {
				label = "sleeping on the couch, like family"
			}
			return &Job{Kind: "sleep", Label: label, Steps: []Step{{Dest: wk.Pos, Do: "sleep"}}}
		}
	}
	return &Job{Kind: "sleep", Label: "going to bed", Machine: bed, Steps: []Step{w.stepTo(bed, 0, "sleep")}}
}

func (w *World) bedTaken(bed string, me *Worker) bool {
	for _, o := range w.Workers {
		if o != me && o.Job != nil && o.Job.Kind == "sleep" && o.Job.Machine == bed {
			return true
		}
	}
	return false
}

func (w *World) jobFeed(wk *Worker) *Job {
	if w.Has("gruel") || w.Cash < float64(4+w.Policy.FeedReserve) {
		return nil
	}
	for _, l := range w.Livestock {
		if l.Cell != "" && l.Hunger > 50 && !w.targeted(l.ID) {
			return &Job{Kind: "feed", Label: "feeding " + l.Name, Target: l.ID,
				Steps: []Step{w.stepTo(l.Cell, 5, "feed")}}
		}
	}
	return nil
}

// targeted reports whether another worker's job already claims a livestock.
func (w *World) targeted(id int) bool {
	for _, o := range w.Workers {
		if o.Job != nil && o.Job.Target == id && o.Job.Kind != "eat" {
			return true
		}
	}
	return false
}

func (w *World) jobCollect(wk *Worker) *Job {
	if len(w.Pantry) >= w.PantryCap() {
		return nil
	}
	for _, f := range w.ownedOfKind(KindStation) {
		m := w.machine(f.ID)
		if m.Output != nil && !w.reserved(f.ID) {
			m.ReservedBy = wk.ID
			return &Job{Kind: "collect", Label: "plating " + RecipeByID(m.Output.Sub).Name, Machine: f.ID,
				Steps: []Step{w.stepTo(f.ID, 2, "take_output"), w.stepTo("pantry", 0, "drop_pantry")}}
		}
	}
	return nil
}

// jobDispose matches the oldest bag on the back step with something that can
// take it: a disposal machine that accepts the kind, a plot with room, or
// (for a car) the van or a long walk.
func (w *World) jobDispose(wk *Worker) *Job {
	for _, e := range w.Pile {
		if w.targeted(e.ID) {
			continue
		}
		for _, f := range w.ownedOfKind(KindDisposal) {
			m := w.machine(f.ID)
			if m.Busy > 0 || w.reserved(f.ID) || m.Off || !contains(f.Accepts, e.Sub) {
				continue
			}
			m.ReservedBy = wk.ID
			return &Job{Kind: "dispose", Label: "bagging " + e.Sub + " for the " + f.Name, Target: e.ID, Machine: f.ID,
				Steps: []Step{w.stepTo("back_step", 0, "take_evidence"), w.stepTo(f.ID, 3, "load_disposal")}}
		}
		if w.Buried < w.PlotCap() {
			for _, f := range w.ownedOfKind(KindPlot) {
				if w.reserved(f.ID) {
					continue
				}
				work := f.Duration
				if w.Has("shed") {
					work /= 2
				}
				w.machine(f.ID).ReservedBy = wk.ID
				return &Job{Kind: "bury", Label: "burying " + e.Sub, Target: e.ID, Machine: f.ID,
					Steps: []Step{w.stepTo("back_step", 0, "take_evidence"), w.stepTo(f.ID, work, "bury")}}
			}
		}
	}
	for _, c := range w.Driveway {
		if w.targeted(c.ID) {
			continue
		}
		work, label := 90, "walking a car to the mall"
		if w.Has("van") {
			work, label = 60, "hauling a car in the van"
		}
		return &Job{Kind: "haul", Label: label, Target: c.ID,
			Steps: []Step{{Dest: Pos{Ground, DrivewayX, SidewalkY}, Work: work, Do: "haul_car"}}}
	}
	return nil
}

func (w *World) jobButcher(wk *Worker) *Job {
	if !w.Has("block") || w.reserved("block") {
		return nil
	}
	fresh, _ := w.fridgeUsed()
	if w.FridgeCap()-fresh < 4 && !(w.Policy.Freeze && w.FreezerCap()-w.frozenCount() >= 4) {
		return nil
	}
	if len(w.Livestock) <= w.Policy.KeepLivestock && !w.hasOrder("butcher") {
		return nil
	}
	full := w.freeCell() == ""
	var pick *Livestock
	for _, l := range w.Livestock {
		if l.Cell == "" || w.targeted(l.ID) {
			continue
		}
		if l.Order == "butcher" {
			pick = l
			break
		}
		if l.Order != "" {
			continue
		}
		ready := int(l.Marbling) >= w.Policy.ButcherMarbling || (full && w.Policy.ButcherWhenFull) || l.Hunger >= 100
		if !ready {
			continue
		}
		if pick == nil || l.Marbling > pick.Marbling {
			pick = l
		}
	}
	if pick == nil {
		return nil
	}
	work := FixtureByID("block").Duration
	if w.Has("knife_set") {
		work = work * 3 / 4
	}
	w.machine("block").ReservedBy = wk.ID
	return &Job{Kind: "butcher", Label: "processing " + pick.Name, Target: pick.ID, Machine: "block",
		Steps: []Step{w.stepTo(pick.Cell, 2, "take_livestock"), w.stepTo("block", work, "butcher"),
			w.stepTo("fridge", 0, "drop_cuts"), w.stepTo("back_step", 0, "drop_bag")}}
}

func (w *World) frozenCount() int {
	_, n := w.fridgeUsed()
	return n
}

func (w *World) jobCookAny(wk *Worker) *Job { return w.jobCook(wk, false) }

// jobCook picks a recipe: food for the shelf until the pantry target is met,
// then sellable recipes for the stand. desperate skips the target check.
func (w *World) jobCook(wk *Worker, desperate bool) *Job {
	if len(w.Pantry) >= w.PantryCap() {
		return nil
	}
	needFood := w.foodOnShelf() < w.Policy.PantryTarget || desperate
	var cands []*Recipe
	for i := range Recipes {
		rc := &Recipes[i]
		if !w.Has(rc.Station) || w.machine(rc.Station).Busy > 0 || w.reserved(rc.Station) || w.machine(rc.Station).Output != nil || w.machine(rc.Station).Off {
			continue
		}
		if !w.hasNeeds(rc) {
			continue
		}
		if needFood && !rc.Food {
			continue
		}
		if !needFood && !w.Policy.Sell[rc.ID] {
			continue
		}
		if rc.Bones {
			if !w.hasPileKind("bones") {
				continue
			}
		} else if w.pickCuts(rc) == nil {
			continue
		}
		cands = append(cands, rc)
	}
	if len(cands) == 0 {
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if needFood {
			// quick food when hungry, best food otherwise
			if desperate {
				return cands[i].Duration < cands[j].Duration
			}
			return cands[i].Base > cands[j].Base
		}
		return cands[i].Price > cands[j].Price
	})
	rc := cands[0]
	w.machine(rc.Station).ReservedBy = wk.ID
	src := "fridge"
	take := "take_cuts"
	if rc.Bones {
		src, take = "back_step", "take_bones"
	}
	return &Job{Kind: "cook", Label: "cooking " + rc.Name, Recipe: rc.ID, Machine: rc.Station,
		Steps: []Step{w.stepTo(src, 0, take), w.stepTo(rc.Station, 10, "start_cook")}}
}

func (w *World) hasNeeds(rc *Recipe) bool {
	for _, n := range rc.Needs {
		if !w.Has(n) {
			return false
		}
	}
	return true
}

func (w *World) hasPileKind(kind string) bool {
	for _, e := range w.Pile {
		if e.Sub == kind && !w.targeted(e.ID) {
			return true
		}
	}
	return false
}

// pickCuts chooses fridge indexes for a recipe: unfrozen first, closest to
// spoiling first, then the best marbling. nil when the fridge can't cover it.
func (w *World) pickCuts(rc *Recipe) []int {
	need := map[string]int{}
	for k, n := range rc.Cuts {
		need[k] = n
	}
	used := map[int]bool{}
	var picked []int
	for k, n := range need {
		for n > 0 {
			best := -1
			for i, c := range w.Fridge {
				if used[i] || c.Kind != "cut" || c.Sub != k {
					continue
				}
				if best < 0 || w.betterCut(c, w.Fridge[best]) {
					best = i
				}
			}
			if best < 0 {
				return nil
			}
			picked = append(picked, best)
			used[best] = true
			n--
		}
	}
	return picked
}

func (w *World) betterCut(a, b Item) bool {
	if a.Frozen != b.Frozen {
		return !a.Frozen
	}
	if !a.Frozen && a.Born != b.Born {
		return a.Born < b.Born
	}
	return a.Marbling > b.Marbling
}

// standID is whichever stand is owned (the folding table or its upgrade).
func (w *World) standID() string {
	for _, f := range w.ownedOfKind(KindStand) {
		return f.ID
	}
	return ""
}

func (w *World) jobStock(wk *Worker) *Job {
	stand := w.standID()
	if stand == "" || len(w.Stand) >= w.StandCap() {
		return nil
	}
	spare := w.foodOnShelf() - w.Policy.PantryTarget
	for _, d := range w.Pantry {
		rc := RecipeByID(d.Sub)
		if rc.Price == 0 || !w.Policy.Sell[rc.ID] {
			continue
		}
		if rc.Food && spare <= 0 {
			continue
		}
		return &Job{Kind: "stock", Label: "stocking the stand", Target: d.ID,
			Steps: []Step{w.stepTo("pantry", 0, "take_sell"), w.stepTo(stand, 2, "drop_stand")}}
	}
	return nil
}

func (w *World) jobBrainwash(wk *Worker) *Job {
	if !w.Has("chair") || (!w.Policy.Brainwash && !w.hasOrder("chair")) {
		return nil
	}
	m := w.machine("chair")
	if m.Busy > 0 || w.reserved("chair") || m.Load != 0 || m.Off || w.Renfields() >= w.CotCount() {
		return nil
	}
	var pick *Livestock
	for _, l := range w.Livestock {
		if l.Cell == "" || w.targeted(l.ID) {
			continue
		}
		if l.Order == "chair" {
			pick = l
			break
		}
		if l.Order != "" || !w.Policy.Brainwash {
			continue
		}
		if pick == nil || VictimByID(l.Victim).Fight < VictimByID(pick.Victim).Fight {
			pick = l
		}
	}
	if pick == nil {
		return nil
	}
	m.ReservedBy = wk.ID
	return &Job{Kind: "brainwash", Label: "walking " + pick.Name + " to The Chair", Target: pick.ID, Machine: "chair",
		Steps: []Step{w.stepTo(pick.Cell, 2, "take_livestock"), w.stepTo("chair", 5, "load_chair")}}
}

func (w *World) hasOrder(order string) bool {
	for _, l := range w.Livestock {
		if l.Order == order && l.Cell != "" {
			return true
		}
	}
	return false
}

func (w *World) jobIdle(wk *Worker) *Job {
	r := w.rng()
	if wk.Kind == "convert" && wk.Role == RoleHand && w.IsNight() && w.freeCell() != "" && wk.Hunger > 30 && r.Intn(100) < 8 {
		w.autoHunt(wk)
		return nil
	}
	if r.Intn(3) > 0 {
		return &Job{Kind: "idle", Label: "standing around", Steps: []Step{{Dest: wk.Pos, Work: 10 + r.Intn(20), Do: "noop"}}}
	}
	p := Plans[wk.Pos.L]
	if wk.Role == RoleGuard || wk.Role == RoleGimp {
		// Guards patrol the cells; the gimp lives among them.
		cells := w.Cells()
		if len(cells) > 0 {
			c := cells[r.Intn(len(cells))]
			at := w.Pos(c.ID)
			label := "patrolling the cells"
			if wk.Role == RoleGimp {
				label = "keeping the livestock company"
			}
			return &Job{Kind: "idle", Label: label, Steps: []Step{{Dest: Pos{at.L, at.X, at.Y + 1}, Work: 20 + r.Intn(30), Do: "noop"}}}
		}
	}
	if wk.Role == RoleShill {
		at := w.TrapPos()
		return &Job{Kind: "idle", Label: "working the porch", Steps: []Step{{Dest: Pos{at.L, at.X - 1, at.Y}, Work: 30 + r.Intn(30), Do: "noop"}}}
	}
	why := "idle: " + w.idleReason()
	for i := 0; i < 20; i++ {
		x, y := wk.Pos.X+r.Intn(13)-6, wk.Pos.Y+r.Intn(7)-3
		if p.Floor(x, y) && w.FixtureAt(Pos{wk.Pos.L, x, y}) == nil {
			return &Job{Kind: "idle", Label: why, Steps: []Step{{Dest: Pos{wk.Pos.L, x, y}, Work: 5, Do: "noop"}}}
		}
	}
	return &Job{Kind: "idle", Label: why, Steps: []Step{{Dest: wk.Pos, Work: 10, Do: "noop"}}}
}

// idleReason is the most useful sentence about why there's no work: what the
// player would change to make some.
func (w *World) idleReason() string {
	fresh, _ := w.fridgeUsed()
	switch {
	case len(w.Livestock) > 0 && w.FridgeCap()-fresh < 4 && !(w.Policy.Freeze && w.FreezerCap()-w.frozenCount() >= 4):
		return "fridge full, can't butcher"
	case len(w.Livestock) > 0 && !w.anyReady():
		return fmt.Sprintf("waiting for livestock to fatten to %d", w.Policy.ButcherMarbling)
	case len(w.Pantry) >= w.PantryCap():
		return "shelf full"
	case w.foodOnShelf() >= w.Policy.PantryTarget && fresh > 0 && w.standID() != "" && len(w.Stand) >= w.StandCap():
		return "stand full, shelf stocked"
	case w.foodOnShelf() >= w.Policy.PantryTarget && fresh > 0:
		return "shelf stocked; nothing marked for sale needs cooking"
	case fresh == 0 && len(w.Livestock) == 0:
		return "nothing in the fridge, nobody in the cells"
	case fresh == 0:
		return "fridge empty"
	}
	return "nothing needs doing"
}

// anyReady reports whether policy would butcher anyone right now.
func (w *World) anyReady() bool {
	full := w.freeCell() == ""
	for _, l := range w.Livestock {
		if l.Cell == "" {
			continue
		}
		if l.Order == "butcher" || int(l.Marbling) >= w.Policy.ButcherMarbling || (full && w.Policy.ButcherWhenFull) || l.Hunger >= 100 {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// step actions
// ---------------------------------------------------------------------------

// do performs a step's action. Returning false abandons the job (the world
// changed under it); items the worker carries are dropped sensibly.
func (w *World) do(wk *Worker, j *Job, s Step) bool {
	switch s.Do {
	case "noop":
		return true
	case "sleep":
		wk.Asleep = true
		return true
	case "take_dish":
		for i, d := range w.Pantry {
			if d.ID == j.Target {
				wk.Carry = append(wk.Carry, d)
				w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
				return true
			}
		}
		return false
	case "eat":
		for i, d := range wk.Carry {
			if d.Kind == "dish" {
				rc := RecipeByID(d.Sub)
				wk.Carry = append(wk.Carry[:i], wk.Carry[i+1:]...)
				wk.Hunger = clampF(wk.Hunger+rc.Restore+5*float64(d.Quality-3), 0, 100)
				wk.Starving = 0
				if wk.Kind == "cannibal" {
					w.Stats.Meals++
					if d.Quality >= 3 {
						w.Stars += d.Quality - 2
					}
					w.say("food", "%s ate %s (%s).", wk.Name, rc.Name, stars(d.Quality))
					if d.Quality == 5 {
						w.post("good", "plated a five-star %s. The smell reached three houses.", rc.Name)
					}
				} else {
					wk.Loyalty = clampF(wk.Loyalty+2, 0, 100)
					wk.Meals++
					if wk.Kind == "renfield" && wk.Meals >= ConvertMeals && wk.Loyalty >= 40 {
						wk.Kind = "convert"
						wk.Loyalty = 100
						w.say("good", "%s asked for seconds. %s knows what it is. %s is in. A fellow cannibal.", wk.Name, wk.Name, wk.Name)
						w.post("info", "'s roommate has started grilling on weeknights. Big smile. Doesn't blink much.")
					}
				}
				return true
			}
		}
		return false
	case "feed":
		l := w.livestockByID(j.Target)
		if l == nil || w.Cash < 4 {
			return false
		}
		w.Cash -= 4
		w.SpentFeed += 4
		w.FedCount++
		l.Hunger = 0
		l.Starve = 0
		w.say("cash", "%s fed %s: $4 of gruel.", wk.Name, l.Name)
		return true
	case "take_output":
		m := w.machine(j.Machine)
		m.ReservedBy = 0
		if m.Output == nil {
			return false
		}
		wk.Carry = append(wk.Carry, *m.Output)
		m.Output = nil
		return true
	case "drop_pantry":
		for len(wk.Carry) > 0 {
			d := wk.Carry[0]
			wk.Carry = wk.Carry[1:]
			if len(w.Pantry) < w.PantryCap() {
				w.Pantry = append(w.Pantry, d)
			} else {
				w.say("warn", "The pantry is full. %s left the %s on the counter to rot.", wk.Name, RecipeByID(d.Sub).Name)
			}
		}
		return true
	case "take_evidence", "take_bones":
		want := j.Target
		for i, e := range w.Pile {
			if (s.Do == "take_evidence" && e.ID == want) || (s.Do == "take_bones" && e.Sub == "bones") {
				wk.Carry = append(wk.Carry, e)
				w.Pile = append(w.Pile[:i], w.Pile[i+1:]...)
				return true
			}
		}
		w.unreserve(j.Machine)
		return false
	case "load_disposal":
		m := w.machine(j.Machine)
		m.ReservedBy = 0
		f := FixtureByID(j.Machine)
		ei := -1
		for i, c := range wk.Carry {
			if c.Kind == "evidence" && contains(f.Accepts, c.Sub) {
				ei = i
				break
			}
		}
		if m.Busy > 0 || ei < 0 {
			w.dropBack(wk)
			return false
		}
		e := wk.Carry[ei]
		wk.Carry = append(wk.Carry[:ei], wk.Carry[ei+1:]...)
		m.Busy = f.Duration
		m.Item = e.Sub
		m.Uses++
		if f.Noise > 0 {
			w.Heat = clampF(w.Heat+f.Noise, 0, 120)
			w.say("heat", "The %s roared through a bag of %s. The street heard it.", f.Name, e.Sub)
		}
		return true
	case "bury":
		m := w.machine(j.Machine)
		m.ReservedBy = 0
		ei := -1
		for i, c := range wk.Carry {
			if c.Kind == "evidence" {
				ei = i
				break
			}
		}
		if ei < 0 || w.Buried >= w.PlotCap() {
			w.dropBack(wk)
			return false
		}
		wk.Carry = append(wk.Carry[:ei], wk.Carry[ei+1:]...)
		w.Buried++
		w.Stats.Buried++
		if !w.Has("shed") && w.IsNight() {
			w.Heat = clampF(w.Heat+0.5, 0, 120)
		}
		return true
	case "haul_car":
		for i, c := range w.Driveway {
			if c.ID == j.Target {
				w.Driveway = append(w.Driveway[:i], w.Driveway[i+1:]...)
				if !w.Has("van") && w.rng().Intn(100) < 30 {
					w.Heat = clampF(w.Heat+8, 0, 120)
					w.say("bad", "A traffic cam caught %s walking away from the car. Heat +8.", wk.Name)
				} else {
					w.say("good", "The car is somebody else's problem now.")
				}
				return true
			}
		}
		return false
	case "take_livestock":
		l := w.livestockByID(j.Target)
		if l == nil || l.Cell == "" {
			w.unreserve(j.Machine)
			return false
		}
		l.Cell = ""
		wk.Carry = append(wk.Carry, Item{ID: w.newID(), Kind: "livestock", Ref: l.ID, Victim: l.Victim})
		return true
	case "butcher":
		m := w.machine("block")
		m.ReservedBy = 0
		li := -1
		for i, c := range wk.Carry {
			if c.Kind == "livestock" {
				li = i
			}
		}
		if li < 0 {
			return false
		}
		l := w.livestockByID(wk.Carry[li].Ref)
		wk.Carry = append(wk.Carry[:li], wk.Carry[li+1:]...)
		if l == nil {
			return false
		}
		w.removeLivestock(l.ID)
		w.Stats.Butchered++
		m.Uses++
		v := VictimByID(l.Victim)
		marb := int(l.Marbling + 0.5)
		prime := 2
		if w.Has("knife_set") {
			prime++
		}
		scraps := 1
		if marb >= 4 {
			scraps++
		}
		for i := 0; i < prime; i++ {
			wk.Carry = append(wk.Carry, Item{ID: w.newID(), Kind: "cut", Sub: CutPrime, Victim: v.ID, Marbling: marb, Born: w.Clock})
		}
		wk.Carry = append(wk.Carry, Item{ID: w.newID(), Kind: "cut", Sub: CutOffal, Victim: v.ID, Marbling: marb, Born: w.Clock})
		for i := 0; i < scraps; i++ {
			wk.Carry = append(wk.Carry, Item{ID: w.newID(), Kind: "cut", Sub: CutScraps, Victim: v.ID, Marbling: marb, Born: w.Clock})
		}
		for _, k := range []string{"bones", "clothes", "phone"} {
			wk.Carry = append(wk.Carry, Item{ID: w.newID(), Kind: "evidence", Sub: k, Victim: v.ID, Born: w.Clock})
		}
		w.say("info", "%s processed %s the %s. Hum something.", wk.Name, l.Name, v.Name)
		return true
	case "drop_cuts":
		var keep []Item
		wasted := 0
		for _, c := range wk.Carry {
			if c.Kind != "cut" {
				keep = append(keep, c)
				continue
			}
			if !w.storeCut(c) {
				wasted++
			}
		}
		wk.Carry = keep
		if wasted > 0 {
			w.Heat = clampF(w.Heat+float64(wasted), 0, 120)
			w.say("bad", "%d cuts wouldn't fit anywhere cold. Into the trash; the trash smells.", wasted)
		}
		return true
	case "drop_bag":
		var keep []Item
		for _, c := range wk.Carry {
			if c.Kind == "evidence" {
				w.Pile = append(w.Pile, c)
			} else {
				keep = append(keep, c)
			}
		}
		wk.Carry = keep
		return true
	case "take_cuts":
		rc := RecipeByID(j.Recipe)
		idx := w.pickCuts(rc)
		if idx == nil {
			w.unreserve(j.Machine)
			return false
		}
		sort.Sort(sort.Reverse(sort.IntSlice(idx)))
		for _, i := range idx {
			wk.Carry = append(wk.Carry, w.Fridge[i])
			w.Fridge = append(w.Fridge[:i], w.Fridge[i+1:]...)
		}
		return true
	case "start_cook":
		m := w.machine(j.Machine)
		m.ReservedBy = 0
		rc := RecipeByID(j.Recipe)
		if m.Busy > 0 || m.Output != nil {
			w.dropBack(wk)
			return false
		}
		q := rc.Base
		marb, worst, frozen, n := 0, 0, false, 0
		var keep []Item
		for _, c := range wk.Carry {
			if c.Kind == "cut" || c.Kind == "evidence" {
				if c.Kind == "cut" {
					marb += c.Marbling
					n++
					age := int(w.Clock-c.Born) / day
					if age > worst {
						worst = age
					}
					if c.Frozen {
						frozen = true
					}
				}
				continue
			}
			keep = append(keep, c)
		}
		wk.Carry = keep
		if n > 0 {
			marb /= n
			switch {
			case marb <= 2:
				q--
			case marb >= 4:
				q++
			}
			if worst >= 2 {
				q--
			}
			if frozen {
				q--
			}
		}
		m.Busy = rc.Duration
		m.Recipe = rc.ID
		m.Uses++
		m.Output = &Item{ID: w.newID(), Kind: "dish", Sub: rc.ID, Quality: clamp(q, 1, 5)}
		if rc.Bones {
			w.Stats.Disposed++
		}
		return true
	case "take_sell":
		for i, d := range w.Pantry {
			if d.ID == j.Target {
				wk.Carry = append(wk.Carry, d)
				w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
				return true
			}
		}
		return false
	case "drop_stand":
		var keep []Item
		for _, d := range wk.Carry {
			if d.Kind == "dish" && len(w.Stand) < w.StandCap() {
				w.Stand = append(w.Stand, d)
			} else {
				keep = append(keep, d)
			}
		}
		wk.Carry = keep
		w.dropBack(wk)
		return true
	case "load_chair":
		m := w.machine("chair")
		m.ReservedBy = 0
		li := -1
		for i, c := range wk.Carry {
			if c.Kind == "livestock" {
				li = i
			}
		}
		if li < 0 || m.Busy > 0 || m.Load != 0 {
			w.dropBack(wk)
			return false
		}
		l := w.livestockByID(wk.Carry[li].Ref)
		wk.Carry = append(wk.Carry[:li], wk.Carry[li+1:]...)
		if l == nil {
			return false
		}
		m.Load = l.ID
		m.Busy = FixtureByID("chair").Duration
		if w.Has("vhs") {
			m.Busy /= 2
		}
		m.Uses++
		w.say("info", "%s is strapped into The Chair. Tape one: 'You Belong Here'.", l.Name)
		return true
	}
	return true
}

func (w *World) unreserve(id string) {
	if id != "" {
		w.machine(id).ReservedBy = 0
	}
}

// dropBack returns whatever a worker holds to where it belongs: cuts to the
// fridge, dishes to the shelf, evidence to the step, livestock to a cell.
func (w *World) dropBack(wk *Worker) {
	for _, c := range wk.Carry {
		switch c.Kind {
		case "cut":
			if !w.storeCut(c) {
				w.say("bad", "A cut went in the trash. The trash smells.")
			}
		case "dish":
			if len(w.Pantry) < w.PantryCap() {
				w.Pantry = append(w.Pantry, c)
			}
		case "evidence":
			w.Pile = append(w.Pile, c)
		case "livestock":
			if l := w.livestockByID(c.Ref); l != nil {
				if cell := w.freeCell(); cell != "" {
					l.Cell = cell
				} else {
					w.say("bad", "%s had nowhere to put %s. Straight to the block, then.", wk.Name, l.Name)
					l.Cell = ""
					w.removeLivestock(l.ID)
					v := VictimByID(l.Victim)
					marb := int(l.Marbling + 0.5)
					w.storeCut(Item{ID: w.newID(), Kind: "cut", Sub: CutPrime, Victim: v.ID, Marbling: marb, Born: w.Clock})
					w.storeCut(Item{ID: w.newID(), Kind: "cut", Sub: CutPrime, Victim: v.ID, Marbling: marb, Born: w.Clock})
					w.storeCut(Item{ID: w.newID(), Kind: "cut", Sub: CutScraps, Victim: v.ID, Marbling: marb, Born: w.Clock})
					for _, k := range []string{"bones", "clothes", "phone"} {
						w.Pile = append(w.Pile, Item{ID: w.newID(), Kind: "evidence", Sub: k, Victim: v.ID, Born: w.Clock})
					}
				}
			}
		}
	}
	wk.Carry = nil
}

// storeCut puts a cut in the fridge, overflowing into the freezer (frozen)
// when policy allows. False when there's no cold space at all.
func (w *World) storeCut(c Item) bool {
	fresh, frozen := w.fridgeUsed()
	if fresh < w.FridgeCap() {
		c.Frozen = false
		w.Fridge = append(w.Fridge, c)
		return true
	}
	if w.Policy.Freeze && frozen < w.FreezerCap() {
		c.Frozen = true
		w.Fridge = append(w.Fridge, c)
		return true
	}
	return false
}

func (w *World) removeLivestock(id int) {
	for i, l := range w.Livestock {
		if l.ID == id {
			w.Livestock = append(w.Livestock[:i], w.Livestock[i+1:]...)
			return
		}
	}
}

func stars(q int) string {
	s := ""
	for i := 0; i < 5; i++ {
		if i < q {
			s += "*"
		} else {
			s += "-"
		}
	}
	return s
}

// JobLabel is what the sidebar shows for a worker.
func (wk *Worker) JobLabel() string {
	if wk.Asleep {
		return "asleep"
	}
	if wk.Job == nil {
		return "idle"
	}
	return wk.Job.Label
}

func (w *World) Describe(wk *Worker) string {
	return fmt.Sprintf("%s: %s", wk.Name, wk.JobLabel())
}
