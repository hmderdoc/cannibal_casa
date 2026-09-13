package game

import (
	"math/rand"
	"strings"
)

// endPhase advances the clock. Night -> Morning rolls a whole day, which is the
// only thing that spends the day bank.
func (s *State) endPhase(r *Result) Result {
	if s.Phase == Night {
		if s.Bank <= 0 {
			return r.fail("You're out of days. Sleep on it; more arrive tomorrow.")
		}
		s.Bank--
		s.Day++
		s.Phase = Morning
		s.dayRoll(r)
	} else {
		s.Phase++
		r.say("info", "%s, day %d.", s.Phase, s.Day)
	}
	if s.Mission != nil {
		s.awayRoll(r)
		s.Mission.Phases--
		if s.Mission.Phases <= 0 {
			s.resolveMission(r)
		}
	}
	if s.Inspection != nil {
		s.Inspection.Phases--
		if s.Inspection.Phases <= 0 {
			s.resolveInspection(r)
		} else {
			r.say("warn", "A car is idling out front. %d phase%s until the knock.", s.Inspection.Phases, plural(s.Inspection.Phases))
		}
	}
	if s.Phase == Night && s.Mission == nil {
		s.Prey = rollPrey(s, s.rng())
		r.say("info", "Night. The street is out walking. Someone should be careful.")
	}
	s.checkDeath(r)
	return *r
}

