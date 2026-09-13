// Modals: a generic list menu with a description pane, a scrolling text
// panel, a yes/no confirm, and every screen built from them (build, cookbook,
// fridge, pantry, livestock, evidence, missions, neighborhood, hunt, options,
// help, death).
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/ui"
	"cannibalcasa/internal/world"
)

// ---------------------------------------------------------------------------
// generic menu
// ---------------------------------------------------------------------------

type item struct {
	label    string
	color    ui.Attr
	disabled bool
	header   bool
	desc     []game.Event
	onSelect func(a *app)
}

type menu struct {
	title  string
	items  []item
	sel    int
	top    int
	footer string
	onKey  func(a *app, k key) bool // extra keys; true = handled
}

func newMenu(title string, items []item) *menu {
	m := &menu{title: title, items: items, footer: "Enter select · Esc back"}
	m.skip(1)
	return m
}

func (m *menu) skip(dir int) {
	for i := 0; i < len(m.items); i++ {
		if m.sel < 0 || m.sel >= len(m.items) {
			m.sel = (m.sel + len(m.items)) % max(len(m.items), 1)
		}
		if len(m.items) == 0 || !m.items[m.sel].header {
			return
		}
		m.sel += dir
	}
}

func (m *menu) key(a *app, k key) {
	if m.onKey != nil && m.onKey(a, k) {
		return
	}
	switch k {
	case keyEsc, 'q', 'Q':
		a.modal = nil
	case keyUp, 'k', 'K':
		if len(m.items) > 0 {
			m.sel = (m.sel - 1 + len(m.items)) % len(m.items)
			m.skip(-1)
		}
	case keyDown, 'j', 'J':
		if len(m.items) > 0 {
			m.sel = (m.sel + 1) % len(m.items)
			m.skip(1)
		}
	case keyPgUp:
		m.sel = max(m.sel-8, 0)
		m.skip(1)
	case keyPgDn:
		m.sel = min(m.sel+8, max(len(m.items)-1, 0))
		m.skip(-1)
	case keyHome:
		m.sel = 0
		m.skip(1)
	case keyEnd:
		m.sel = max(len(m.items)-1, 0)
		m.skip(-1)
	case keyEnter, ' ':
		if m.sel < len(m.items) {
			it := m.items[m.sel]
			if it.onSelect != nil && !it.disabled && !it.header {
				it.onSelect(a)
			}
		}
	}
}

// frame computes the centered modal rectangle.
func frame(b *ui.Buffer, wantW, wantH int) (x, y, w, h int) {
	w = min(wantW, b.W-2)
	h = min(wantH, b.H-2)
	x = (b.W - w) / 2
	y = (b.H - h) / 2
	return
}

func (m *menu) draw(a *app, b *ui.Buffer) {
	descH := 0
	if m.sel < len(m.items) && len(m.items[m.sel].desc) > 0 {
		descH = 5
	}
	listH := min(len(m.items), b.H-6-descH)
	if listH < 1 {
		listH = 1
	}
	x, y, w, h := frame(b, 70, listH+descH+3)
	if descH > 0 {
		descH = h - 3 - listH
		if descH < 2 {
			descH = 2
			listH = h - 3 - descH
		}
	}
	b.Fill(x, y, w, h, ' ', ui.White, ui.Black)
	b.Box(x, y, w, h, ui.Col("border"), ui.Black, m.title, ui.Col("title"))
	if m.sel < m.top {
		m.top = m.sel
	}
	if m.sel >= m.top+listH {
		m.top = m.sel - listH + 1
	}
	for i := 0; i < listH; i++ {
		idx := m.top + i
		if idx >= len(m.items) {
			break
		}
		it := m.items[idx]
		fg, bg := it.color, ui.Black
		if fg == (ui.Attr{}) {
			fg = ui.Col("text")
		}
		if it.disabled {
			fg = ui.Col("dim")
		}
		if it.header {
			fg = ui.Col("title")
		}
		prefix := "  "
		if idx == m.sel {
			fg, bg = ui.Col("sel_fg"), ui.Col("sel_bg")
			prefix = "> "
		}
		b.PrintN(x+1, y+1+i, prefix+trunc(it.label, w-4), w-2, fg, bg)
	}
	if len(m.items) == 0 {
		b.Print(x+3, y+1, "(nothing here)", ui.Col("dim"), ui.Black)
	}
	if listH < len(m.items) {
		b.Print(x+w-8, y+h-1, fmt.Sprintf(" %d/%d ", m.sel+1, len(m.items)), ui.Col("dim"), ui.Black)
	}
	if descH > 0 {
		dy := y + 1 + listH
		for i := 0; i < w-2; i++ {
			b.Set(x+1+i, dy, '─', ui.Col("border"), ui.Black)
		}
		lines := 0
		for _, e := range m.items[m.sel].desc {
			for _, l := range wrap(e.Text, w-4) {
				if lines >= descH-1 {
					break
				}
				b.PrintN(x+2, dy+1+lines, l, w-4, kindColor(e.Kind), ui.Black)
				lines++
			}
		}
	}
	b.Print(x+2, y+h-1, " "+m.footer+" ", ui.Col("dim"), ui.Black)
}

