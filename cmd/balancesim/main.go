// balancesim runs Cannibal Casa's simulation headless under a few build
// orders and reports how the machine fares over N days: survival, meals,
// stars, cash, heat, and what killed it. Same package as the door and server.
//
//	go run ./cmd/balancesim -runs 50 -days 60
//	go run ./cmd/balancesim -runs 1 -days 30 -trace factory
package main

import (
	"flag"
	"fmt"
	"sort"

	"cannibalcasa/internal/sim"
)

type plan struct {
	name  string
	order []string // fixtures bought in order whenever affordable (+$40 reserve)
	tune  func(w *sim.World)
}

var plans = []plan{
	{"hermit", nil, nil},
	{"tidy", []string{"microwave", "stockpot", "plot", "compost", "fence"}, nil},
	{"factory", []string{"lure_couch", "microwave", "grinder", "stand", "stockpot", "cell", "compost", "freezer", "fence", "shed", "cell", "soundproof", "gruel", "curtains", "smoker", "chair", "cot", "cameras"},
		func(w *sim.World) { w.Policy.Brainwash = true }},
	{"sweeney", []string{"grinder", "stand", "microwave", "lure_openhouse", "stockpot", "compost", "fence", "dining_set", "wine_rack", "cell", "freezer", "knife_set", "curtains", "cell", "chair", "vhs", "cot", "big_tv", "smoker", "cameras", "trapdoor_pro"},
		func(w *sim.World) { w.Policy.Brainwash = true; w.Policy.PantryTarget = 4 }},
}

// freeTile is the first legal tile for a fixture, scanning from the top left.
func freeTile(w *sim.World, f *sim.Fixture) sim.Pos {
	p := sim.Plans[f.Level]
	for y, row := range p.Rows {
		for x := range row {
			pos := sim.Pos{L: f.Level, X: x, Y: y}
			if w.CanPlace(f, pos) == nil {
				return pos
			}
		}
	}
	return sim.Pos{}
}

func main() {
	runs := flag.Int("runs", 30, "games per plan")
	days := flag.Int("days", 60, "days per game")
	trace := flag.String("trace", "", "print the ticker of the first game for this plan")
	deaths := flag.Bool("deaths", false, "print the last ticker lines before each death")
	flag.Parse()
	for _, p := range plans {
		var alive, totalDays, stars, meals, captured, sold int
		var cash, heat float64
		causes := map[string]int{}
		for run := 0; run < *runs; run++ {
			w := sim.New("bot", "sim", int64(run*7919+1))
			if p.tune != nil {
				p.tune(w)
			}
			shown := 0
			for d := 0; d < *days && w.Alive; d++ {
				for _, id := range p.order {
					f := sim.FixtureByID(id)
					owned := w.Has(id)
					if f.Repeat {
						owned = w.Instances(id) >= 3
					}
					if !owned && w.Cash >= float64(w.NextCost(f)+40) {
						w.Build(id, freeTile(w, f))
					}
				}
				w.Advance(1440)
				if *trace == p.name && run == 0 {
					for ; shown < len(w.Ticker); shown++ {
						l := w.Ticker[shown]
						fmt.Printf("d%-3d %02d:%02d [%s] %s\n", l.Tick/1440+1, l.Tick%1440/60, l.Tick%60, l.Kind, l.Text)
					}
					fmt.Printf("     -- heat %.0f cash %.0f livestock %d fridge %d pantry %d pile %d buried %d hunger %.0f workers %d\n",
						w.Heat, w.Cash, len(w.Livestock), len(w.Fridge), len(w.Pantry), len(w.Pile), w.Buried, w.Player().Hunger, len(w.Workers))
				}
			}
			if w.Alive {
				alive++
			} else {
				causes[w.Cause]++
				if *deaths && run < 4 {
					fmt.Printf("  %s run %d died day %d (%s):\n", p.name, run, w.DaysAlive(), w.Cause)
					start := len(w.Ticker) - 24
					if start < 0 {
						start = 0
					}
					for _, l := range w.Ticker[start:] {
						if l.Kind == "bad" || l.Kind == "heat" || l.Kind == "warn" {
							fmt.Printf("    d%-3d %02d:%02d %s\n", l.Tick/1440+1, l.Tick%1440/60, l.Tick%60, l.Text)
						}
					}
				}
			}
			totalDays += w.DaysAlive()
			stars += w.Stars
			meals += w.Stats.Meals
			captured += w.Stats.Captured
			sold += w.Stats.Sold
			cash += w.Cash
			heat += w.Heat
		}
		n := float64(*runs)
		fmt.Printf("%-8s survive %3.0f%%  days %5.1f  stars %5.1f  meals %5.1f  captured %5.1f  sold %5.1f  cash $%5.0f  heat %4.1f\n",
			p.name, 100*float64(alive)/n, float64(totalDays)/n, float64(stars)/n, float64(meals)/n, float64(captured)/n, float64(sold)/n, cash/n, heat/n)
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
