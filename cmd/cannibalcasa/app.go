// The door's application state and key handling. The app owns the player's
// State (as last returned by the backend), the caller's position on the
// current floor, the journal, and at most one modal (a menu or text panel).
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/ui"
	"cannibalcasa/internal/world"
)

type modal interface {
	key(a *app, k key)
	draw(a *app, b *ui.Buffer)
}

type app struct {
	be        world.Backend
	bbs       string
	handle    string
	st        *game.State
	buf       *ui.Buffer
	cfg       config
	level     game.Level
	px, py    int
	log       []game.Event
	modal     modal
	quit      bool
	seenDeath bool
}

func newApp(be world.Backend, bbs, handle string, cfg config, buf *ui.Buffer) *app {
	a := &app{be: be, bbs: bbs, handle: handle, cfg: cfg, buf: buf}
	a.px, a.py = game.Plans[game.Ground].Spawn()
	return a
}

// say appends to the journal.
func (a *app) say(kind, format string, args ...any) {
	a.log = append(a.log, game.Event{Kind: kind, Text: fmt.Sprintf(format, args...)})
	if len(a.log) > 300 {
		a.log = a.log[len(a.log)-300:]
	}
}

func (a *app) absorb(st *game.State, r game.Result, err error) bool {
	if err != nil {
		a.say("bad", "Connection trouble: %v", err)
		return false
	}
	if st != nil {
		a.st = st
	}
	if r.Err != "" {
		a.say("warn", "%s", r.Err)
		return false
	}
	for _, e := range r.Events {
		a.log = append(a.log, e)
	}
	if len(a.log) > 300 {
		a.log = a.log[len(a.log)-300:]
	}
	a.checkDeath()
	return true
}

// do sends an action and folds the result into the journal.
func (a *app) do(act game.Action) bool {
	st, r, err := a.be.Do(a.bbs, a.handle, act)
	return a.absorb(st, r, err)
}

func (a *app) join() {
	st, ev, err := a.be.Join(a.bbs, a.handle)
	if err != nil {
		a.say("bad", "Could not join the neighborhood: %v", err)
		return
	}
	a.st = st
	for _, e := range st.Journal {
		if len(a.log) < 6 {
			a.log = append(a.log, game.Event{Kind: "dim", Text: e})
		}
	}
	a.log = append(a.log, ev...)
	if st.ColorMode == "256" {
		a.buf.Mode = ui.Mode256
	}
	a.checkDeath()
}

func (a *app) checkDeath() {
	if a.st != nil && !a.st.Alive && !a.seenDeath {
		a.seenDeath = true
		a.modal = deathScreen(a)
	}
}

func (a *app) plan() game.Plan { return game.Plans[a.level] }

// handleKey routes a key to the modal if one is open, else to the map.
func (a *app) handleKey(k key) {
	if a.modal != nil {
		a.modal.key(a, k)
		return
	}
	if a.st == nil {
		if k == keyEsc || k == 'q' || k == 'Q' {
			a.quit = true
		}
		return
	}
	switch k {
	case keyUp, 'w', 'W':
		a.move(0, -1)
	case keyDown, 's', 'S':
		a.move(0, 1)
	case keyLeft, 'a', 'A':
		a.move(-1, 0)
	case keyRight, 'd', 'D':
		a.move(1, 0)
	case 'h', 'H':
		a.openHunt()
	case 'l', 'L':
		a.openLivestock()
	case 'p', 'P':
		a.openDinner()
	case keyEnter, ' ':
		a.interact()
	case keyTab, '>', '.':
		a.switchLevel(1)
	case '<', ',':
		a.switchLevel(-1)
	case 't', 'T':
		a.endPhase()
	case 'b', 'B':
		a.openBuild("")
	case 'c', 'C':
		a.openCookbook()
	case 'e', 'E':
		a.openPantry()
	case 'x', 'X':
		a.openEvidence()
	case 'm', 'M':
		a.openMissions()
	case 'n', 'N':
		a.openNeighborhood()
	case 'f', 'F':
		a.openFridge()
	case 'o', 'O':
		a.openOptions()
	case '?':
		a.openHelp()
	case keyEsc, 'q', 'Q':
		a.modal = confirm("Leave the house?", "Your progress is saved. The neighbors will assume you're at work.", func(a *app) { a.quit = true })
	}
}