// ---------------------------------------------------------------------------
// text panel and confirm
// ---------------------------------------------------------------------------

type panel struct {
	title  string
	lines  []game.Event
	scroll int
	onDone func(a *app)
	footer string
}

func textPanel(title string, lines []game.Event) *panel {
	return &panel{title: title, lines: lines, footer: "Esc close"}
}

func (p *panel) key(a *app, k key) {
	switch k {
	case keyUp, 'k':
		if p.scroll > 0 {
			p.scroll--
		}
	case keyDown, 'j':
		p.scroll++
	case keyPgUp:
		p.scroll = max(p.scroll-10, 0)
	case keyPgDn:
		p.scroll += 10
	default:
		a.modal = nil
		if p.onDone != nil {
			p.onDone(a)
		}
	}
}

func (p *panel) draw(a *app, b *ui.Buffer) {
	x, y, w, h := frame(b, 72, b.H-2)
	var rows []game.Event
	for _, e := range p.lines {
		if e.Text == "" {
			rows = append(rows, e)
			continue
		}
		for _, l := range wrap(e.Text, w-4) {
			rows = append(rows, game.Event{Kind: e.Kind, Text: l})
		}
	}
	if len(rows)+2 < h {
		h = len(rows) + 2
		y = (b.H - h) / 2
	}
	if p.scroll > max(len(rows)-(h-2), 0) {
		p.scroll = max(len(rows)-(h-2), 0)
	}
	b.Fill(x, y, w, h, ' ', ui.White, ui.Black)
	b.Box(x, y, w, h, ui.Col("border"), ui.Black, p.title, ui.Col("title"))
	for i := 0; i < h-2; i++ {
		idx := p.scroll + i
		if idx >= len(rows) {
			break
		}
		b.PrintN(x+2, y+1+i, rows[idx].Text, w-4, kindColor(rows[idx].Kind), ui.Black)
	}
	if len(rows) > h-2 {
		b.Print(x+w-12, y+h-1, " ↑↓ more ", ui.Col("dim"), ui.Black)
	}
	b.Print(x+2, y+h-1, " "+p.footer+" ", ui.Col("dim"), ui.Black)
}

func confirm(title, text string, yes func(a *app)) *menu {
	m := newMenu(title, []item{
		{label: "Yes", color: ui.Col("good"), desc: []game.Event{{Kind: "info", Text: text}}, onSelect: func(a *app) { a.modal = nil; yes(a) }},
		{label: "No", desc: []game.Event{{Kind: "info", Text: text}}, onSelect: func(a *app) { a.modal = nil }},
	})
	return m
}

// ---------------------------------------------------------------------------
// build
// ---------------------------------------------------------------------------

