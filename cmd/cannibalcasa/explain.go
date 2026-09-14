// Literal explanations of what a fixture does: category, mechanics with the
// real numbers, who it attracts or what it accepts, where it has to go. The
// data table's Desc is the joke; this is the manual.
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/sim"
)

// effects are the mechanics of passive fixtures, which have no Kind behavior
// of their own to describe.
var effects = map[string]string{
	"doorbell_cam": "Inspectors take 30 minutes to work up to knocking instead of 15, so you see them coming. Passers-by are 0.5% less likely to be lured. Adds +10 to your defense against raids.",
	"grinder":      "Unlocks Mrs. Lovett's Meat Pies at the stove ($30 each at the stand, 90 minutes to cook).",
	"knife_set":    "Every butchered body yields 3 prime cuts instead of 2, and the block works in 45 minutes instead of 60.",
	"dining_set":   "At 19:00 every day, if a four-star-or-better dish is on the shelf, the neighbors come over and eat it: Heat -20 (or -25 for five stars), stars +4 or +5.",
	"curtains":     "Visitors see 2 points less of what's in the house, and Heat cools 0.2/hour faster.",
	"big_tv":       "Renfield loyalty drifts down 1 per day instead of 3.",
	"wine_rack":    "Unlocks Liver, Fava Beans, a Nice Chianti at the stove (four stars from a single offal cut). Dinner parties drop 3 more Heat.",
	"vhs":          "The Chair works in 3 hours instead of 6.",
	"chemistry":    "Trap odds +12% (chloroform on the mat). Raids you make get +10 attack.",
	"soundproof":   "Livestock no longer radiates Heat (0.04/hour each, 0.12 when hungry) and is 0.1%/hour less likely to try the door. Screaming stops counting for inspections.",
	"gruel":        "Feeds every cell automatically the moment a guest gets hungry, at $2 instead of $4, with no worker trip.",
	"buffet":       "Livestock gains a marbling point every 2 days instead of 3.",
	"garden":       "Pays $8 a day plus $2 per bag the compost has eaten (up to 6), every morning. Needs the compost bin.",
	"fence":        "The dog can't get into the yard, bags on the back step no longer count for inspections, and Heat cools 0.2/hour faster. Adds +15 to raid defense.",
	"shed":         "Burying takes half the time and makes no noise at night.",
	"gnome":        "Trap odds +3%. That's it. He's a gnome.",
	"cameras":      "Livestock is five times less likely to escape. Adds +25 to raid defense.",
	"telescope":    "Trap odds +8% after dark, and joggers are lured more often.",
	"soap_molds":   "Unlocks Paper Street Soap at the stockpot (scraps in, $25 out); one bar sells itself on Etsy every morning.",
	"lure_airbnb":  "Also pays $15 every morning.",
}

