// balancesim plays Cannibal Casa headless with a handful of bot policies and
// reports survival, stars and causes of death, so the economy can be tuned
// before a caller ever sees it. Same rules package as the door and server.
//
//	go run ./cmd/balancesim -runs 200 -days 60
package main

import (
	"flag"
	"fmt"
	"sort"

	gm "cannibalcasa/internal/game"
)

type policy struct {
	name string
	// step is called whenever the bot may act; return false when it wants to
	// end the phase.
	step func(s *gm.State) bool
}

var tracing bool

func act(s *gm.State, a gm.Action) bool {
	r := gm.Apply(s, a)
	if tracing {
		if r.Err != "" {
			fmt.Printf("   %s %s %d -> ERR %s\n", a.Kind, a.Name, a.ID, r.Err)
		} else {
			for _, e := range r.Events {
				fmt.Printf("   %s %s -> [%s] %s\n", a.Kind, a.Name, e.Kind, e.Text)
			}
		}
	}
	return r.Err == ""
}

// eatIfHungry eats the best food dish when hunger is low.
func eatIfHungry(s *gm.State, below int) bool {
	if s.Hunger >= below {
		return false
	}
	best := -1
	for i, d := range s.Dishes {
		if gm.RecipeByID(d.Recipe).Food && (best < 0 || d.Quality > s.Dishes[best].Quality) {
			best = i
		}
	}
	if best < 0 {
		return false
	}
	return act(s, gm.Action{Kind: "eat", ID: s.Dishes[best].ID})
}

// cookSomething cooks the best recipe the fridge supports, keeping a small
// pantry rather than turning the whole fridge into dishes that spoil.
func cookSomething(s *gm.State) bool {
	if len(s.Dishes) >= 2 {
		return false
	}
	order := []string{"charcuterie", "sous_vide", "pulled_pork", "ribs", "fava", "meat_pie", "pan_sear", "liver_onions", "fritters", "jerky", "tartare", "soylent"}
	for _, id := range order {
		rc := gm.RecipeByID(id)
		if !s.CanCook(rc) {
			continue
		}
		if act(s, gm.Action{Kind: "cook", Name: id}) {
			return true
		}
	}
	return false
}

// broth turns loose bones into soup whenever the stockpot allows.
func broth(s *gm.State) bool {
	if !s.CanCook(gm.RecipeByID("broth")) {
		return false
	}
	for _, e := range s.Evidence {
		if e.Kind == "bones" && !e.Buried {
			return act(s, gm.Action{Kind: "cook", Name: "broth"})
		}
	}
	return false
}

// tidy disposes of one piece of evidence with the first available method.
func tidy(s *gm.State) bool {
	for _, e := range s.Evidence {
		if e.Buried {
			continue
		}
		for _, m := range s.DisposalMethods(e) {
			if act(s, gm.Action{Kind: "dispose", ID: e.ID, Name: m}) {
				return true
			}
		}
	}
	return false
}

func buyNext(s *gm.State, wish []string) bool {
	for _, id := range wish {
		f := gm.FixtureByID(id)
		if s.Has(id) || s.Cash < f.Cost+40 {
			continue
		}
		if act(s, gm.Action{Kind: "build", Name: id}) {
			return true
		}
	}
	return false
}

// hunt goes out only when the larder is thin: a fed cannibal who hunts every
// night racks up missing-person Heat faster than any fence can cool it.
func hunt(s *gm.State) bool {
	if s.Phase != gm.Night || len(s.Prey) == 0 {
		return false
	}
	if len(s.Fridge)+len(s.Dishes) >= 3 || s.Livestock() > 0 {
		return false
	}
	best := s.Prey[0]
	for _, p := range s.Prey {
		if s.HuntChance(p) > s.HuntChance(best) {
			best = p
		}
	}
	return act(s, gm.Action{Kind: "hunt", Name: best})
}

func butcherWhenNeeded(s *gm.State) bool {
	if len(s.Fridge) > 2 || s.Livestock() == 0 {
		return false
	}
	for _, c := range s.Captives {
		if !c.Renfield {
			return act(s, gm.Action{Kind: "butcher", ID: c.ID})
		}
	}
	return false
}