func (a *app) openBuild(focus string) {
	s := a.st
	var items []item
	sel := -1
	for l := game.Level(0); l < game.NumLevels; l++ {
		items = append(items, item{label: strings.ToUpper(l.String()), header: true})
		for _, f := range game.Fixtures {
			if f.Level != l || s.Has(f.ID) {
				continue
			}
			f := f
			it := item{label: fmt.Sprintf("$%-4d %s", f.Cost, f.Name), color: ui.Col(f.Color)}
			desc := []game.Event{{Kind: "info", Text: f.Desc}}
			switch {
			case f.Prereq != "" && !s.Has(f.Prereq):
				it.disabled = true
				desc = append(desc, game.Event{Kind: "warn", Text: "Needs the " + game.FixtureByID(f.Prereq).Name + " first."})
			case s.Cash < f.Cost:
				it.disabled = true
				desc = append(desc, game.Event{Kind: "warn", Text: fmt.Sprintf("Costs $%d; you have $%d.", f.Cost, s.Cash)})
			default:
				desc = append(desc, game.Event{Kind: "cash", Text: fmt.Sprintf("Costs $%d of your $%d.", f.Cost, s.Cash)})
			}
			it.desc = desc
			it.onSelect = func(a *app) {
				if a.do(game.Action{Kind: "build", Name: f.ID}) {
					a.modal = nil
				}
			}
			if f.ID == focus {
				sel = len(items)
			}
			items = append(items, it)
		}
	}
	m := newMenu(fmt.Sprintf("Build ─ $%d", s.Cash), items)
	if sel >= 0 {
		m.sel = sel
	}
	a.modal = m
}

// ---------------------------------------------------------------------------
// kitchen
// ---------------------------------------------------------------------------

func cutsText(rc *game.Recipe) string {
	if rc.Bones {
		return "1 bones (evidence)"
	}
	var parts []string
	for _, k := range []string{game.CutPrime, game.CutOffal, game.CutScraps} {
		if n := rc.Cuts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, k))
		}
	}
	return strings.Join(parts, ", ")
}

func (a *app) openCookbook() {
	s := a.st
	var items []item
	for _, rc := range game.Recipes {
		rc := rc
		can := s.CanCook(&rc)
		it := item{label: rc.Name}
		if rc.Food {
			it.color = ui.Col("food")
		} else {
			it.color = ui.Col("cash")
		}
		it.disabled = !can
		desc := []game.Event{{Kind: "dim", Text: rc.Flavor}}
		var need []string
		for _, n := range rc.Needs {
			name := game.FixtureByID(n).Name
			if !s.Has(n) {
				name = "[" + name + "]"
			}
			need = append(need, name)
		}
		info := fmt.Sprintf("Uses: %s.", cutsText(&rc))
		if len(need) > 0 {
			info += " Needs: " + strings.Join(need, ", ") + "."
		}
		desc = append(desc, game.Event{Kind: "info", Text: info})
		stat := fmt.Sprintf("Base %s  hunger +%d  keeps %dd", strings.Repeat("*", rc.Base), rc.Restore, rc.Shelf)
		if rc.Price > 0 {
			stat += fmt.Sprintf("  sells $%d", rc.Price)
		}
		if !rc.Food {
			stat = fmt.Sprintf("Not food. Sells $%d at the market or on Etsy.", rc.Price)
		}
		desc = append(desc, game.Event{Kind: "cash", Text: stat})
		it.desc = desc
		it.onSelect = func(a *app) { a.do(game.Action{Kind: "cook", Name: rc.ID}); a.refreshMenu(a.openCookbook) }
		items = append(items, it)
	}
	a.modal = newMenu("Cookbook ─ fridge "+a.fridgeSummary(), items)
}

func (a *app) fridgeSummary() string {
	n := map[string]int{}
	for _, c := range a.st.Fridge {
		n[c.Kind]++
	}
	return fmt.Sprintf("%dp %do %ds", n[game.CutPrime], n[game.CutOffal], n[game.CutScraps])
}

// refreshMenu rebuilds a menu after an action, keeping the cursor row.
func (a *app) refreshMenu(open func()) {
	sel := 0
	if m, ok := a.modal.(*menu); ok {
		sel = m.sel
	}
	open()
	if m, ok := a.modal.(*menu); ok {
		m.sel = min(sel, max(len(m.items)-1, 0))
		m.skip(-1)
	}
}

