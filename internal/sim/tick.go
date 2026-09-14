package sim

import (
	"math/rand"
	"strings"
	"time"
)

// Advance runs n game minutes. Everything the house does happens here, in the
// same order every tick, watched or not.
func (w *World) Advance(n int) {
	for i := 0; i < n && w.Alive; i++ {
		w.tick()
	}
}

// Catchup advances the world by the wall-clock time since the last sync, at
// the world's scale. A compute guard stops a save from years ago from
// simulating for minutes; it is not a rule of the game.
func (w *World) Catchup(now time.Time) int { return w.Sync(now, 1) }

// Sync is Catchup with a speed multiplier: a connected door fast-forwarding
// (2, 4) or paused (0). Whole seconds only; a sub-second call accumulates.
func (w *World) Sync(now time.Time, speed int) int {
	if w.LastWall == 0 {
		w.LastWall = now.Unix()
		return 0
	}
	elapsed := now.Unix() - w.LastWall
	if elapsed <= 0 {
		return 0
	}
	if speed <= 0 {
		w.LastWall = now.Unix()
		return 0
	}
	ticks := int(elapsed * int64(w.Scale) * int64(speed) / 60)
	if ticks <= 0 {
		return 0
	}
	w.LastWall = now.Unix()
	if ticks > 180*day {
		ticks = 180 * day
	}
	w.Advance(ticks)
	return ticks
}

func (w *World) tick() {
	w.Clock++
	w.tickPassers()
	w.tickLivestock()
	w.tickMachines()
	w.tickWorkers()
	w.tickVisitors()
	if w.Heat >= 100 && !w.RaidCalled {
		w.RaidCalled = true
		for i := 0; i < 3; i++ {
			w.sendVisitor("police", 5, "the FBI")
		}
		w.say("bad", "Sirens. Several. Getting closer.")
	}
	if w.Clock%60 == 0 {
		w.hourly()
	}
	if w.Clock%day == 6*60 {
		w.daily()
	}
	w.checkDeath()
}

// ---------------------------------------------------------------------------
// the sidewalk
// ---------------------------------------------------------------------------

func (w *World) trafficInterval() int {
	h := w.Hour()
	switch {
	case h >= 8 && h < 20:
		return 9
	case h >= 20 || h < 1:
		return 18
	}
	return 40
}

func (w *World) rollVictim(r *rand.Rand) *Victim {
	night := w.IsNight()
	total := 0
	var pool []*Victim
	var weights []int
	for i := range Victims {
		v := &Victims[i]
		if v.Night && !night {
			continue
		}
		wgt := v.Weight
		if night && !v.Night {
			wgt /= 2
		}
		if v.ID == "podcaster" && w.Tier() < 3 {
			continue
		}
		if v.ID == "hoa" && w.Tier() < 1 {
			continue
		}
		for _, l := range v.Lures {
			if w.Has(l) {
				wgt += 5
			}
		}
		if wgt <= 0 {
			continue
		}
		pool = append(pool, v)
		weights = append(weights, wgt)
		total += wgt
	}
	x := r.Intn(total)
	for i, v := range pool {
		x -= weights[i]
		if x < 0 {
			return v
		}
	}
	return pool[0]
}

func (w *World) tickPassers() {
	r := w.rng()
	if w.Clock >= w.NextPasserby {
		v := w.rollVictim(r)
		dir := 1
		x := SidewalkEnds[0]
		if r.Intn(2) == 0 {
			dir, x = -1, SidewalkEnds[1]
		}
		w.Passers = append(w.Passers, &Passerby{ID: w.newID(), Victim: v.ID, Pos: Pos{Ground, x, SidewalkY}, Dir: dir, State: "walk", Pair: v.Pair})
		iv := w.trafficInterval()
		w.NextPasserby = w.Clock + int64(iv/2+r.Intn(iv))
	}
	var keep []*Passerby
	for _, p := range w.Passers {
		switch p.State {
		case "walk":
			if p.Pos.X == GateTile.X && w.tryLure(p, r) {
				p.State = "lured"
				p.Path = Path(p.Pos, w.TrapPos())
				keep = append(keep, p)
				continue
			}
			p.Pos.X += p.Dir
			if p.Pos.X < SidewalkEnds[0] || p.Pos.X > SidewalkEnds[1] {
				continue // walked on by
			}
		case "lured":
			if len(p.Path) == 0 {
				w.springTrap(p, r)
				if p.State != "flee" {
					continue
				}
			} else {
				p.Pos = p.Path[0]
				p.Path = p.Path[1:]
			}
		case "flee":
			if len(p.Path) == 0 {
				continue
			}
			p.Pos = p.Path[0]
			p.Path = p.Path[1:]
		}
		keep = append(keep, p)
	}
	w.Passers = keep
}