var policies = []policy{
	{"glutton", func(s *gm.State) bool {
		if s.Phase == gm.Night && len(s.Prey) > 0 {
			if act(s, gm.Action{Kind: "hunt", Name: s.Prey[0]}) {
				return true
			}
		}
		return eatIfHungry(s, 55) || butcherWhenNeeded(s) || cookSomething(s)
	}},
	{"tidy", func(s *gm.State) bool {
		if s.Phase == gm.Morning && s.Mission == nil && s.Cash < 60 {
			if act(s, gm.Action{Kind: "mission", Name: "day_job"}) {
				return true
			}
		}
		return eatIfHungry(s, 55) || broth(s) || tidy(s) || buyNext(s, []string{"microwave", "stockpot", "plot_2"}) || butcherWhenNeeded(s) || cookSomething(s) || hunt(s)
	}},
	{"builder", func(s *gm.State) bool {
		wish := []string{"microwave", "compost", "shed", "fence", "grinder", "smoker", "dining_set", "chair", "cell_2", "soundproof", "freezer", "curtains", "knife_set"}
		if s.Phase == gm.Morning && s.Mission == nil && s.Cash < 150 && s.Hunger > 40 {
			if act(s, gm.Action{Kind: "mission", Name: "day_job"}) {
				return true
			}
		}
		if s.Phase == gm.Afternoon && s.Mission == nil && (len(s.Dishes) >= 3) {
			if act(s, gm.Action{Kind: "mission", Name: "market"}) {
				return true
			}
		}
		if s.Phase == gm.Evening && s.Has("dining_set") && s.Heat > 35 {
			if act(s, gm.Action{Kind: "dinner"}) {
				return true
			}
		}
		return eatIfHungry(s, 55) || tidy(s) || buyNext(s, wish) || butcherWhenNeeded(s) || cookSomething(s) || hunt(s)
	}},
	{"sweeney", func(s *gm.State) bool {
		wish := []string{"grinder", "microwave", "shed", "compost", "fence", "dining_set", "curtains", "stockpot", "fridge_big", "smoker", "chair", "vhs", "cameras"}
		if s.Phase == gm.Morning && s.Mission == nil && s.Cash < 150 && s.Hunger > 40 {
			if act(s, gm.Action{Kind: "mission", Name: "day_job"}) {
				return true
			}
		}
		if s.Phase == gm.Afternoon && s.Mission == nil && (len(s.Dishes) >= 2) {
			if act(s, gm.Action{Kind: "mission", Name: "market"}) {
				return true
			}
		}
		if s.Phase == gm.Evening && s.Has("dining_set") && s.Heat > 30 {
			if act(s, gm.Action{Kind: "dinner"}) {
				return true
			}
		}
		if s.Has("chair") {
			for _, c := range s.Captives {
				if !c.Renfield && c.LastTry != s.Day && s.Renfields() < s.RenfieldCap() {
					if act(s, gm.Action{Kind: "brainwash", ID: c.ID}) {
						return true
					}
				}
			}
		}
		return eatIfHungry(s, 55) || tidy(s) || buyNext(s, wish) || butcherWhenNeeded(s) || cookSomething(s) || hunt(s)
	}},
}

func main() {
	runs := flag.Int("runs", 100, "games per policy")
	days := flag.Int("days", 60, "in-game days per game")
	trace := flag.String("trace", "", "print every event of the first game for this policy")
	flag.Parse()
	for _, p := range policies {
		var survived, totalDays, totalStars, totalMeals, totalCash, totalCaptured int
		causes := map[string]int{}
		heatSum := 0
		for run := 0; run < *runs; run++ {
			s := gm.NewState("bot", "sim", int64(run*7919+1))
			tracing = *trace == p.name && run == 0
			for s.Day <= *days && s.Alive {
				gm.GrantAllowance(s, gm.Rules{DaysPerDay: 1, BankMax: 3}, fmt.Sprint("d", s.Day, s.Phase))
				for i := 0; i < 12 && p.step(s); i++ {
				}
				r := gm.Apply(s, gm.Action{Kind: "end_phase"})
				if *trace == p.name && run == 0 {
					fmt.Printf("-- d%d %s hunger=%d heat=%d cash=%d fridge=%d dishes=%d captives=%d evidence=%d\n", s.Day, s.Phase, s.Hunger, s.Heat, s.Cash, len(s.Fridge), len(s.Dishes), len(s.Captives), len(s.Evidence))
					for _, e := range r.Events {
						fmt.Printf("   [%s] %s\n", e.Kind, e.Text)
					}
				}
				if r.Err != "" {
					break
				}
			}
			if s.Alive {
				survived++
			} else {
				causes[s.Cause]++
			}
			totalDays += s.Day
			totalStars += s.Stars
			totalMeals += s.Meals
			totalCash += s.Cash
			totalCaptured += s.Stats.Captured
			heatSum += s.Heat
		}
		n := float64(*runs)
		fmt.Printf("%-8s survive %3.0f%%  days %5.1f  stars %5.1f  meals %5.1f  cash $%5.0f  captured %4.1f  heat %4.1f\n",
			p.name, 100*float64(survived)/n, float64(totalDays)/n, float64(totalStars)/n, float64(totalMeals)/n,
			float64(totalCash)/n, float64(totalCaptured)/n, float64(heatSum)/n)
		var keys []string
		for k := range causes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("         %-40s %d\n", k, causes[k])
		}
	}
}