// dayRoll is everything that happens while you sleep.
func (s *State) dayRoll(r *Result) {
	rng := s.rng()
	tierBefore := s.Tier()
	r.say("info", "Day %d. %s", s.Day, morningLine(rng))

	// Hunger.
	s.Hunger = clamp(s.Hunger-15, 0, 100)
	if s.Hunger <= 25 {
		r.say("warn", "You're hungry enough to eat the mailman. Which is an option.")
	}

	// Fridge decay.
	spoiled := 0
	kept := s.Fridge[:0]
	for _, c := range s.Fridge {
		if !c.Frozen {
			c.Freshness--
			if c.Freshness <= 0 {
				spoiled++
				continue
			}
		}
		kept = append(kept, c)
	}
	s.Fridge = kept
	if spoiled > 0 {
		s.Heat = clamp(s.Heat+spoiled, 0, 120)
		r.say("bad", "%d cut%s spoiled. The trash smells like a crime. Heat +%d.", spoiled, plural(spoiled), spoiled)
	}
	keptD := s.Dishes[:0]
	stale := 0
	for _, d := range s.Dishes {
		d.Shelf--
		if d.Shelf <= 0 {
			stale++
			continue
		}
		keptD = append(keptD, d)
	}
	s.Dishes = keptD
	if stale > 0 {
		r.say("bad", "%d dish%s went off in the pantry.", stale, map[bool]string{true: "es", false: ""}[stale != 1])
	}

	// Livestock: feeding, fattening, dying, escaping.
	cost := 4
	if s.Has("gruel") {
		cost = 2
	}
	keptC := s.Captives[:0]
	for _, c := range s.Captives {
		if c.Renfield {
			keptC = append(keptC, c)
			continue
		}
		v := VictimByID(c.Victim)
		c.Days++
		if s.Cash >= cost {
			s.Cash -= cost
			c.Hungry = 0
			every := 3
			if s.Has("buffet") {
				every = 2
			}
			if c.Days%every == 0 && c.Marbling < 5 {
				c.Marbling++
				r.say("food", "%s is filling out nicely. Marbling %d.", c.Name, c.Marbling)
			}
		} else {
			c.Hungry++
			if c.Hungry >= 3 {
				r.say("bad", "%s starved in the cell. Wasteful. Bones and clothes to deal with.", c.Name)
				s.Evidence = append(s.Evidence, Evidence{ID: s.newID(), Kind: "bones", Victim: c.Victim}, Evidence{ID: s.newID(), Kind: "clothes", Victim: c.Victim})
				s.Heat = clamp(s.Heat+2, 0, 120)
				continue
			}
			r.say("warn", "No money to feed %s. Day %d without gruel.", c.Name, c.Hungry)
		}
		esc := 5 + 3*c.Hungry + v.Fight
		if s.Has("soundproof") {
			esc -= 3
		}
		if s.Has("shed") {
			esc--
		}
		esc -= s.Renfields()
		if rng.Intn(100) < clamp(esc, 1, 60) {
			s.Stats.Escaped++
			s.Heat = clamp(s.Heat+20, 0, 120)
			r.say("bad", "%s the %s ESCAPED in the night, screaming down the cul-de-sac. Heat +20.", c.Name, v.Name)
			r.post("bad", "had someone run out of their house at 4 A.M. wearing a bedsheet. \"Wine cellar\" my foot.")
			s.log("%s escaped.", c.Name)
			continue
		}
		keptC = append(keptC, c)
	}
	s.Captives = keptC

	// Renfields drift.
	keptC = s.Captives[:0]
	over := s.Renfields() - s.RenfieldCap()
	for _, c := range s.Captives {
		if !c.Renfield {
			keptC = append(keptC, c)
			continue
		}
		drop := 3
		if s.Has("big_tv") {
			drop = 1
		}
		if over > 0 {
			drop += 6
			over--
		}
		c.Loyalty -= drop
		if c.Loyalty <= 0 {
			s.Heat = clamp(s.Heat+30, 0, 120)
			r.say("bad", "%s snapped out of it and walked to the police station in your bathrobe. Heat +30.", c.Name)
			r.post("bad", "'s roommate has been giving interviews. Something about a slideshow.")
			s.log("Renfield %s fled.", c.Name)
			continue
		}
		if c.Loyalty <= 20 {
			r.say("warn", "%s keeps asking what year it is. Loyalty %d.", c.Name, c.Loyalty)
		}
		keptC = append(keptC, c)
	}
	s.Captives = keptC

	// Evidence radiates; buried evidence risks the dog.
	heat := 0
	for i := range s.Evidence {
		e := &s.Evidence[i]
		e.Age++
		if e.Buried {
			if !s.Has("fence") {
				dig := 4
				if s.Has("gnome") {
					dig = 3
				}
				if rng.Intn(100) < dig {
					e.Buried = false
					s.Heat = clamp(s.Heat+6, 0, 120)
					r.say("bad", "Gladys's labradoodle dug up the %s and paraded it down the street. Heat +6.", e.Kind)
				}
			}
			continue
		}
		heat += evidenceHeat(e.Kind, e.Age)
	}
	if heat > 0 {
		s.Heat = clamp(s.Heat+heat, 0, 120)
		r.say("heat", "Evidence lying around: Heat +%d.", heat)
	}
	if n := s.Livestock(); n > 0 && !s.Has("soundproof") {
		s.Heat = clamp(s.Heat+n, 0, 120)
		r.say("heat", "Muffled noises from the basement. Heat +%d.", n)
	}

	// Missing persons.
	keptM := s.Missing[:0]
	for _, m := range s.Missing {
		m.Days--
		if m.Days <= 0 {
			v := VictimByID(m.Victim)
			s.Heat = clamp(s.Heat+v.HeatMissing, 0, 120)
			if v.HeatMissing < 0 {
				r.say("good", "The %s was reported missing. Nobody seems upset. Heat %d.", strings.ToLower(v.Name), v.HeatMissing)
				r.post("info", "'s street: a %s has vanished. The comment section is celebrating.", strings.ToLower(v.Name))
			} else {
				r.say("heat", "A %s was reported missing. Heat +%d.", strings.ToLower(v.Name), v.HeatMissing)
				r.post("heat", "'s street: MISSING - %s, last seen near a house with a very nice hedge.", v.Name)
			}
			continue
		}
		keptM = append(keptM, m)
	}
	s.Missing = keptM

	// Heat cools; a house with nothing to see cools faster.
	cool := 3
	if s.VisibleEvidence() <= 0 {
		cool += 2
	}
	if s.Has("fence") {
		cool++
	}
	if s.Has("curtains") {
		cool++
	}
	s.Heat = clamp(s.Heat-cool, 0, 120)

	// Passive income.
	if s.Has("garden") {
		earn := 8 + 2*minInt(s.Compost, 6)
		s.Cash += earn
		r.say("cash", "Tomatoes sold at the stand: +$%d. Customers ask what your secret is.", earn)
	}
	if s.Has("lure_airbnb") {
		s.Cash += 15
		r.say("cash", "Airbnb payout: +$15. Five stars, would stay again. Some didn't leave.")
	}

	s.randomEvent(r, rng)

	// Crossing into a worse tier sends someone to the door.
	if t := s.Tier(); t > tierBefore && t >= 1 && s.Inspection == nil {
		ph := 2
		if s.Has("doorbell_cam") {
			ph++
		}
		s.Inspection = &Inspection{Phases: ph + 1, Tier: t, Who: inspectors[t]} // +1: endPhase decrements it in this same tick
		r.say("warn", "Heat tier: %s. Word is %s will be by in %d phases. Tidy up.", TierName(t), s.Inspection.Who, ph)
	}
}