func (a *app) openFridge() {
	s := a.st
	var items []item
	for _, c := range s.Fridge {
		c := c
		v := game.VictimByID(c.Victim)
		state := fmt.Sprintf("%dd left", c.Freshness)
		col := ui.Col("food")
		if c.Frozen {
			state = "FROZEN"
			col = ui.Col("fix_cold")
		} else if c.Freshness <= 1 {
			col = ui.Col("warn")
		}
		it := item{label: fmt.Sprintf("%-6s %-28s marbling %-5s %s", c.Kind, trunc(v.Name, 28), strings.Repeat("*", c.Marbling), state), color: col}
		it.desc = []game.Event{{Kind: "dim", Text: v.Flavor}}
		if c.Frozen {
			it.desc = append(it.desc, game.Event{Kind: "info", Text: "Enter: thaw (2 days to use it, no star penalty once thawed... the pan disagrees)."})
			it.onSelect = func(a *app) { a.do(game.Action{Kind: "thaw", ID: c.ID}); a.refreshMenu(a.openFridge) }
		} else {
			it.desc = append(it.desc, game.Event{Kind: "info", Text: "Enter: freeze it. Frozen cuts never spoil but cook one star lower."})
			it.onSelect = func(a *app) { a.do(game.Action{Kind: "freeze", ID: c.ID}); a.refreshMenu(a.openFridge) }
		}
		items = append(items, it)
	}
	fresh, frozen := 0, 0
	for _, c := range s.Fridge {
		if c.Frozen {
			frozen++
		} else {
			fresh++
		}
	}
	m := newMenu(fmt.Sprintf("Fridge %d/%d ─ Freezer %d/%d", fresh, s.FridgeCap(), frozen, s.FreezerCap()), items)
	m.footer = "Enter freeze/thaw · C cook · Esc back"
	m.onKey = func(a *app, k key) bool {
		if k == 'c' || k == 'C' {
			a.openCookbook()
			return true
		}
		return false
	}
	a.modal = m
}

func (a *app) openPantry() {
	s := a.st
	var items []item
	for _, d := range s.Dishes {
		d := d
		rc := game.RecipeByID(d.Recipe)
		col := ui.Col("food")
		if !rc.Food {
			col = ui.Col("cash")
		}
		it := item{label: fmt.Sprintf("%-34s %s  %dd", trunc(rc.Name, 34), strings.Repeat("*", d.Quality)+strings.Repeat("-", 5-d.Quality), d.Shelf), color: col}
		it.desc = []game.Event{{Kind: "dim", Text: rc.Flavor}}
		if rc.Food {
			it.desc = append(it.desc, game.Event{Kind: "info", Text: fmt.Sprintf("Eat: hunger +%d, stars +%d. Or [S]end it to a neighbor as a casserole.", rc.Restore+5*(d.Quality-3), max(d.Quality-2, 0))})
			it.onSelect = func(a *app) { a.do(game.Action{Kind: "eat", ID: d.ID}); a.refreshMenu(a.openPantry) }
		} else {
			it.desc = append(it.desc, game.Event{Kind: "cash", Text: fmt.Sprintf("Sells for about $%d at the Farmers Market (M), or on Etsy by itself.", rc.Price)})
		}
		items = append(items, it)
	}
	m := newMenu(fmt.Sprintf("Pantry ─ hunger %d", s.Hunger), items)
	m.footer = "Enter eat · S send casserole · Esc back"
	m.onKey = func(a *app, k key) bool {
		if (k == 's' || k == 'S') && m.sel < len(s.Dishes) {
			d := s.Dishes[m.sel]
			if !game.RecipeByID(d.Recipe).Food {
				a.say("warn", "That's not a casserole. That's soap.")
				return true
			}
			a.openNeighborPicker("Send "+game.RecipeByID(d.Recipe).Name+" to", func(a *app, n world.Neighbor) {
				st, r, err := a.be.Casserole(a.bbs, a.handle, n.Key, d.ID)
				a.absorb(st, r, err)
				a.modal = nil
			})
			return true
		}
		return false
	}
	a.modal = m
}

// ---------------------------------------------------------------------------
// basement
// ---------------------------------------------------------------------------