func (a *app) move(dx, dy int) {
	p := a.plan()
	nx, ny := a.px+dx, a.py+dy
	if !p.Walkable(nx, ny) {
		return
	}
	a.px, a.py = nx, ny
	if p.SlotAt(nx, ny) != 0 {
		a.interact()
	}
}

func (a *app) switchLevel(d int) {
	a.level = game.Level((int(a.level) + d + int(game.NumLevels)) % int(game.NumLevels))
	a.px, a.py = a.plan().Spawn()
}

// interact is bump-to-use: whatever fixture slot the caller stands on.
func (a *app) interact() {
	slot := a.plan().SlotAt(a.px, a.py)
	if slot == 0 {
		return
	}
	f := game.FixtureAtSlot(a.level, slot)
	if f == nil {
		return
	}
	if !a.st.Has(f.ID) {
		a.openBuild(f.ID)
		return
	}
	switch f.ID {
	case "stove", "grinder", "smoker", "sous_vide", "dehydrator", "blender", "stockpot", "knife_set", "wine_rack", "grill", "drying_rack", "soap_molds":
		a.openCookbook()
	case "fridge", "fridge_big", "freezer", "chest_freezer":
		a.openFridge()
	case "cell_1", "cell_2", "cell_3", "cell_4", "cell_5", "chair", "vhs", "buffet", "gruel", "soundproof", "cot":
		a.openLivestock()
	case "plot_1", "plot_2", "plot_3", "compost", "chipper", "lye_tub", "microwave", "van", "shed":
		a.openEvidence()
	case "dining_set":
		a.openDinner()
	case "garden", "lure_airbnb":
		a.modal = textPanel(f.Name, []game.Event{{Kind: "info", Text: f.Desc}, {Kind: "cash", Text: "Pays out every morning."}})
	default:
		a.modal = textPanel(f.Name, []game.Event{{Kind: "info", Text: f.Desc}})
	}
}

func (a *app) endPhase() {
	if a.st.Phase == game.Night && a.st.Mission == nil && len(a.st.Prey) > 0 {
		a.modal = confirm("Sleep through the night?",
			fmt.Sprintf("Someone's out walking. Sleeping spends 1 banked day (%d left). [H] hunts instead.", a.st.Bank),
			func(a *app) { a.do(game.Action{Kind: "end_phase"}) })
		return
	}
	a.do(game.Action{Kind: "end_phase"})
}

// tierColor picks the Heat meter color by tier.
func tierColor(heat int) ui.Attr {
	switch {
	case heat >= 60:
		return ui.Col("heat_hi")
	case heat >= 30:
		return ui.Col("heat_mid")
	}
	return ui.Col("heat_low")
}

func hungerColor(h int) ui.Attr {
	switch {
	case h <= 25:
		return ui.Col("heat_hi")
	case h <= 50:
		return ui.Col("heat_mid")
	}
	return ui.Col("hunger")
}

func kindColor(kind string) ui.Attr {
	switch kind {
	case "cash":
		return ui.Col("cash_ev")
	case "dim":
		return ui.Col("dim")
	}
	return ui.Col(kind)
}

// wrap breaks s into lines no wider than w.
func wrap(s string, w int) []string {
	if w < 8 {
		w = 8
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
			} else if len(line)+1+len(word) <= w {
				line += " " + word
			} else {
				lines = append(lines, line)
				line = word
			}
			for len(line) > w {
				lines = append(lines, line[:w])
				line = line[w:]
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "~"
}