// evidenceHeat is the daily Heat one loose piece radiates. A phone pings hard
// until its battery dies on day three; after that it only matters to a search.
func evidenceHeat(kind string, age int) int {
	switch kind {
	case "bones":
		return 1
	case "clothes":
		return 1
	case "phone":
		if age <= 3 {
			return 3
		}
		return 0
	case "car":
		return 4
	}
	return 1
}

var morningLines = []string{
	"Sprinklers on. Somewhere a leaf blower.",
	"The HOA newsletter is in the mailbox. You are not in it. Yet.",
	"A neighbor waves. You wave back with the clean hand.",
	"Coffee. The good creamer. Nobody asks where the creamer comes from.",
	"Someone's garage band is covering Sweet Caroline again.",
	"The recycling truck is early. The other truck is never coming.",
	"Cul-de-sac kids are chalking the driveway. Not yours. Never yours.",
	"Fog on the lawn. Good fog. Working fog.",
}

func morningLine(rng *rand.Rand) string { return morningLines[rng.Intn(len(morningLines))] }

// randomEvent rolls one flavor event per day, about half the time.
func (s *State) randomEvent(r *Result, rng *rand.Rand) {
	if rng.Intn(100) >= 45 {
		return
	}
	type ev struct {
		ok bool
		fn func()
	}
	hasMissing := func(id string) bool {
		for _, m := range s.Missing {
			if m.Victim == id {
				return true
			}
		}
		return false
	}
	frozen := 0
	for _, c := range s.Fridge {
		if c.Frozen {
			frozen++
		}
	}
	bones := 0
	for _, e := range s.Evidence {
		if e.Kind == "bones" && !e.Buried {
			bones++
		}
	}
	soap := -1
	for i, d := range s.Dishes {
		if d.Recipe == "soap" {
			soap = i
		}
	}
	events := []ev{
		{hasMissing("delivery"), func() {
			s.Heat = clamp(s.Heat+3, 0, 120)
			r.say("heat", "Packages are piling up on porches. Nobody's driver is coming. Heat +3.")
		}},
		{true, func() {
			s.Hunger = clamp(s.Hunger+12, 0, 100)
			r.say("food", "Gladys dropped off a tuna casserole. Insulting, but filling. +12 hunger.")
		}},
		{s.Tier() >= 2, func() {
			fine := minInt(25, s.Cash)
			s.Cash -= fine
			r.say("cash", "HOA fine: non-compliant mailbox. -$%d. The mailbox is fine. They know.", fine)
		}},
		{frozen > 0, func() {
			for i := range s.Fridge {
				if s.Fridge[i].Frozen {
					s.Fridge[i].Frozen = false
					s.Fridge[i].Freshness = 2
				}
			}
			r.say("bad", "Power outage. Everything in the freezer is thawing. Cook it or lose it.")
		}},
		{bones > 0, func() {
			s.Heat = clamp(s.Heat+3, 0, 120)
			r.say("heat", "Raccoons dragged something onto Gladys's lawn. Heat +3.")
		}},
		{soap >= 0, func() {
			s.Dishes = append(s.Dishes[:soap], s.Dishes[soap+1:]...)
			s.Cash += 25
			r.say("cash", "Etsy sale: one bar of Paper Street Soap, +$25. Review: 'so creamy, what's in it?'")
		}},
		{s.Tier() >= 4, func() {
			s.Heat = clamp(s.Heat+4, 0, 120)
			r.say("heat", "Episode 12 dropped: 'The Cul-de-Sac Killer'. Heat +4. The ad read is for a meal kit.")
		}},
		{s.Renfields() > 0, func() {
			for i := range s.Captives {
				if s.Captives[i].Renfield {
					s.Captives[i].Loyalty = clamp(s.Captives[i].Loyalty+10, 0, 100)
					r.say("good", "%s made you a lanyard. Loyalty +10.", s.Captives[i].Name)
					return
				}
			}
		}},
		{s.Livestock() < s.CellCount(), func() {
			v := VictimByID("crypto")
			s.capture(r, v, rng)
			r.say("good", "A very drunk man walked in thinking it was his house. It's not. It's his cell now.")
		}},
		{true, func() {
			s.Cash += 10
			r.say("cash", "Found $10 in a pair of donated jeans. Finders keepers.")
		}},
	}
	var pool []ev
	for _, e := range events {
		if e.ok {
			pool = append(pool, e)
		}
	}
	if len(pool) > 0 {
		pool[rng.Intn(len(pool))].fn()
	}
}