func (a *app) openLivestock() {
	s := a.st
	var items []item
	for _, c := range s.Captives {
		c := c
		v := game.VictimByID(c.Victim)
		label := fmt.Sprintf("%-10s %-30s marb %-5s", c.Name, trunc(v.Name, 30), strings.Repeat("*", c.Marbling))
		col := ui.Col(v.Color)
		if c.Renfield {
			label = fmt.Sprintf("%-10s RENFIELD (%s) loyalty %d", c.Name, trunc(v.Name, 22), c.Loyalty)
			col = ui.Col("fix_closet")
		} else if c.Hungry > 0 {
			label += fmt.Sprintf(" hungry %dd", c.Hungry)
		}
		it := item{label: label, color: col}
		it.desc = []game.Event{{Kind: "dim", Text: v.Flavor}}
		if c.Renfield {
			it.desc = append(it.desc, game.Event{Kind: "info", Text: "Minds the house while you're out. Loyalty drifts; a TV helps. Enter butchers them (loyalty is delicious)."})
		} else {
			it.desc = append(it.desc, game.Event{Kind: "info", Text: fmt.Sprintf("Day %d in the cell. Enter: butcher. W: brainwash into a Renfield (needs The Chair).", c.Days)})
		}
		it.onSelect = func(a *app) {
			a.modal = confirm("Butcher "+c.Name+"?", "Yields cuts for the fridge and leaves bones, clothes and a phone behind.",
				func(a *app) { a.do(game.Action{Kind: "butcher", ID: c.ID}); a.openLivestock() })
		}
		items = append(items, it)
	}
	m := newMenu(fmt.Sprintf("Livestock %d/%d cells ─ Renfields %d/%d", s.Livestock(), s.CellCount(), s.Renfields(), s.RenfieldCap()), items)
	m.footer = "Enter butcher · W brainwash · Esc back"
	m.onKey = func(a *app, k key) bool {
		if (k == 'w' || k == 'W') && m.sel < len(s.Captives) {
			a.do(game.Action{Kind: "brainwash", ID: s.Captives[m.sel].ID})
			a.refreshMenu(a.openLivestock)
			return true
		}
		return false
	}
	a.modal = m
}

// ---------------------------------------------------------------------------
// evidence
// ---------------------------------------------------------------------------

var methodNames = map[string]string{
	"bury": "Bury in the backyard", "chipper": "Wood chipper (loud)", "lye": "Lye bath",
	"microwave": "Microwave it", "donate": "Donate to Goodwill", "compost": "Compost (tomatoes!)",
	"van": "Haul it away in the van", "abandon": "Abandon it at the mall",
}

func (a *app) openEvidence() {
	s := a.st
	var items []item
	for _, e := range s.Evidence {
		e := e
		v := game.VictimByID(e.Victim)
		state := "LOOSE"
		col := ui.Col("heat")
		if e.Buried {
			state = "buried"
			col = ui.Col("dim")
		}
		it := item{label: fmt.Sprintf("%-8s %-30s %2dd old  %s", e.Kind, trunc(v.Name, 30), e.Age, state), color: col}
		methods := s.DisposalMethods(e)
		var names []string
		for _, m := range methods {
			names = append(names, methodNames[m])
		}
		if e.Buried {
			it.desc = []game.Event{{Kind: "info", Text: "Underground. Quiet unless the neighbor's dog gets curious. A fence fixes that."}}
			it.disabled = true
		} else if len(methods) == 0 {
			it.desc = []game.Event{{Kind: "warn", Text: "No way to dispose of this yet. Bones: stockpot, compost, chipper, lye or a free plot. Phones: microwave."}}
			it.disabled = true
		} else {
			it.desc = []game.Event{{Kind: "info", Text: "Options: " + strings.Join(names, ", ") + "."}}
		}
		it.onSelect = func(a *app) {
			var opts []item
			for _, m := range methods {
				m := m
				opts = append(opts, item{label: methodNames[m], onSelect: func(a *app) {
					a.do(game.Action{Kind: "dispose", ID: e.ID, Name: m})
					a.openEvidence()
				}})
			}
			a.modal = newMenu("Dispose of the "+e.Kind, opts)
		}
		items = append(items, it)
	}
	m := newMenu(fmt.Sprintf("Evidence ─ plots %d/%d ─ visible score %d", s.Buried(), s.PlotCap(), s.VisibleEvidence()), items)
	a.modal = m
}

// ---------------------------------------------------------------------------
// missions, hunting, dinner
// ---------------------------------------------------------------------------