func explain(w *sim.World, f *sim.Fixture) []sim.Line {
	var out []sim.Line
	out = append(out, sim.Line{Kind: "dim", Text: f.Desc})
	switch f.Kind {
	case sim.KindLure:
		var names []string
		for _, v := range sim.Victims {
			for _, l := range v.Lures {
				if l == f.ID {
					names = append(names, v.Name)
				}
			}
		}
		out = append(out, sim.Line{Kind: "info", Text: "LURE. Sits on the porch and pulls passers-by off the sidewalk toward the trap. It doesn't capture anyone; the mat does that. Every lure you own adds 0.6% to any passer-by's chance of turning in at the gate, and 3% for the types it's built for."})
		if len(names) > 0 {
			out = append(out, sim.Line{Kind: "food", Text: "Pulls hardest on: " + strings.Join(names, ", ") + "."})
		}
	case sim.KindTrap:
		base := 45
		if f.ID == "trapdoor_pro" {
			base = 70
		}
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("TRAP. When a lured passer-by steps on it and a cell is free, it fires: %d%% base, minus 8%% per point of the victim's fight, plus gear (chemistry set +12, telescope +8 at night, gnome +3). Success drops them into a cell and takes their wallet; failure sometimes screams (Heat). Resets for an hour either way.", base)})
	case sim.KindStand:
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("SHOP. Holds %d items. Sells one an hour from 09:00 to 18:00 at the recipe's price (+/- $5 per star from three). Workers stock it with dishes marked for sale in Policy.", f.Capacity)})
	case sim.KindStation:
		var rs []string
		for _, rc := range sim.Recipes {
			if rc.Station == f.ID {
				rs = append(rs, fmt.Sprintf("%s (%s)", rc.Name, minutes(rc.Duration)))
			}
		}
		out = append(out, sim.Line{Kind: "info", Text: "COOKING STATION. A worker carries the cuts here and spends 10 minutes loading; the station then runs on its own and holds the finished dish until someone plates it to the shelf."})
		out = append(out, sim.Line{Kind: "food", Text: "Cooks: " + strings.Join(rs, "; ") + "."})
	case sim.KindDisposal:
		noise := "silent"
		if f.Noise > 0 {
			noise = fmt.Sprintf("Heat +%.0f per use, the street hears it", f.Noise)
		}
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("DISPOSAL. A worker carries a bag from the back step and loads it (3 minutes); it takes %s per bag, one at a time, %s. Accepts: %s.", minutes(f.Duration), noise, strings.Join(f.Accepts, ", "))})
	case sim.KindPlot:
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("BURIAL. Room for %d bags underground. A worker digs for %s per bag (half with the shed). Buried bags stop radiating Heat but the dog can dig them up unless there's a fence.", f.Capacity, minutes(f.Duration))})
	case sim.KindContainer:
		switch f.ID {
		case "fridge", "fridge_big":
			out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("COLD STORAGE. +%d fresh cuts. Cuts last 3 days here (offal 2) before they spoil. Put it close to the block and the stove: workers walk.", f.Capacity)})
		case "freezer", "chest_freezer":
			out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("FREEZER. +%d frozen cuts. When the fridge is full, cuts freeze here instead of going in the trash (Policy: Freeze overflow). Frozen never spoils but cooks one star lower.", f.Capacity)})
		case "pantry", "pantry_big":
			out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("SHELF. +%d finished dishes. Workers eat from here and stock the stand from here. Full shelf means stations sit holding dishes.", f.Capacity)})
		case "back_step":
			out = append(out, sim.Line{Kind: "info", Text: "EVIDENCE PILE. Every butchered body leaves bones, clothes and a phone here. Each bag radiates Heat every hour until a disposal machine or a plot takes it."})
		}
	case sim.KindCell:
		out = append(out, sim.Line{Kind: "info", Text: "CELL. Holds one guest. The trap only fires when a cell is free, so cells are your intake rate. A guest eats $4 a day and gains a marbling point every 3 days when fed; hungry guests scream and try the door."})
	case sim.KindBed:
		if f.ID == "bed" {
			out = append(out, sim.Line{Kind: "info", Text: "BED. Your cannibal sleeps 8 hours a night here. Nothing else happens while they do, so keep it near the work."})
		} else {
			out = append(out, sim.Line{Kind: "info", Text: "COT. Each cot lets you keep one more Renfield. The Chair won't take anyone when every cot is spoken for."})
		}
	case sim.KindChair:
		out = append(out, sim.Line{Kind: "info", Text: "BRAINWASHING. A worker walks a guest here; 6 hours later (3 with the tapes) they come out a Renfield who works for you, if a cot is free. Order someone in from the roster or set the policy."})
	case sim.KindBlock:
		out = append(out, sim.Line{Kind: "info", Text: "BUTCHER STATION. A worker walks a guest here and works 60 minutes (45 with the knife set): 2 prime, 1 offal, 1-2 scraps to the fridge; bones, clothes and a phone to the back step."})
	case sim.KindVan:
		out = append(out, sim.Line{Kind: "info", Text: "VEHICLE. A victim's car at the curb is the loudest evidence there is; a worker hauls it away in an hour instead of walking it to the mall for 90 minutes with a 30% chance of a traffic camera. Also +10 attack on raids."})
	case sim.KindPassive:
		out = append(out, sim.Line{Kind: "info", Text: "UPGRADE. " + effects[f.ID]})
	}
	if e, ok := effects[f.ID]; ok && f.Kind != sim.KindPassive {
		out = append(out, sim.Line{Kind: "info", Text: e})
	}
	where := "Goes anywhere on the " + f.Level.String() + "."
	if f.Front {
		where = "Goes on the porch, out front."
	}
	if f.Draw > 0 {
		where += fmt.Sprintf(" Draws $%.2f an hour in utilities.", f.Draw)
	}
	out = append(out, sim.Line{Kind: "warn", Text: where})
	return out
}