// awayRoll runs the house while its owner is out. Cameras and Renfields keep
// it honest; an unwatched basement is an escape waiting to happen.
func (s *State) awayRoll(r *Result) {
	rng := s.rng()
	keptC := s.Captives[:0]
	for _, c := range s.Captives {
		if c.Renfield {
			if !s.Has("cameras") {
				c.Loyalty--
			}
			keptC = append(keptC, c)
			continue
		}
		esc := 4
		if s.Has("cameras") {
			esc = 1
		}
		esc -= s.Renfields()
		if s.Has("soundproof") {
			esc--
		}
		if rng.Intn(100) < clamp(esc, 1, 50) {
			s.Stats.Escaped++
			s.Heat = clamp(s.Heat+20, 0, 120)
			r.say("bad", "While you were out, %s got the cell door open and ran. Heat +20.", c.Name)
			r.post("bad", "'s house: someone climbed out of a basement window while the owner was out. Neighbors 'concerned'.")
			continue
		}
		keptC = append(keptC, c)
	}
	s.Captives = keptC
	if !s.Has("fence") {
		for i := range s.Evidence {
			if s.Evidence[i].Buried && rng.Intn(100) < 2 {
				s.Evidence[i].Buried = false
				s.Heat = clamp(s.Heat+6, 0, 120)
				r.say("bad", "The labradoodle had the yard to itself. Something is un-buried. Heat +6.")
			}
		}
	}
}

func (s *State) resolveMission(r *Result) {
	m := MissionByID(s.Mission.ID)
	s.Mission = nil
	rng := s.rng()
	switch m.ID {
	case "day_job":
		s.Cash += 40
		r.say("cash", "Three calls, one 'circle back', +$40. Someone asked about your background. You said 'meat'.")
	case "market":
		earned, sold := 0, 0
		kept := s.Dishes[:0]
		for _, d := range s.Dishes {
			rc := RecipeByID(d.Recipe)
			if rc.Price > 0 {
				earned += rc.Price + 5*(d.Quality-3)
				sold++
				continue
			}
			kept = append(kept, d)
		}
		s.Dishes = kept
		if s.Has("garden") {
			earned += 15
		}
		s.Cash += earned
		s.Stats.Sold += sold
		r.say("cash", "Market day: sold %d item%s for $%d. 'Is it organic?' 'He was.'", sold, plural(sold), earned)
		if sold >= 4 {
			r.post("cash", "sold out at the farmers market again. The pies. People line up for the pies.")
		}
	case "city_hunt":
		v := &Victims[rng.Intn(len(Victims))]
		for v.Weight == 0 || v.MinTier > s.Tier() {
			v = &Victims[rng.Intn(len(Victims))]
		}
		if rng.Intn(100) < clamp(s.huntChance(v)+15, 10, 95) {
			r.say("good", "Downtown delivered.")
			s.capture(r, v, rng)
		} else {
			s.Heat = clamp(s.Heat+2, 0, 120)
			r.say("bad", "Came home empty-handed. Someone downtown got your plate. Heat +2.")
		}
	case "tinder":
		if rng.Intn(100) < 75 {
			opts := []string{"techbro", "crypto", "influencer", "vegan", "mlm", "landlord"}
			v := VictimByID(opts[rng.Intn(len(opts))])
			s.Heat = clamp(s.Heat+3, 0, 120)
			r.say("good", "The date went great. They came back for a nightcap. Heat +3 (their car is at the bar).")
			s.capture(r, v, rng)
		} else {
			r.say("bad", "Ghosted. Their loss, and yours, and specifically your fridge's.")
		}
	case "burglary":
		take := 60 + rng.Intn(100)
		s.Cash += take
		r.say("cash", "The Hendersons' garage: +$%d and a very nice cordless drill.", take)
		if rng.Intn(100) < 15 {
			s.Heat = clamp(s.Heat+10, 0, 120)
			r.say("bad", "Their doorbell cam got your van. Heat +10.")
		}
	case "vigilante":
		s.Heat = clamp(s.Heat-12, 0, 120)
		s.Stars++
		r.say("good", "Somebody cleaned up the street. Heat -12, +1 star.")
		r.post("good", "'s street sleeps easier. A certain guy who had it coming has not been seen.")
		s.capture(r, VictimByID("vigilante_target"), rng)
	}
	s.log("Mission done: %s.", m.Name)
}