func (a *app) openMissions() {
	s := a.st
	var items []item
	for _, m := range game.Missions {
		m := m
		it := item{label: fmt.Sprintf("%-24s %d phase(s)", m.Name, m.Phases), color: ui.Col("warn")}
		it.desc = []game.Event{{Kind: "info", Text: m.Desc}}
		if s.Tier() < m.MinTier {
			it.disabled = true
			it.desc = append(it.desc, game.Event{Kind: "warn", Text: "Not until the street needs a hero."})
		}
		it.onSelect = func(a *app) {
			if a.do(game.Action{Kind: "mission", Name: m.ID}) {
				a.modal = nil
			}
		}
		items = append(items, it)
	}
	guard := "Nobody is watching the house while you're out."
	if s.Renfields() > 0 || s.Has("cameras") {
		guard = fmt.Sprintf("%d Renfield(s) and %s keep an eye on things.", s.Renfields(), map[bool]string{true: "the cameras", false: "no cameras"}[s.Has("cameras")])
	}
	m := newMenu("Leave the house ─ "+guard, items)
	a.modal = m
}

func (a *app) openHunt() {
	s := a.st
	if s.Phase != game.Night {
		a.say("warn", "Hunting is for the Night. Pass the time with T.")
		return
	}
	if len(s.Prey) == 0 {
		a.say("warn", "The street is empty tonight. Sleep it off.")
		return
	}
	var items []item
	for _, id := range s.Prey {
		v := game.VictimByID(id)
		id := id
		it := item{label: fmt.Sprintf("%-34s %3d%%  marbling %-5s fight %d", trunc(v.Name, 34), s.HuntChance(id), strings.Repeat("*", v.Marbling), v.Fight), color: ui.Col(v.Color)}
		it.desc = []game.Event{{Kind: "dim", Text: v.Flavor},
			{Kind: "heat", Text: fmt.Sprintf("Missed after %d days (Heat %+d). %s", v.Notice, v.HeatMissing, map[bool]string{true: "Drives a car.", false: ""}[v.Drives])}}
		it.onSelect = func(a *app) {
			a.do(game.Action{Kind: "hunt", Name: id})
			a.modal = nil
		}
		items = append(items, it)
	}
	if s.Livestock() >= s.CellCount() {
		items = append(items, item{label: "(cells are full: a catch goes straight to the kitchen)", header: true})
	}
	a.modal = newMenu("Out walking tonight", items)
}

func (a *app) openDinner() {
	s := a.st
	if !s.Has("dining_set") {
		a.say("warn", "You need a Dining Set before anyone comes over. Build one in the living room.")
		return
	}
	a.modal = confirm("Host a dinner party?",
		fmt.Sprintf("Serves your best dish (3 stars or better) to the neighbors. Heat drops, stars rise, and the Evening is spent. It is %s.", s.Phase),
		func(a *app) { a.do(game.Action{Kind: "dinner"}) })
}

// ---------------------------------------------------------------------------
// neighborhood
// ---------------------------------------------------------------------------

func (a *app) openNeighborPicker(title string, pick func(a *app, n world.Neighbor)) {
	hood, err := a.be.Neighborhood(a.bbs, a.handle)
	if err != nil {
		a.say("bad", "Couldn't reach the street: %v", err)
		return
	}
	me := world.Key(a.bbs, a.handle)
	var items []item
	for _, n := range hood.Neighbors {
		n := n
		if n.Key == me {
			continue
		}
		items = append(items, item{label: fmt.Sprintf("%-24s %s@%s", trunc(n.Cannibal, 24), n.Handle, n.BBS), color: ui.Col("text"),
			onSelect: func(a *app) { pick(a, n) }})
	}
	a.modal = newMenu(title, items)
}

