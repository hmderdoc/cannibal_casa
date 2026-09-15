// The player guide: a menu of sections, each a panel. Reachable with ? or H.
package main

import (
	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/ui"
)

type section struct {
	title string
	lines []sim.Line
}

var guide = []section{
	{"What is this", []sim.Line{
		{Kind: "title", Text: "The house is a machine. Feed it people, keep it quiet, keep everyone fed."},
		{Kind: "info", Text: "You don't walk around. You move a cursor, look at things (Enter), build and move machines, set policy, and send yourself out on errands. Everything else happens on its own, on one clock that keeps running after you log off."},
		{Kind: "info", Text: "Winning is surviving and eating well: stars come from good dishes and dinner parties. Losing is the raid (Heat 100) or starving. A relative then inherits the house."},
	}},
	{"Money", []sim.Line{
		{Kind: "cash", Text: "WALLETS. Every captive who drops through the mat carries cash. Tech bros, landlords and tourists carry a lot; joggers carry nothing."},
		{Kind: "cash", Text: "THE STAND. Policy marks recipes for sale; workers cook them once the shelf is stocked and carry them to the curb stand, which sells one an hour, nine to six. Fritters need no gear; pies need the grinder ($80) and pay better."},
		{Kind: "cash", Text: "GO OUT (G). A shift at the mall pays $40-60 in four hours. A market run sells everything marked for sale at once. Scavenging finds a little. Raiding a house finds more, or a doorbell camera."},
		{Kind: "cash", Text: "PASSIVE. A tomato garden pays daily (more per bag composted). An Airbnb listing pays $15 a day. Soap molds sell a bar on Etsy every day."},
		{Kind: "warn", Text: "COSTS. Feed is $4 a head a day ($2 with the gruel dispenser). Appliances draw a little power every hour. Spending everything on day one is the classic mistake: keep $50 for feed and go to work."},
	}},
	{"Intake", []sim.Line{
		{Kind: "food", Text: "Passers-by walk the sidewalk, more by day, joggers and bachelorette parties at night. At the gate a lure may pull one onto the porch; the welcome mat drops them into a free cell. No free cell, no drop. The mat resets for an hour."},
		{Kind: "food", Text: "Each lure pulls its own crowd: the couch gets CrossFit guys and missionaries, the Wi-Fi sign gets tech bros and huns, the Airbnb listing gets influencers and tourists, the pool gets yoga instructors. A Shill on the porch makes every lure pull harder."},
		{Kind: "food", Text: "Fight is what beats the trap. The hydraulic trapdoor, the chemistry set and (at night) the telescope raise the odds. Or go hunting yourself (G)."},
		{Kind: "heat", Text: "Drivers leave a car at the curb. Cars are the loudest evidence there is; haul them with the van or walk them to the mall."},
	}},
	{"Livestock and the kitchen", []sim.Line{
		{Kind: "food", Text: "Cells hold one each. Fed livestock fattens a marbling point every three days (two with the trough). Policy says when to butcher: at a marbling, or when every cell is full."},
		{Kind: "food", Text: "The block takes an hour per body (faster with the knife set): cuts go to the fridge, the bag goes to the back step. Fridge cuts last three days (offal two); overflow freezes if you own a freezer. Frozen cooks a star lower."},
		{Kind: "food", Text: "Stations run one recipe for its duration and hold the dish until someone plates it. The shelf holds eight. Quality comes from marbling and freshness; five-star dishes get talked about on the street."},
		{Kind: "food", Text: "Place things close together. Workers walk, and a fridge next to the stairs saves an hour a day."},
	}},
	{"Evidence and Heat", []sim.Line{
		{Kind: "heat", Text: "A body leaves bones, clothes and a phone on the back step. Every bag radiates Heat hourly; the phone pings hard for three days. Disposal: chipper (5 minutes, loud), lye tub (12 hours, silent), compost (a day, silent, feeds the garden), plots (a worker digs), the stockpot (bones into broth), the microwave (phones)."},
		{Kind: "heat", Text: "Heat cools every hour, faster with a fence and curtains, fastest when nothing is visible. Missing-person reports land once per captive after their notice period. Failed traps and loud machines spike it."},
		{Kind: "heat", Text: "Crossing a tier sends a visitor to the front door. They score what they can see: cars, bags without a fence, screaming without soundproofing (or a Gimp). Pass and Heat drops; fail and it jumps. 100 is the raid."},
	}},
	{"slaves and converts", []sim.Line{
		{Kind: "fix_closet", Text: "The Chair (six hours, three with the tapes) turns a captive into a slave: a worker who needs a cot, food and a TV to stay loyal. Loyalty drifts down; at zero they walk to the police."},
		{Kind: "fix_closet", Text: "Roles (Roster, L): H Hand does everything; C Cook keeps the kitchen moving; X Guard patrols the cells (escapes drop to a third, raiders get caught); S Shill works the porch (lures pull harder); G Gimp lives in the basement (the livestock goes quiet and stops trying the door); K Clerk works the mall booth. The letter is their map symbol; the color is theirs alone."},
		{Kind: "fix_closet", Text: "A slave who eats six dishes from your shelf knows what they are and asks for seconds. A fellow cannibal: loyal for life, eats half again as much, and goes hunting at night on their own."},
	}},
	{"The mall", []sim.Line{
		{Kind: "cash", Text: "Tab past the attic and you're at the mall: stores off a concourse, customers all day. Buy a booth on the concourse and they buy what's stocked at mall prices. A kiosk holds more and sells better; a storefront sells to a queue."},
		{Kind: "cash", Text: "Give a slave the Clerk role and they carry stock over from the shelf and mind the counter. A shift at the food court (Go out) now walks you there for real; watch yourself work."},
		{Kind: "heat", Text: "Every sale, at home or at the mall, makes you a little more respectable: Heat cools with commerce."},
	}},
	{"Heat: the levers", []sim.Line{
		{Kind: "heat", Text: "Policy: Intake. Set it to 'only when the fridge runs low' and the mat stays quiet while you're stocked; set it to 'never' to lay low completely."},
		{Kind: "heat", Text: "Go out: volunteer at the bake sale (two dishes), drop a check at the HOA ($200), or throw a block party ($150, three dishes, once a week). Money and food become goodwill."},
		{Kind: "heat", Text: "Upkeep: a landscaping service and a PR firm on retainer cool the street every hour for as long as you pay them; a lawyer on retainer makes inspections overlook more."},
		{Kind: "heat", Text: "Plots: Enter on a flower bed to dig a bag back up and destroy it properly. Burials older than a month become flowers on their own."},
	}},
	{"Going out and raids", []sim.Line{
		{Kind: "warn", Text: "G sends you out. The house keeps running on your policy; slaves keep working; livestock still eats. You come home through the front door."},
		{Kind: "warn", Text: "Raiding another house on the street is settled by their defenses (cameras, fence, doorbell camera, guards, anyone awake) against your gear (van, chemistry set). Win: 15% of their cash and a few things from the kitchen. Lose: Heat +10. Once per house per real day. They hear about it either way."},
		{Kind: "warn", Text: "Your own house can be raided the same way. Cameras, a fence and a Guard are the answer."},
	}},
	{"Keys", []sim.Line{
		{Kind: "dim", Text: "Arrows / WASD: cursor.  Enter: look at / open whatever is under it.  Tab: next floor.  Space: pause.  1 2 3 4: speed 1x 2x 4x 8x."},
		{Kind: "dim", Text: "B build (then place with the cursor).  G go out.  P policy.  L roster and roles.  I inventory.  N the street.  J journal order.  O options.  ? this guide.  Q quit."},
		{Kind: "dim", Text: "On a machine, Enter offers a switch, upgrades, and Move it. Moving is free and instant; workers re-path."},
	}},
}

func (a *app) openHelp() {
	var items []item
	for _, s := range guide {
		s := s
		items = append(items, item{label: s.title, color: ui.Col("text"), desc: s.lines[:1], onSelect: func(a *app) {
			p := textPanel(s.title, s.lines)
			p.onDone = func(a *app) { a.openHelp() }
			a.modal = p
		}})
	}
	a.modal = newMenu("Guide ─ How to Cannibal", items)
}
