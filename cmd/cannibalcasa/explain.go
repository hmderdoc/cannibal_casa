// Plain-language explanations of what a fixture is for. The data table's
// Desc is the joke; this is the sentence under it that tells you what you're
// buying. No math.
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/sim"
)

// effects say what the passive upgrades do, in a sentence.
var effects = map[string]string{
	"doorbell_cam": "Gives you warning before a visitor knocks, and helps keep raiders out.",
	"grinder":      "Lets the stove make meat pies, the best thing you can sell at the stand.",
	"knife_set":    "Cleaner, faster butchering: more prime cuts from every body.",
	"dining_set":   "Neighbors come over in the evening and eat your best dish. Heat drops, stars rise.",
	"curtains":     "Visitors see less of what's inside, and the street calms down faster.",
	"big_tv":       "Keeps slaves loyal.",
	"wine_rack":    "Unlocks the liver with fava beans, and makes dinner parties go better.",
	"vhs":          "The Chair works much faster.",
	"chemistry":    "Chloroform for the mat: the trap catches more of the people it fires on.",
	"soundproof":   "Nobody hears the basement. Livestock stops raising Heat and stops trying the door.",
	"gruel":        "Feeds every cell by itself, cheaply. No worker has to walk down.",
	"buffet":       "Livestock fattens faster.",
	"garden":       "Pays a little cash every morning, more the more the compost has eaten.",
	"fence":        "Keeps the dog out, hides the yard from visitors, calms the street, slows raiders.",
	"shed":         "Burying is quicker and quieter.",
	"gnome":        "A little luck on the trap. He's a gnome.",
	"cameras":      "Livestock can't slip out, and raiders get caught.",
	"telescope":    "Better trapping after dark, and joggers take the bait.",
	"soap_molds":   "Turns scraps into soap that sells itself online.",
	"lure_airbnb":  "Captives also pay for the room.",
}

func explain(w *sim.World, f *sim.Fixture) []sim.Line {
	out := []sim.Line{{Kind: "flavor", Text: f.Desc}}
	switch f.Kind {
	case sim.KindLure:
		var who []string
		for _, d := range f.Draws {
			who = append(who, sim.Traits[d])
		}
		text := "Lure. Draws people off the sidewalk onto the porch, where the trap can get them. It doesn't catch anyone by itself."
		if len(who) > 0 {
			text += " Works best on " + strings.Join(who, " and ") + "."
		}
		out = append(out, sim.Line{Kind: "info", Text: text})
	case sim.KindTrap:
		out = append(out, sim.Line{Kind: "info", Text: "Trap. Drops a lured visitor into a free cell. Fighters get away more often; better gear helps."})
	case sim.KindStand:
		out = append(out, sim.Line{Kind: "info", Text: "Shop. Workers stock it with whatever Policy marks for sale; it sells through the day. Your steady income."})
	case sim.KindStation:
		var rs []string
		for _, rc := range sim.Recipes {
			if rc.Station == f.ID {
				rs = append(rs, rc.Name)
			}
		}
		out = append(out, sim.Line{Kind: "info", Text: "Cooking station. Makes: " + strings.Join(rs, "; ") + "."})
	case sim.KindDisposal:
		how := "quietly"
		if f.Noise > 0 {
			how = "loudly"
		}
		out = append(out, sim.Line{Kind: "info", Text: "Evidence disposal. Gets rid of " + strings.Join(f.Accepts, ", ") + " from the back step, " + how + ", one bag at a time."})
	case sim.KindPlot:
		out = append(out, sim.Line{Kind: "info", Text: "Burial. Somewhere to put bags so they stop raising Heat. The dog digs unless there's a fence."})
	case sim.KindContainer:
		switch f.ID {
		case "fridge", "fridge_big":
			out = append(out, sim.Line{Kind: "info", Text: "Cold storage for cuts. Keep it near the block and the stove; workers walk."})
		case "freezer", "chest_freezer":
			out = append(out, sim.Line{Kind: "info", Text: "Frozen storage. Catches the overflow when the fridge is full; frozen meat never spoils but cooks a little worse."})
		case "pantry", "pantry_big":
			out = append(out, sim.Line{Kind: "info", Text: "Shelf for finished dishes. Everyone eats from here; the stand is stocked from here."})
		case "back_step":
			out = append(out, sim.Line{Kind: "info", Text: "Where the bags pile up after butchering. Every bag here raises Heat until something takes it."})
		}
	case sim.KindCell:
		out = append(out, sim.Line{Kind: "info", Text: "Holds one captive. The trap only fires when a cell is free, so more cells means more intake. Captives eat, fatten, and get loud if they're hungry."})
	case sim.KindBed:
		if f.ID == "bed" {
			out = append(out, sim.Line{Kind: "info", Text: "Where you sleep. Keep it near the work."})
		} else {
			out = append(out, sim.Line{Kind: "info", Text: "Each cot lets you keep one more slave."})
		}
	case sim.KindChair:
		out = append(out, sim.Line{Kind: "info", Text: "Brainwashing. Turns a captive into a slave who works for you. Needs a free cot."})
	case sim.KindBlock:
		out = append(out, sim.Line{Kind: "info", Text: "Butchering. Captives become cuts for the fridge and bags for the back step."})
	case sim.KindVan:
		out = append(out, sim.Line{Kind: "info", Text: "Hauls away the cars your visitors leave at the curb, and helps on raids."})
	case sim.KindBooth:
		out = append(out, sim.Line{Kind: "info", Text: "Your spot at the mall. Customers on the concourse buy what's stocked, at mall prices, and every sale makes you a little more respectable. A clerk keeps it stocked; rent comes out hourly."})
	case sim.KindPassive:
		out = append(out, sim.Line{Kind: "info", Text: effects[f.ID]})
	}
	if e, ok := effects[f.ID]; ok && f.Kind != sim.KindPassive {
		out = append(out, sim.Line{Kind: "info", Text: e})
	}
	where := "Goes anywhere on the " + f.Levels() + "."
	if f.Level == sim.Mall {
		where = "Goes on the mall concourse, where the customers walk."
	}
	if f.Repeat {
		where += fmt.Sprintf(" Build as many as fit; each costs $%d more than the last.", f.CostStep)
	}
	if f.Front {
		where = "Goes on the porch."
	}
	out = append(out, sim.Line{Kind: "rule", Text: where})
	return out
}