func (a *app) openNeighborhood() {
	hood, err := a.be.Neighborhood(a.bbs, a.handle)
	if err != nil {
		a.say("bad", "Couldn't reach the street: %v", err)
		return
	}
	me := world.Key(a.bbs, a.handle)
	var items []item
	for _, n := range hood.Neighbors {
		n := n
		status := fmt.Sprintf("gen %d  day %-3d  %3d stars  %s", n.Generation, n.Day, n.Stars, game.TierName(game.HeatTier(n.Heat)))
		col := ui.Col("text")
		if !n.Alive {
			status = "deceased (awaiting an heir)"
			col = ui.Col("dim")
		}
		label := fmt.Sprintf("%-22s %s", trunc(n.Cannibal, 22), status)
		if n.Key == me {
			label = "* " + label
			col = ui.Col("bright")
		}
		it := item{label: label, color: col}
		it.desc = []game.Event{{Kind: "info", Text: fmt.Sprintf("%s on %s. %d livestock, %d Renfields, %d meals served.", n.Handle, n.BBS, n.Livestock, n.Renfields, n.Meals)}}
		if n.Key != me {
			it.desc = append(it.desc, game.Event{Kind: "dim", Text: "Enter: anonymous tip to the HOA (+10 Heat for them, may backfire). Casseroles go from the Pantry (E, then S)."})
			it.onSelect = func(a *app) {
				a.modal = confirm("Call the HOA about "+n.Cannibal+"?", "One tip per day. Caller ID exists. Heat +10 for them; 35% chance of +8 for you.",
					func(a *app) {
						st, r, err := a.be.Tip(a.bbs, a.handle, n.Key)
						a.absorb(st, r, err)
					})
			}
		}
		items = append(items, it)
	}
	m := newMenu(fmt.Sprintf("The cul-de-sac ─ %d households", len(hood.Neighbors)), items)
	m.footer = "Enter tip · V feed · Esc back"
	m.onKey = func(a *app, k key) bool {
		if k == 'v' || k == 'V' {
			a.openFeed(hood)
			return true
		}
		return false
	}
	a.modal = m
}

func (a *app) openFeed(hood world.Neighborhood) {
	var lines []game.Event
	for i := len(hood.Feed) - 1; i >= 0; i-- {
		e := hood.Feed[i]
		lines = append(lines, game.Event{Kind: e.Kind, Text: fmt.Sprintf("%s %s", e.Cannibal, e.Text)})
	}
	if len(lines) == 0 {
		lines = append(lines, game.Event{Kind: "dim", Text: "Nothing posted yet. The street is suspiciously quiet."})
	}
	p := textPanel("Nextdoor ─ newest first", lines)
	p.footer = "↑↓ scroll · any key closes"
	a.modal = p
}

// ---------------------------------------------------------------------------
// options, help, death
// ---------------------------------------------------------------------------

func (a *app) openOptions() {
	mode := "16-color"
	if a.buf.Mode == ui.Mode256 {
		mode = "256-color"
	}
	var anc []game.Event
	for _, an := range a.st.Ancestors {
		anc = append(anc, game.Event{Kind: "dim", Text: fmt.Sprintf("%s: %d days, %d meals, %d stars. %s.", an.Name, an.Days, an.Meals, an.Stars, an.Cause)})
	}
	if len(anc) == 0 {
		anc = append(anc, game.Event{Kind: "dim", Text: "No one has died in this house. Yet."})
	}
	items := []item{
		{label: "Colors: " + mode, desc: []game.Event{{Kind: "info", Text: "Toggle between classic 16-color (every terminal) and 256-color (richer hues on SyncTERM, NetRunner, most modern terminals)."}},
			onSelect: func(a *app) {
				if a.buf.Mode == ui.Mode256 {
					a.buf.Mode = ui.Mode16
					a.do(game.Action{Kind: "set_color", Name: "16"})
				} else {
					a.buf.Mode = ui.Mode256
					a.do(game.Action{Kind: "set_color", Name: "256"})
				}
				a.buf.Invalidate()
				a.openOptions()
			}},
		{label: "Rename " + a.st.Cannibal, desc: []game.Event{{Kind: "info", Text: "Type a new name for the current cannibal."}},
			onSelect: func(a *app) {
				a.modal = newPrompt("New name", a.st.Cannibal, func(a *app, v string) { a.do(game.Action{Kind: "rename", Name: v}) })
			}},
		{label: "Family history", desc: anc, onSelect: func(a *app) { a.modal = textPanel("The dynasty", anc) }},
	}
	a.modal = newMenu("Options", items)
}