// tryLure rolls whether a passer-by at the gate turns onto the porch. The
// house needs a free cell and an idle trap, or the mat stays a mat.
func (w *World) tryLure(p *Passerby, r *rand.Rand) bool {
	if w.freeCell() == "" || w.trapBusy() || w.Clock < w.TrapReady {
		return false
	}
	v := VictimByID(p.Victim)
	chance := 15 // per mille
	for _, f := range w.ownedOfKind(KindLure) {
		if contains(v.Lures, f.ID) {
			chance += 30
		} else {
			chance += 6
		}
	}
	if w.Has("doorbell_cam") {
		chance -= 5
	}
	chance += 15 * w.RoleCount(RoleShill)
	return r.Intn(1000) < clamp(chance, 5, 250)
}

func (w *World) trapBusy() bool {
	for _, p := range w.Passers {
		if p.State == "lured" {
			return true
		}
	}
	return false
}

func (w *World) trapChance(v *Victim) int {
	c := 45
	if w.Has("trapdoor_pro") {
		c = 70
	}
	if w.Has("chemistry") {
		c += 12
	}
	if w.Has("telescope") && w.IsNight() {
		c += 8
	}
	if w.Has("gnome") {
		c += 3
	}
	return clamp(c-v.Fight*8, 10, 95)
}

func (w *World) springTrap(p *Passerby, r *rand.Rand) {
	v := VictimByID(p.Victim)
	if r.Intn(100) >= w.trapChance(v) {
		p.State = "flee"
		end := SidewalkEnds[0]
		if r.Intn(2) == 0 {
			end = SidewalkEnds[1]
		}
		p.Path = Path(p.Pos, Pos{Ground, end, SidewalkY})
		w.Stats.Fled++
		w.TrapReady = w.Clock + 120
		if r.Intn(100) < 50 {
			w.say("warn", "The mat gave a little under the %s. They frowned at it and left.", strings.ToLower(v.Name))
			return
		}
		w.Heat = clampF(w.Heat+1+float64(v.Fight), 0, 120)
		w.say("bad", "The mat didn't take. The %s ran off yelling. Heat +%d.", strings.ToLower(v.Name), 1+v.Fight)
		return
	}
	w.TrapReady = w.Clock + 60
	w.capture(v, r)
	more := 0
	if p.Pair {
		more = 1
	}
	if v.Group > 1 {
		more = v.Group - 1
	}
	for i := 0; i < more; i++ {
		if w.freeCell() != "" {
			if i == 0 && p.Pair {
				w.say("good", "There were two of them. There are always two of them.")
			}
			w.capture(v, r)
		} else {
			w.Heat = clampF(w.Heat+4, 0, 120)
			w.say("bad", "The rest of the %s saw the drop and sprinted. Heat +4.", strings.ToLower(v.Name))
			break
		}
	}
	if v.Drives {
		w.Driveway = append(w.Driveway, Item{ID: w.newID(), Kind: "evidence", Sub: "car", Victim: v.ID, Born: w.Clock})
		w.say("heat", "Their car is parked out front. That's evidence with a license plate.")
	}
}