// VisibleEvidence scores what a visitor would notice from the front hall.
func (s *State) VisibleEvidence() int {
	v := 0
	for _, e := range s.Evidence {
		if e.Buried {
			continue
		}
		switch e.Kind {
		case "bones":
			v += 2
		case "clothes":
			v++
		case "phone":
			v += 3
		case "car":
			v += 4
		}
	}
	if !s.Has("soundproof") {
		v += 2 * s.Livestock()
	}
	if s.Has("curtains") {
		v -= 3
	}
	if s.Mission != nil {
		v += 2
	}
	return v
}

// InspectionThreshold is how much a given tier lets slide.
func InspectionThreshold(tier int) int {
	return []int{9, 6, 5, 4, 3, 2}[clamp(tier, 0, 5)]
}

func (s *State) resolveInspection(r *Result) {
	in := s.Inspection
	s.Inspection = nil
	visible := s.VisibleEvidence()
	limit := InspectionThreshold(in.Tier)
	if visible <= limit {
		drop := 6
		for i, d := range s.Dishes {
			if RecipeByID(d.Recipe).Food && d.Quality >= 3 {
				s.Dishes = append(s.Dishes[:i], s.Dishes[i+1:]...)
				drop += 4
				r.say("good", "You had a %s to offer. They left with a slice and a smile.", RecipeByID(d.Recipe).Name)
				break
			}
		}
		s.Heat = clamp(s.Heat-drop, 0, 120)
		r.say("good", "Visit from %s. They saw a lovely home. Heat -%d.", in.Who, drop)
		return
	}
	rise := 10 + 3*in.Tier
	s.Heat = clamp(s.Heat+rise, 0, 120)
	r.say("bad", "Visit from %s. They saw too much (%d vs %d allowed). Heat +%d.", in.Who, visible, limit, rise)
	r.post("heat", "had visitors. They left quickly and made a phone call in the driveway.")
	if in.Tier >= 4 && s.Livestock() > 0 && s.rng().Intn(100) < 30 {
		for i, c := range s.Captives {
			if !c.Renfield {
				s.Captives = append(s.Captives[:i], s.Captives[i+1:]...)
				s.Heat = clamp(s.Heat+15, 0, 120)
				r.say("bad", "They took %s with them. Heat +15.", c.Name)
				break
			}
		}
	}
}

func (s *State) checkDeath(r *Result) {
	if !s.Alive {
		return
	}
	switch {
	case s.Heat >= 100:
		s.Alive = false
		s.Cause = "raided by " + TierName(5)
		r.say("bad", "Flashbangs. Boots. A tactical team in the kitchen admiring the knife set. %s is done.", s.Cannibal)
		r.post("bad", "'s house got raided at dawn. Fourteen vehicles. The HOA is 'reviewing the hedge situation'.")
	case s.Hunger <= 0:
		s.Alive = false
		s.Cause = "starved, surrounded by neighbors"
		r.say("bad", "%s starved. In the suburbs. Surrounded by food. The shame outlives the body.", s.Cannibal)
		r.post("bad", "was found weeks later. The freezer was empty. The street called it 'a tragedy' and 'about time'.")
	}
	if !s.Alive {
		s.log("%s: %s.", s.Cannibal, s.Cause)
	}
}