func (a *app) openHelp() {
	lines := []game.Event{
		{Kind: "title", Text: "You are a cannibal in the suburbs. Stay fed, stay unnoticed, upgrade the house."},
		{Kind: "info", Text: ""},
		{Kind: "good", Text: "THE DAY  Morning, Afternoon, Evening, Night. T passes a phase. Sleeping through Night rolls the day and spends a banked day; you get a few per real day."},
		{Kind: "food", Text: "FOOD  Hunt at Night (H). Captives wait in cells (L), fattening if fed ($4/day). Butcher for cuts (fridge, F). Cook (C) at any kitchen fixture. Eat from the pantry (E). Meat spoils in days; freeze it or lose it."},
		{Kind: "heat", Text: "HEAT  Loose evidence, noisy livestock, missing-person reports and escapes raise it. Dispose (X): bury, microwave, compost, chipper, lye, bone broth. Fences, curtains and dinner parties cool it (P hosts a party). Crossing a tier brings a visitor: tidy up before the knock. 100 = the raid."},
		{Kind: "cash", Text: "CASH  Remote Work and the Farmers Market (M) pay. Pies, jerky, broth and soap sell. A tomato garden and an Airbnb listing pay daily."},
		{Kind: "fix_closet", Text: "RENFIELDS  The Chair brainwashes a captive into a helper who frees a cell and guards the house during missions. Loyalty drifts; a big TV helps."},
		{Kind: "warn", Text: "MISSIONS  Leave the house for a few phases. Unwatched livestock escape; the dog digs. Cameras and Renfields make it safe."},
		{Kind: "info", Text: "THE STREET (N)  Every player lives on the cul-de-sac. Read the feed, send a casserole (E then S), or call the HOA on a neighbor."},
		{Kind: "bad", Text: "DEATH  Getting raided or starving ends this cannibal. A relative inherits the house: fixtures and recipes stay, cash halves, the freezer empties."},
		{Kind: "info", Text: ""},
		{Kind: "dim", Text: "Keys: arrows/WASD move · Enter use · Tab floor · T next · B build · C cook · E eat · F fridge · L livestock · X evidence · M missions · N street · H hunt · P party · O options · Q quit"},
	}
	a.modal = textPanel("How to Cannibal", lines)
}

func deathScreen(a *app) *panel {
	s := a.st
	lines := []game.Event{
		{Kind: "bad", Text: fmt.Sprintf("%s: %s.", s.Cannibal, s.Cause)},
		{Kind: "info", Text: ""},
		{Kind: "info", Text: fmt.Sprintf("%d days on the cul-de-sac. %d meals. %d stars. %d guests captured, %d escaped, %d dinner parties.", s.Day, s.Meals, s.Stars, s.Stats.Captured, s.Stats.Escaped, s.Stats.Dinners)},
		{Kind: "info", Text: ""},
		{Kind: "warn", Text: "A relative will inherit the house. Fixtures and recipes stay; cash is halved; the basement and the freezer come empty; the neighbors come curious."},
		{Kind: "dim", Text: "Press any key to read the will."},
	}
	p := textPanel("In Memoriam", lines)
	p.footer = "any key"
	p.onDone = func(a *app) {
		if a.do(game.Action{Kind: "inherit"}) {
			a.seenDeath = false
			a.level = game.Ground
			a.px, a.py = a.plan().Spawn()
		}
	}
	return p
}

// ---------------------------------------------------------------------------
// a one-line text prompt
// ---------------------------------------------------------------------------

type prompt struct {
	title string
	value []rune
	done  func(a *app, v string)
}

func newPrompt(title, initial string, done func(a *app, v string)) *prompt {
	return &prompt{title: title, value: []rune(initial), done: done}
}

func (p *prompt) key(a *app, k key) {
	switch {
	case k == keyEsc:
		a.modal = nil
	case k == keyEnter:
		a.modal = nil
		p.done(a, strings.TrimSpace(string(p.value)))
	case k == keyBack:
		if len(p.value) > 0 {
			p.value = p.value[:len(p.value)-1]
		}
	case k > 0 && k < 0x7f && len(p.value) < 28:
		p.value = append(p.value, rune(k))
	}
}

func (p *prompt) draw(a *app, b *ui.Buffer) {
	x, y, w, h := frame(b, 44, 5)
	b.Fill(x, y, w, h, ' ', ui.White, ui.Black)
	b.Box(x, y, w, h, ui.Col("border"), ui.Black, p.title, ui.Col("title"))
	b.PrintN(x+2, y+2, string(p.value)+"_", w-4, ui.Col("bright"), ui.Black)
	b.Print(x+2, y+h-1, " Enter ok · Esc cancel ", ui.Col("dim"), ui.Black)
}