func (w *World) capture(v *Victim, r *rand.Rand) {
	cell := w.freeCell()
	if cell == "" {
		return
	}
	name := captiveNames[r.Intn(len(captiveNames))]
	w.Livestock = append(w.Livestock, &Livestock{ID: w.newID(), Victim: v.ID, Name: name, Cell: cell,
		Marbling: float64(v.Marbling), Since: w.Clock})
	w.Missing = append(w.Missing, Missing{Victim: v.ID, Due: w.Clock + int64(v.NoticeDays*day)})
	w.Stats.Captured++
	w.Heat = clampF(w.Heat+1, 0, 120)
	w.say("good", "Thump. %s the %s is in %s. %s", name, v.Name, FixtureByID(cell).Name, v.Flavor)
	if v.Loot > 0 {
		w.Cash += float64(v.Loot)
		w.Stats.Looted += v.Loot
		w.say("cash", "Their wallet: $%d.", v.Loot)
	}
}

// ---------------------------------------------------------------------------
// the basement
// ---------------------------------------------------------------------------

func (w *World) tickLivestock() {
	fatten := 1.0 / (3 * day)
	if w.Has("buffet") {
		fatten = 1.0 / (2 * day)
	}
	var keep []*Livestock
	for _, l := range w.Livestock {
		l.Hunger = clampF(l.Hunger+100.0/(2*day), 0, 100)
		if l.Cell != "" && w.Has("gruel") && l.Hunger > 50 && w.Cash >= 2 {
			w.Cash -= 2
			l.Hunger = 0
		}
		if l.Hunger < 50 && l.Marbling < 5 {
			l.Marbling += fatten
		}
		if l.Hunger >= 100 {
			l.Starve++
			if l.Starve > day {
				w.say("bad", "%s starved in the cell. Wasteful. Bones and clothes on the step.", l.Name)
				w.Pile = append(w.Pile,
					Item{ID: w.newID(), Kind: "evidence", Sub: "bones", Victim: l.Victim, Born: w.Clock},
					Item{ID: w.newID(), Kind: "evidence", Sub: "clothes", Victim: l.Victim, Born: w.Clock})
				w.Heat = clampF(w.Heat+2, 0, 120)
				continue
			}
		} else {
			l.Starve = 0
		}
		keep = append(keep, l)
	}
	w.Livestock = keep
}

// ---------------------------------------------------------------------------
// machines
// ---------------------------------------------------------------------------

func (w *World) tickMachines() {
	for id, m := range w.Machines {
		if m.Busy <= 0 {
			continue
		}
		m.Busy--
		if m.Busy > 0 {
			continue
		}
		f := FixtureByID(id)
		switch f.Kind {
		case KindStation:
			// Output waits for a worker to plate it (jobCollect).
			if m.Output != nil {
				w.say("food", "%s is done at the %s.", RecipeByID(m.Output.Sub).Name, f.Name)
			}
		case KindDisposal:
			w.Stats.Disposed++
			if id == "compost" {
				w.Compost++
			}
			m.Item = ""
		case KindChair:
			l := w.livestockByID(m.Load)
			m.Load = 0
			if l == nil {
				continue
			}
			w.removeLivestock(l.ID)
			if w.Renfields() < w.CotCount() {
				w.Workers = append(w.Workers, &Worker{ID: w.newID(), Name: l.Name, Kind: "renfield", Victim: l.Victim,
					Pos: w.Pos("chair"), Hunger: 60, Energy: 60, Loyalty: 70})
				w.say("good", "%s blinks twice and asks what needs doing. A Renfield.", l.Name)
				w.post("info", "has a new 'roommate'. Very quiet. Very helpful.")
			} else if cell := w.freeCell(); cell != "" {
				l.Cell = cell
				w.Livestock = append(w.Livestock, l)
				w.say("warn", "No cot for %s. Back to the cell, mind wiped for nothing.", l.Name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// workers
// ---------------------------------------------------------------------------

func (w *World) tickWorkers() {
	var keep []*Worker
	for _, wk := range w.Workers {
		wk.Hunger = clampF(wk.Hunger-100.0/(3*day), 0, 100)
		if wk.Asleep {
			wk.Energy = clampF(wk.Energy+100.0/(8*60), 0, 100)
			if wk.Energy >= 100 {
				wk.Asleep = false
				wk.Job = nil
			}
		} else {
			wk.Energy = clampF(wk.Energy-100.0/(16*60), 0, 100)
		}
		if wk.Hunger <= 0 {
			wk.Starving++
		}
		if wk.Kind == "convert" {
			wk.Hunger = clampF(wk.Hunger-100.0/(6*day), 0, 100) // eats like family: half again as much
		}
		if wk.Kind == "renfield" {
			drop := 3.0 / day
			if w.Has("big_tv") {
				drop = 1.0 / day
			}
			wk.Loyalty -= drop
			if wk.Loyalty <= 0 {
				w.Heat = clampF(w.Heat+25, 0, 120)
				w.say("bad", "%s snapped out of it and walked to the police station in your bathrobe. Heat +25.", wk.Name)
				w.post("bad", "'s roommate has been giving interviews. Something about a slideshow.")
				w.dropBack(wk)
				w.unreserve(jobMachine(wk))
				continue
			}
		}
		if wk.Kind != "cannibal" {
			if wk.Starving > 2*day {
				w.say("bad", "%s starved. They need feeding too. Bones on the step.", wk.Name)
				w.Pile = append(w.Pile, Item{ID: w.newID(), Kind: "evidence", Sub: "bones", Victim: wk.Victim, Born: w.Clock})
				w.dropBack(wk)
				w.unreserve(jobMachine(wk))
				continue
			}
		}
		keep = append(keep, wk)
		if wk.Away != nil {
			if w.Clock >= wk.Away.Until {
				w.returnHome(wk)
			}
			continue
		}
		if wk.Asleep {
			continue
		}
		if wk.Job == nil {
			if w.Clock%3 == 0 {
				wk.Job = w.pickJob(wk)
				if wk.Job != nil {
					wk.Path = Path(wk.Pos, wk.Job.Steps[0].Dest)
					wk.Job.Timer = wk.Job.Steps[0].Work
				}
			}
			continue
		}
		j := wk.Job
		if j.Step >= len(j.Steps) {
			wk.Job = nil
			continue
		}
		s := j.Steps[j.Step]
		if wk.Pos != s.Dest {
			if len(wk.Path) == 0 {
				wk.Path = Path(wk.Pos, s.Dest)
				if len(wk.Path) == 0 {
					w.unreserve(jobMachine(wk))
					w.dropBack(wk)
					wk.Job = nil
					continue
				}
			}
			for i := 0; i < 2 && len(wk.Path) > 0; i++ {
				wk.Pos = wk.Path[0]
				wk.Path = wk.Path[1:]
			}
			continue
		}
		if j.Timer > 0 {
			j.Timer--
			continue
		}
		ok := w.do(wk, j, s)
		if !ok {
			w.dropBack(wk)
			wk.Job = nil
			continue
		}
		if wk.Asleep {
			continue
		}
		j.Step++
		if j.Step >= len(j.Steps) {
			wk.Job = nil
		} else {
			j.Timer = j.Steps[j.Step].Work
			wk.Path = nil
		}
	}
	w.Workers = keep
}

func jobMachine(wk *Worker) string {
	if wk.Job == nil {
		return ""
	}
	return wk.Job.Machine
}

// ---------------------------------------------------------------------------
// visitors: inspectors, the dog, the police
// ---------------------------------------------------------------------------

func (w *World) tickVisitors() {
	var keep []*Visitor
	for _, v := range w.Visitors {
		switch v.State {
		case "arrive":
			if len(v.Path) == 0 {
				v.State = "knock"
				v.Timer = 15
				if w.Has("doorbell_cam") && v.Kind == "inspector" {
					v.Timer = 30
				}
				if v.Kind == "police" {
					v.Timer = 2
				}
				if v.Kind == "dog" {
					v.Timer = 10
				}
			} else {
				v.Pos = v.Path[0]
				v.Path = v.Path[1:]
			}
		case "knock":
			v.Timer--
			if v.Timer > 0 {
				break
			}
			switch v.Kind {
			case "inspector":
				w.resolveInspection(v)
			case "dog":
				w.resolveDog(v)
			case "police":
				if !w.Alive {
					break
				}
				w.Alive = false
				w.Cause = "raided by " + TierName(5)
				w.say("bad", "Flashbangs. Boots. A tactical team in the kitchen admiring the knife set. %s is done.", w.Cannibal)
				w.post("bad", "'s house got raided at dawn. Fourteen vehicles. The HOA is 'reviewing the hedge situation'.")
			}
			v.State = "leave"
			end := SidewalkEnds[0]
			if v.Kind == "dog" {
				end = -1
			}
			if end >= 0 {
				v.Path = Path(v.Pos, Pos{Ground, end, SidewalkY})
			}
		case "leave":
			if len(v.Path) == 0 {
				continue
			}
			v.Pos = v.Path[0]
			v.Path = v.Path[1:]
		}
		keep = append(keep, v)
	}
	w.Visitors = keep
}

func (w *World) sendVisitor(kind string, tier int, name string) {
	start := Pos{Ground, SidewalkEnds[1], SidewalkY}
	dest := FrontDoor
	if kind == "dog" {
		start = Pos{Backyard, 44, 3}
		plots := w.ownedOfKind(KindPlot)
		dest = w.Pos(plots[w.rng().Intn(len(plots))].ID)
	}
	w.Visitors = append(w.Visitors, &Visitor{ID: w.newID(), Kind: kind, Tier: tier, Name: name, Pos: start,
		Path: Path(start, dest), State: "arrive"})
}

func (w *World) resolveInspection(v *Visitor) {
	w.Stats.Inspections++
	visible := w.VisibleEvidence()
	limit := InspectionThreshold(v.Tier)
	if visible <= limit {
		drop := 8.0
		for i, d := range w.Pantry {
			if RecipeByID(d.Sub).Food && d.Quality >= 3 {
				w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
				drop += 4
				w.say("good", "There was a %s on the shelf to offer. They left with a slice and a smile.", RecipeByID(d.Sub).Name)
				break
			}
		}
		w.Heat = clampF(w.Heat-drop, 0, 120)
		w.say("good", "Visit from %s. They saw a lovely home. Heat -%.0f.", v.Name, drop)
		return
	}
	rise := 8.0 + 2*float64(v.Tier)
	w.Heat = clampF(w.Heat+rise, 0, 120)
	w.say("bad", "Visit from %s. They saw too much (%d vs %d allowed). Heat +%.0f.", v.Name, visible, limit, rise)
	w.post("heat", "had visitors. They left quickly and made a phone call in the driveway.")
	if v.Tier >= 4 && len(w.Livestock) > 0 && w.rng().Intn(100) < 30 {
		l := w.Livestock[0]
		w.removeLivestock(l.ID)
		w.Heat = clampF(w.Heat+15, 0, 120)
		w.say("bad", "They took %s with them. Heat +15.", l.Name)
	}
}

func (w *World) resolveDog(v *Visitor) {
	if w.Buried > 0 && w.rng().Intn(100) < 25 {
		w.Buried--
		w.Pile = append(w.Pile, Item{ID: w.newID(), Kind: "evidence", Sub: "bones", Born: w.Clock})
		w.Heat = clampF(w.Heat+6, 0, 120)
		w.say("bad", "Gladys's labradoodle dug something up and paraded it down the street. Heat +6. It's back on the step.")
	}
}

// ---------------------------------------------------------------------------
// hourly and daily
// ---------------------------------------------------------------------------

func (w *World) hourly() {
	r := w.rng()
	// Evidence radiates.
	heat := 0.0
	for _, e := range w.Pile {
		switch e.Sub {
		case "bones":
			heat += 0.08
		case "clothes":
			heat += 0.04
		case "phone":
			if w.Clock-e.Born < 72*60 {
				heat += 0.25
			}
		}
	}
	heat += 0.3 * float64(len(w.Driveway))
	if !w.Has("soundproof") && w.RoleCount(RoleGimp) == 0 {
		for _, l := range w.Livestock {
			if l.Hunger > 50 {
				heat += 0.12
			} else {
				heat += 0.04
			}
		}
	}
	for _, f := range w.ownedOfKind(KindStation) {
		if f.Noise > 0 && w.machine(f.ID).Busy > 0 {
			heat += f.Noise
		}
	}
	// ...and cools.
	cool := 0.7
	if w.Has("fence") {
		cool += 0.2
	}
	if w.Has("curtains") {
		cool += 0.2
	}
	if w.VisibleEvidence() <= 0 {
		cool += 0.5
	}
	w.Heat = clampF(w.Heat+heat-cool, 0, 120)

	// Utilities.
	draw := 0.0
	for _, f := range Fixtures {
		if w.Fixtures[f.ID] {
			draw += f.Draw
		}
	}
	w.Cash -= draw
	if w.Cash < 0 {
		w.Cash = 0
	}

	// Spoilage.
	var keep []Item
	spoiled := 0
	for _, c := range w.Fridge {
		life := 3 * day
		if c.Sub == CutOffal {
			life = 2 * day
		}
		if !c.Frozen && w.Clock-c.Born > int64(life) {
			spoiled++
			continue
		}
		keep = append(keep, c)
	}
	w.Fridge = keep
	keep = nil
	for _, d := range w.Pantry {
		if w.Clock-d.Born > int64(RecipeByID(d.Sub).ShelfMin) && d.Born > 0 {
			spoiled++
			continue
		}
		keep = append(keep, d)
	}
	w.Pantry = keep
	if spoiled > 0 {
		w.Heat = clampF(w.Heat+0.3*float64(spoiled), 0, 120)
		w.say("bad", "%d thing%s went off. The trash smells like a crime.", spoiled, plural(spoiled))
	}
	for i := range w.Pantry {
		if w.Pantry[i].Born == 0 {
			w.Pantry[i].Born = w.Clock
		}
	}

	// The stand sells, nine to six.
	if h := w.Hour(); h >= 9 && h < 18 && len(w.Stand) > 0 {
		d := w.Stand[0]
		w.Stand = w.Stand[1:]
		rc := RecipeByID(d.Sub)
		price := rc.Price + 5*(d.Quality-3)
		w.Cash += float64(price)
		w.Stats.Sold++
		if w.Stats.Sold%10 == 0 {
			w.post("cash", "sold out at the curb stand again. The pies. People line up for the pies.")
		}
	}

	// Missing persons.
	var stillMissing []Missing
	for _, m := range w.Missing {
		if w.Clock >= m.Due {
			v := VictimByID(m.Victim)
			w.Heat = clampF(w.Heat+v.HeatMissing, 0, 120)
			if v.HeatMissing < 0 {
				w.say("good", "A %s was reported missing. Nobody seems upset. Heat %.0f.", strings.ToLower(v.Name), v.HeatMissing)
				w.post("info", "'s street: a %s has vanished. The comment section is celebrating.", strings.ToLower(v.Name))
			} else {
				w.say("heat", "A %s was reported missing. Heat +%.0f.", strings.ToLower(v.Name), v.HeatMissing)
				w.post("heat", "'s street: MISSING - %s, last seen near a house with a very nice hedge.", v.Name)
			}
			continue
		}
		stillMissing = append(stillMissing, m)
	}
	w.Missing = stillMissing

	// Escapes: an unwatched, hungry cell is a door being tried.
	for _, l := range w.Livestock {
		if l.Cell == "" {
			continue
		}
		pct := 0.08 + float64(VictimByID(l.Victim).Fight)*0.03
		if l.Hunger > 50 {
			pct += 0.3
		}
		if w.Has("cameras") {
			pct *= 0.2
		}
		if w.Has("soundproof") {
			pct -= 0.1
		}
		pct -= 0.1 * float64(w.Renfields())
		if w.RoleCount(RoleGuard) > 0 {
			pct /= 3
		}
		if w.RoleCount(RoleGimp) > 0 {
			pct /= 2
		}
		if r.Float64()*100 < pct {
			w.removeLivestock(l.ID)
			w.Stats.Escaped++
			w.Heat = clampF(w.Heat+20, 0, 120)
			w.say("bad", "%s the %s got the cell door open and ran screaming down the cul-de-sac. Heat +20.", l.Name, VictimByID(l.Victim).Name)
			w.post("bad", "had someone run out of their house at 4 A.M. wearing a bedsheet. \"Wine cellar\" my foot.")
			break
		}
	}

	// The street reacts to tiers, once per crossing, with a day between visits.
	if t := w.Tier(); t > w.InspectTier && t >= 1 && t <= 4 && !w.visitorOut("inspector") && w.Clock-w.LastInspect > 2*day {
		w.InspectTier = t
		w.LastInspect = w.Clock
		w.sendVisitor("inspector", t, inspectors[t])
		w.say("warn", "Heat tier: %s. A car just pulled up: %s.", TierName(t), inspectors[t])
	}
	if w.Tier() < w.InspectTier {
		w.InspectTier = w.Tier()
	}
	// The dog, sometimes.
	if !w.Has("fence") && !w.visitorOut("dog") && w.Buried > 0 && r.Intn(100) < 2 {
		w.sendVisitor("dog", 0, "Gladys's labradoodle")
	}
	// Ambient: an evening dinner party if the dining set and a good dish exist.
	if w.Has("dining_set") && w.Hour() == 19 {
		for i, d := range w.Pantry {
			if RecipeByID(d.Sub).Food && d.Quality >= 4 {
				w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
				drop := 5.0 * float64(d.Quality)
				if w.Has("wine_rack") {
					drop += 3
				}
				w.Heat = clampF(w.Heat-drop, 0, 120)
				w.Stars += d.Quality
				w.Stats.Dinners++
				w.say("good", "The neighbors came for %s. They asked for the recipe. Heat -%.0f, +%d stars.", RecipeByID(d.Sub).Name, drop, d.Quality)
				w.post("good", "hosted a dinner party. \"Best %s on the street,\" said Gladys, licking the plate.", strings.ToLower(RecipeByID(d.Sub).Name))
				break
			}
		}
	}
}

func (w *World) visitorOut(kind string) bool {
	for _, v := range w.Visitors {
		if v.Kind == kind {
			return true
		}
	}
	return false
}

func (w *World) daily() {
	if w.Has("garden") {
		earn := 8 + 2*minInt(w.Compost, 6)
		w.Cash += float64(earn)
		w.say("cash", "Tomatoes sold at the stand: +$%d. Customers ask what your secret is.", earn)
	}
	if w.Has("lure_airbnb") {
		w.Cash += 15
		w.say("cash", "Airbnb payout: +$15. Five stars, would stay again. Some didn't leave.")
	}
	if w.Has("soap_molds") {
		for i, d := range w.Pantry {
			if d.Sub == "soap" {
				w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
				w.Cash += 25
				w.Stats.Sold++
				w.say("cash", "Etsy sale: one bar of Paper Street Soap, +$25. Review: 'so creamy, what's in it?'")
				break
			}
		}
	}
	w.say("info", "Day %d. %s", w.Day(), morningLines[w.rng().Intn(len(morningLines))])
}

var morningLines = []string{
	"Sprinklers on. Somewhere a leaf blower.",
	"The HOA newsletter is in the mailbox. You are not in it. Yet.",
	"A neighbor waves. You wave back with the clean hand.",
	"Coffee. The good creamer. Nobody asks where the creamer comes from.",
	"Someone's garage band is covering Sweet Caroline again.",
	"The recycling truck is early. The other truck is never coming.",
	"Fog on the lawn. Good fog. Working fog.",
}

func (w *World) checkDeath() {
	if !w.Alive {
		return
	}
	p := w.Player()
	if p != nil && p.Starving > 2*day {
		w.Alive = false
		w.Cause = "starved, surrounded by neighbors"
		w.say("bad", "%s starved. In the suburbs. Surrounded by food. The shame outlives the body.", w.Cannibal)
		w.post("bad", "was found weeks later. The freezer was empty. The street called it 'a tragedy' and 'about time'.")
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
