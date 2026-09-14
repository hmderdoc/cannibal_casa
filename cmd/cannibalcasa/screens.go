// Modals: a generic list menu with a description pane, a scrolling text
// panel, a confirm, a one-line prompt, and every screen built from them.
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/sim"
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
	desc     []sim.Line
	onSelect func(a *app)
}

type menu struct {
	title  string
	items  []item
	sel    int
	top    int
	footer string
	onKey  func(a *app, k key) bool
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
	case keyUp, 'k', 'K', 'w', 'W':
		if len(m.items) > 0 {
			m.sel = (m.sel - 1 + len(m.items)) % len(m.items)
			m.skip(-1)
		}
	case keyDown, 'j', 'J', 's', 'S':
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
		n := 0
		for _, e := range m.items[m.sel].desc {
			n += len(wrap(e.Text, 66))
		}
		descH = min(n+1, b.H/2)
		if descH < 5 {
			descH = 5
		}
	}
	listH := min(len(m.items), b.H-6-descH)
	if listH < 1 {
		listH = 1
	}
	x, y, w, h := frame(b, 72, listH+descH+3)
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
		b.Print(x+w-8, y+h-1, fmt.Sprintf(" %d/%d ", m.sel+1, len(m.items)), ui.Col("rule"), ui.Black)
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
	b.Print(x+2, y+h-1, " "+m.footer+" ", ui.Col("rule"), ui.Black)
}

// ---------------------------------------------------------------------------
// text panel, confirm, prompt
// ---------------------------------------------------------------------------

type panel struct {
	title  string
	lines  []sim.Line
	scroll int
	onDone func(a *app)
	footer string
}

func textPanel(title string, lines []sim.Line) *panel {
	return &panel{title: title, lines: lines, footer: "Esc close"}
}

func (p *panel) key(a *app, k key) {
	switch k {
	case keyUp, 'k', 'w':
		if p.scroll > 0 {
			p.scroll--
		}
	case keyDown, 'j', 's':
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
	var rows []sim.Line
	for _, e := range p.lines {
		if e.Text == "" {
			rows = append(rows, e)
			continue
		}
		for _, l := range wrap(e.Text, w-4) {
			rows = append(rows, sim.Line{Kind: e.Kind, Text: l})
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
		b.Print(x+w-12, y+h-1, " ↑↓ more ", ui.Col("rule"), ui.Black)
	}
	b.Print(x+2, y+h-1, " "+p.footer+" ", ui.Col("rule"), ui.Black)
}

func confirm(title, text string, yes func(a *app)) *menu {
	return newMenu(title, []item{
		{label: "Yes", color: ui.Col("good"), desc: []sim.Line{{Kind: "info", Text: text}}, onSelect: func(a *app) { a.modal = nil; yes(a) }},
		{label: "No", desc: []sim.Line{{Kind: "info", Text: text}}, onSelect: func(a *app) { a.modal = nil }},
	})
}

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

// ---------------------------------------------------------------------------
// fixtures and building
// ---------------------------------------------------------------------------

func (a *app) fixtureStatus(f *sim.Fixture) []sim.Line {
	w := a.w
	m := w.Machines[f.ID]
	var out []sim.Line
	out = append(out, explain(w, f)...)
	switch f.Kind {
	case sim.KindStation:
		switch {
		case m != nil && m.Busy > 0:
			out = append(out, sim.Line{Kind: "busy", Text: fmt.Sprintf("Cooking %s, %s left.", sim.RecipeByID(m.Recipe).Name, minutes(m.Busy))})
		case m != nil && m.Output != nil:
			out = append(out, sim.Line{Kind: "food", Text: fmt.Sprintf("%s is done, waiting for someone to plate it.", sim.RecipeByID(m.Output.Sub).Name)})
		default:
			out = append(out, sim.Line{Kind: "info", Text: "Idle."})
		}
		var rs []string
		for _, rc := range sim.Recipes {
			if rc.Station == f.ID {
				rs = append(rs, rc.Name)
			}
		}
		out = append(out, sim.Line{Kind: "info", Text: "Recipes here: " + strings.Join(rs, "; ") + "."})
	case sim.KindDisposal:
		if m != nil && m.Busy > 0 {
			out = append(out, sim.Line{Kind: "busy", Text: fmt.Sprintf("Working on %s, %s left.", m.Item, minutes(m.Busy))})
		} else {
			out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("Idle. Takes %s per item; accepts %s.", minutes(f.Duration), strings.Join(f.Accepts, ", "))})
		}
	case sim.KindChair:
		if m != nil && m.Busy > 0 {
			l := w.Livestock
			name := "someone"
			for _, x := range l {
				if x.ID == m.Load {
					name = x.Name
				}
			}
			out = append(out, sim.Line{Kind: "busy", Text: fmt.Sprintf("%s is in the chair, %s to go.", name, minutes(m.Busy))})
		} else {
			out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("Idle. Brainwashing is %s in Policy. Renfields %d, cots %d.", onOff(w.Policy.Brainwash), w.Renfields(), w.CotCount())})
		}
	case sim.KindCell:
		found := false
		for _, l := range w.Livestock {
			if l.Cell == f.ID {
				v := sim.VictimByID(l.Victim)
				out = append(out, sim.Line{Kind: "food", Text: fmt.Sprintf("%s the %s. Marbling %.1f, hunger %.0f, %d day(s) here.", l.Name, v.Name, l.Marbling, l.Hunger, int(w.Clock-l.Since)/1440)})
				out = append(out, sim.Line{Kind: "flavor", Text: v.Flavor})
				found = true
			}
		}
		if !found {
			out = append(out, sim.Line{Kind: "info", Text: "Empty. The mat only fires when a cell is free."})
		}
	case sim.KindContainer:
		switch f.ID {
		case "fridge", "fridge_big", "freezer", "chest_freezer":
			fresh, frozen := fridgeUsed(w)
			out = append(out, sim.Line{Kind: "food", Text: fmt.Sprintf("Fridge %d/%d fresh, freezer %d/%d frozen. Press I for the full inventory.", fresh, w.FridgeCap(), frozen, w.FreezerCap())})
		case "pantry", "pantry_big":
			out = append(out, sim.Line{Kind: "food", Text: fmt.Sprintf("%d/%d dishes on the shelf.", len(w.Pantry), w.PantryCap())})
		case "back_step":
			counts := map[string]int{}
			for _, e := range w.Pile {
				counts[e.Sub]++
			}
			out = append(out, sim.Line{Kind: "heat", Text: fmt.Sprintf("%d bag(s): %d bones, %d clothes, %d phones. Visible score %d.", len(w.Pile), counts["bones"], counts["clothes"], counts["phone"], w.VisibleEvidence())})
		}
	case sim.KindPlot:
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("%d of %d plots used across the yard.", w.Buried, w.PlotCap())})
	case sim.KindStand:
		out = append(out, sim.Line{Kind: "cash", Text: fmt.Sprintf("%d/%d items out. Sells one an hour, nine to six.", len(w.Stand), w.StandCap())})
	case sim.KindTrap:
		out = append(out, sim.Line{Kind: "info", Text: fmt.Sprintf("Fires on a lured passer-by when a cell is free. Base odds %d%% before the victim's fight.", trapOdds(w, &sim.Victim{}))})
	}
	if m != nil && m.Off {
		out = append(out, sim.Line{Kind: "warn", Text: "Switched OFF."})
	}
	return out
}

func minutes(m int) string {
	if m >= 60 {
		return fmt.Sprintf("%dh%02dm", m/60, m%60)
	}
	return fmt.Sprintf("%dm", m)
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

func (a *app) openFixture(f *sim.Fixture) {
	status := a.fixtureStatus(f)
	items := []item{}
	switch f.Kind {
	case sim.KindStation, sim.KindDisposal, sim.KindChair:
		label := "Switch off"
		if m := a.w.Machines[f.ID]; m != nil && m.Off {
			label = "Switch on"
		}
		items = append(items, item{label: label, desc: status, onSelect: func(a *app) {
			a.do(sim.Command{Kind: "toggle", Name: f.ID})
			a.openFixture(f)
		}})
	}
	// Upgrades that replace this fixture in place.
	for up, old := range replaces {
		u := sim.FixtureByID(up)
		if old == f.ID && u != nil && !a.w.Has(u.ID) && (u.Prereq == "" || a.w.Has(u.Prereq)) {
			items = append(items, item{label: fmt.Sprintf("Upgrade: %s ($%d)", u.Name, u.Cost), color: ui.Col("cash"),
				desc: []sim.Line{{Kind: "info", Text: u.Desc}}, onSelect: func(a *app) { a.openBuild(u.ID) }})
		}
	}
	if f.ID == "chair" {
		items = append(items, item{label: "Put someone in it...", color: ui.Col("fix_closet"), desc: append([]sim.Line{{Kind: "info", Text: "Pick a guest from the cells for the next session."}}, status...),
			onSelect: func(a *app) { a.openChairPicker() }})
		items = append(items, item{label: "Policy: brainwashing " + onOff(a.w.Policy.Brainwash), desc: status, onSelect: func(a *app) { a.openPolicy() }})
	}
	items = append(items, item{label: "Move it", color: ui.Col("key"), desc: append([]sim.Line{{Kind: "info", Text: "Pick it up and put it somewhere closer to whatever it works with. Workers walk; distance is throughput."}}, status...),
		onSelect: func(a *app) { a.startPlacing(f.ID, true) }})
	items = append(items, item{label: "Close", desc: status, onSelect: func(a *app) { a.modal = nil }})
	a.modal = newMenu(f.Name, items)
}

func (a *app) openBuild(focus string) {
	w := a.w
	var items []item
	sel := -1
	for l := sim.Level(0); l < sim.NumLevels; l++ {
		items = append(items, item{label: strings.ToUpper(l.String()), header: true})
		for i := range sim.Fixtures {
			f := &sim.Fixtures[i]
			if f.Level != l || w.Has(f.ID) || f.Cost == 0 {
				continue
			}
			it := item{label: fmt.Sprintf("$%-4d %s", f.Cost, f.Name), color: ui.Col(f.Color)}
			desc := explain(w, f)
			switch {
			case f.Prereq != "" && !w.Has(f.Prereq):
				it.disabled = true
				desc = append(desc, sim.Line{Kind: "warn", Text: "Needs the " + sim.FixtureByID(f.Prereq).Name + " first."})
			case w.Cash < float64(f.Cost):
				it.disabled = true
				desc = append(desc, sim.Line{Kind: "warn", Text: fmt.Sprintf("Costs $%d; you have $%.0f.", f.Cost, w.Cash)})
			default:
				desc = append(desc, sim.Line{Kind: "cash", Text: fmt.Sprintf("Costs $%d of your $%.0f.", f.Cost, w.Cash)})
			}
			it.desc = desc
			it.onSelect = func(a *app) {
				if _, replaced := replaces[f.ID]; replaced {
					if a.do(sim.Command{Kind: "build", Name: f.ID}) {
						a.modal = nil
					}
					return
				}
				a.startPlacing(f.ID, false)
			}
			if f.ID == focus {
				sel = len(items)
			}
			items = append(items, it)
		}
	}
	m := newMenu(fmt.Sprintf("Build ─ $%.0f ─ pick something, then pick a tile", w.Cash), items)
	if sel >= 0 {
		m.sel = sel
	}
	a.modal = m
}

var replaces = map[string]string{"trapdoor_pro": "welcome_mat", "stand_big": "stand"}

// ---------------------------------------------------------------------------
// going out
// ---------------------------------------------------------------------------

func (a *app) openGoOut() {
	w := a.w
	p := w.Player()
	if p == nil {
		return
	}
	if p.Away != nil {
		left := p.Away.Until - w.Clock
		a.modal = textPanel("You're out", []sim.Line{{Kind: "warn", Text: fmt.Sprintf("%s: back in %dh%02dm. The house runs itself meanwhile.", capitalize(p.Away.Label), left/60, left%60)}})
		return
	}
	var items []item
	for _, e := range sim.Errands {
		e := e
		desc := []sim.Line{{Kind: "info", Text: e.Desc}}
		switch e.ID {
		case "market":
			n, total := 0, 0
			for _, d := range w.Stand {
				n++
				total += sim.RecipeByID(d.Sub).Price + 5*(d.Quality-3)
			}
			spare := 0
			for _, d := range w.Pantry {
				if sim.RecipeByID(d.Sub).Food {
					spare++
				}
			}
			spare -= w.Policy.PantryTarget
			for _, d := range w.Pantry {
				rc := sim.RecipeByID(d.Sub)
				if rc.Price > 0 && w.Policy.Sell[rc.ID] && (!rc.Food || spare > 0) {
					n++
					total += rc.Price + 5*(d.Quality-3)
					if rc.Food {
						spare--
					}
				}
			}
			desc = append(desc, sim.Line{Kind: "cash", Text: fmt.Sprintf("Right now: %d item(s), about $%d.", n, total)})
		case "hunt":
			desc = append(desc, sim.Line{Kind: "heat", Text: fmt.Sprintf("Free cells: %d. It is %s.", freeCells(w), map[bool]string{true: "night, good", false: "daytime, worse odds"}[w.IsNight()])})
		case "raid":
			desc = append(desc, sim.Line{Kind: "heat", Text: "Pick a house on the street, or a CPU house nobody will miss. Once per house per real day."})
		}
		items = append(items, item{label: fmt.Sprintf("%-32s %dh", e.Name, e.Hours), color: ui.Col("warn"), desc: desc, onSelect: func(a *app) {
			if e.ID == "raid" {
				a.openRaidPicker()
				return
			}
			if a.do(sim.Command{Kind: "errand", Name: e.ID}) {
				a.modal = nil
			}
		}})
	}
	a.modal = newMenu(fmt.Sprintf("Go out ─ $%.0f in hand, hunger %.0f", w.Cash, p.Hunger), items)
}

func freeCells(w *sim.World) int {
	n := len(w.Cells()) - len(w.Livestock)
	if n < 0 {
		n = 0
	}
	return n
}

func (a *app) openRaidPicker() {
	me := world.Key(a.bbs, a.handle)
	items := []item{{label: "A house down the street (CPU)", color: ui.Col("warn"),
		desc: []sim.Line{{Kind: "info", Text: "The Hendersons, the Nguyens, the model home. 55% for $30-90 and something from the freezer; else a doorbell cam and Heat +8."}},
		onSelect: func(a *app) {
			if a.do(sim.Command{Kind: "errand", Name: "raid"}) {
				a.modal = nil
			}
		}}}
	if hood, err := a.be.Neighborhood(a.bbs, a.handle); err == nil {
		for _, n := range hood.Neighbors {
			n := n
			if n.Key == me {
				continue
			}
			items = append(items, item{label: fmt.Sprintf("%-24s %s@%s  %d stars", trunc(n.Cannibal, 24), n.Handle, n.BBS, n.Stars), color: ui.Col("heat"),
				desc: []sim.Line{{Kind: "info", Text: "Their cameras, fence, doorbell and anyone awake fight back; your van and chemistry set help. Win: 15% of their cash (max $100) and a few things from their kitchen. Lose: Heat +10."}},
				onSelect: func(a *app) {
					w, err := a.be.Raid(a.bbs, a.handle, n.Key)
					if a.take(w, err) {
						a.modal = nil
					}
				}})
		}
	}
	a.modal = newMenu("Raid whose house?", items)
}

// ---------------------------------------------------------------------------
// policy
// ---------------------------------------------------------------------------

func (a *app) openPolicy() {
	p := a.w.Policy
	set := func(a *app, p sim.Policy) {
		a.do(sim.Command{Kind: "policy", Policy: &p})
		a.refreshMenu(a.openPolicy)
	}
	items := []item{
		{label: fmt.Sprintf("Butcher at marbling %d", p.ButcherMarbling), color: ui.Col("food"),
			desc:     []sim.Line{{Kind: "info", Text: "Livestock is processed once it fattens to this marbling (1 = as soon as it lands, 5 = wait for prime). Fed livestock gains a point every three days."}},
			onSelect: func(a *app) { p.ButcherMarbling = p.ButcherMarbling%5 + 1; set(a, p) }},
		{label: "Butcher when every cell is full: " + onOff(p.ButcherWhenFull), color: ui.Col("food"),
			desc:     []sim.Line{{Kind: "info", Text: "With no free cell the trap can't fire. ON processes the fattest guest to make room."}},
			onSelect: func(a *app) { p.ButcherWhenFull = !p.ButcherWhenFull; set(a, p) }},
		{label: fmt.Sprintf("Always keep %d livestock", p.KeepLivestock), color: ui.Col("food"),
			desc:     []sim.Line{{Kind: "info", Text: "A floor under the herd, for The Chair or a rainy day."}},
			onSelect: func(a *app) { p.KeepLivestock = (p.KeepLivestock + 1) % 4; set(a, p) }},
		{label: fmt.Sprintf("Keep %d meals on the shelf", p.PantryTarget), color: ui.Col("food"),
			desc:     []sim.Line{{Kind: "info", Text: "Workers cook food until the shelf holds this many, then cook for the stand instead."}},
			onSelect: func(a *app) { p.PantryTarget = (p.PantryTarget+1)%9 + 0; set(a, p) }},
		{label: "Recipes for the stand...", color: ui.Col("cash"),
			desc:     []sim.Line{{Kind: "info", Text: "Which sellable recipes get cooked once the shelf is stocked, and which dishes get carried to the curb."}},
			onSelect: func(a *app) { a.openSellList() }},
		{label: "Brainwash livestock in The Chair: " + onOff(p.Brainwash), color: ui.Col("fix_closet"),
			desc:     []sim.Line{{Kind: "info", Text: "Needs The Chair and a free cot. Turns the least combative guest into a Renfield who works the house."}},
			onSelect: func(a *app) { p.Brainwash = !p.Brainwash; set(a, p) }},
		{label: fmt.Sprintf("Feed reserve $%d", p.FeedReserve), color: ui.Col("cash"),
			desc:     []sim.Line{{Kind: "info", Text: "Livestock isn't fed when cash would drop below this. Hungry livestock screams, stops fattening, and eventually dies on the step."}},
			onSelect: func(a *app) { p.FeedReserve = (p.FeedReserve + 20) % 120; set(a, p) }},
		{label: "Freeze overflow cuts: " + onOff(p.Freeze), color: ui.Col("fix_cold"),
			desc:     []sim.Line{{Kind: "info", Text: "When the fridge is full, cuts go to the freezer (if you own one) instead of the trash. Frozen cuts cook a star lower."}},
			onSelect: func(a *app) { p.Freeze = !p.Freeze; set(a, p) }},
	}
	a.modal = newMenu("Policy ─ how the house runs itself", items)
}

func (a *app) openSellList() {
	p := a.w.Policy
	var items []item
	for _, rc := range sim.Recipes {
		if rc.Price == 0 {
			continue
		}
		rc := rc
		state := "no"
		col := ui.Col("dim")
		if p.Sell[rc.ID] {
			state, col = "SELL", ui.Col("cash")
		}
		can := a.w.Has(rc.Station)
		for _, n := range rc.Needs {
			if !a.w.Has(n) {
				can = false
			}
		}
		label := fmt.Sprintf("%-4s %-34s $%d", state, rc.Name, rc.Price)
		if !can {
			label += "  (missing gear)"
		}
		items = append(items, item{label: label, color: col, desc: []sim.Line{{Kind: "flavor", Text: rc.Flavor}},
			onSelect: func(a *app) {
				np := a.w.Policy
				sell := map[string]bool{}
				for k, v := range np.Sell {
					sell[k] = v
				}
				sell[rc.ID] = !sell[rc.ID]
				np.Sell = sell
				a.do(sim.Command{Kind: "policy", Policy: &np})
				a.refreshMenu(a.openSellList)
			}})
	}
	a.modal = newMenu("Recipes for the stand", items)
}

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

// ---------------------------------------------------------------------------
// roster and inventory
// ---------------------------------------------------------------------------

func (a *app) openRoster() {
	w := a.w
	var items []item
	items = append(items, item{label: "WORKERS", header: true})
	for _, wk := range w.Workers {
		wk := wk
		col := ui.Col("player")
		kind := "cannibal"
		if wk.Kind == "renfield" {
			col, kind = ui.Col("fix_closet"), fmt.Sprintf("Renfield, loyalty %.0f", wk.Loyalty)
		}
		if wk.Kind == "convert" {
			col, kind = ui.Col("player"), "fellow cannibal, "+strings.ToLower(sim.RoleName(wk.Role))
		} else if wk.Kind == "renfield" {
			kind = fmt.Sprintf("Renfield %s, loyalty %.0f, %d/%d meals", strings.ToLower(sim.RoleName(wk.Role)), wk.Loyalty, wk.Meals, sim.ConvertMeals)
		}
		items = append(items, item{label: fmt.Sprintf("%-12s %-30s hunger %3.0f energy %3.0f", wk.Name, trunc(kind, 30), wk.Hunger, wk.Energy), color: col,
			desc: append(describeWorker(wk), sim.Line{Kind: "dim", Text: map[bool]string{true: "Enter: assign a role.", false: ""}[wk.Kind != "cannibal"]}),
			onSelect: func(a *app) {
				if wk.Kind == "cannibal" {
					a.modal = textPanel(wk.Name, describeWorker(wk))
					return
				}
				a.openRoles(wk)
			}})
	}
	items = append(items, item{label: fmt.Sprintf("LIVESTOCK %d/%d", len(w.Livestock), len(w.Cells())), header: true})
	for _, l := range w.Livestock {
		v := sim.VictimByID(l.Victim)
		where := "in transit"
		if l.Cell != "" {
			where = sim.FixtureByID(l.Cell).Name
		}
		col := ui.Col(v.Color)
		if l.Hunger > 50 {
			col = ui.Col("heat_hi")
		}
		l := l
		if l.Order != "" {
			where += " *" + l.Order
		}
		items = append(items, item{label: fmt.Sprintf("%-10s %-28s %-12s marb %.1f hunger %3.0f", l.Name, trunc(v.Name, 28), where, l.Marbling, l.Hunger), color: col,
			desc:     []sim.Line{{Kind: "flavor", Text: v.Flavor}, {Kind: "info", Text: fmt.Sprintf("Fight %d. Processed at marbling %d per policy. Enter: give a direct order.", v.Fight, w.Policy.ButcherMarbling)}},
			onSelect: func(a *app) { a.openOrders(l) }})
	}
	a.modal = newMenu("Roster", items)
}

func (a *app) openOrders(l *sim.Livestock) {
	v := sim.VictimByID(l.Victim)
	give := func(order string) func(a *app) {
		return func(a *app) {
			if a.do(sim.Command{Kind: "order", ID: l.ID, Name: order}) {
				a.openRoster()
			}
		}
	}
	items := []item{
		{label: "Send to The Chair", color: ui.Col("fix_closet"), desc: []sim.Line{{Kind: "info", Text: "The next free worker walks them to The Chair. Needs The Chair and a free cot; ignores the brainwashing policy."}}, onSelect: give("chair")},
		{label: "Butcher now", color: ui.Col("food"), desc: []sim.Line{{Kind: "info", Text: "The next free worker takes them to the block, whatever their marbling."}}, onSelect: give("butcher")},
		{label: "Back to policy", desc: []sim.Line{{Kind: "info", Text: "Clear any order; policy decides."}}, onSelect: give("")},
	}
	a.modal = newMenu(fmt.Sprintf("%s the %s", l.Name, v.Name), items)
}

func (a *app) openRoles(wk *sim.Worker) {
	var items []item
	for _, r := range sim.Roles {
		r := r
		label := r.Name
		if wk.Role == r.ID || (wk.Role == "" && r.ID == sim.RoleHand) {
			label += "  (current)"
		}
		items = append(items, item{label: label, color: ui.Col("fix_closet"), desc: []sim.Line{{Kind: "info", Text: r.Desc}},
			onSelect: func(a *app) {
				if a.do(sim.Command{Kind: "role", ID: wk.ID, Name: r.ID}) {
					a.openRoster()
				}
			}})
	}
	a.modal = newMenu("What is "+wk.Name+" for?", items)
}

func (a *app) openChairPicker() {
	var items []item
	for _, l := range a.w.Livestock {
		l := l
		if l.Cell == "" {
			continue
		}
		v := sim.VictimByID(l.Victim)
		items = append(items, item{label: fmt.Sprintf("%-10s %-28s fight %d  marbling %.1f", l.Name, trunc(v.Name, 28), v.Fight, l.Marbling), color: ui.Col(v.Color),
			desc: []sim.Line{{Kind: "flavor", Text: v.Flavor}},
			onSelect: func(a *app) {
				if a.do(sim.Command{Kind: "order", ID: l.ID, Name: "chair"}) {
					a.modal = nil
				}
			}})
	}
	a.modal = newMenu("Who goes in The Chair?", items)
}

func (a *app) openInventory() {
	w := a.w
	var items []item
	fresh, frozen := fridgeUsed(w)
	items = append(items, item{label: fmt.Sprintf("FRIDGE %d/%d  FREEZER %d/%d", fresh, w.FridgeCap(), frozen, w.FreezerCap()), header: true})
	for _, c := range w.Fridge {
		v := sim.VictimByID(c.Victim)
		age := int(w.Clock-c.Born) / 60
		state := fmt.Sprintf("%dh old", age)
		col := ui.Col("food")
		if c.Frozen {
			state, col = "FROZEN", ui.Col("fix_cold")
		} else if age >= 48 {
			col = ui.Col("warn")
		}
		items = append(items, item{label: fmt.Sprintf("%-6s %-28s marbling %-5s %s", c.Sub, trunc(v.Name, 28), strings.Repeat("*", c.Marbling), state), color: col})
	}
	items = append(items, item{label: fmt.Sprintf("SHELF %d/%d", len(w.Pantry), w.PantryCap()), header: true})
	for _, d := range w.Pantry {
		d := d
		rc := sim.RecipeByID(d.Sub)
		col := ui.Col("food")
		if !rc.Food {
			col = ui.Col("cash")
		}
		items = append(items, item{label: fmt.Sprintf("%-34s %s", trunc(rc.Name, 34), starsText(d.Quality)), color: col,
			desc: []sim.Line{{Kind: "flavor", Text: rc.Flavor}, {Kind: "info", Text: "Enter: send this dish to a neighbor as a casserole."}},
			onSelect: func(a *app) {
				if !rc.Food {
					return
				}
				a.openNeighborPicker("Send "+rc.Name+" to", func(a *app, n world.Neighbor) {
					w, err := a.be.Casserole(a.bbs, a.handle, n.Key, d.ID)
					if a.take(w, err) {
						a.note("good", "Left a %s on %s's porch. What a neighbor you are.", rc.Name, n.Cannibal)
					}
					a.modal = nil
				})
			}})
	}
	items = append(items, item{label: fmt.Sprintf("CURB STAND %d/%d", len(w.Stand), w.StandCap()), header: true})
	for _, d := range w.Stand {
		rc := sim.RecipeByID(d.Sub)
		items = append(items, item{label: fmt.Sprintf("%-34s $%d", trunc(rc.Name, 34), rc.Price+5*(d.Quality-3)), color: ui.Col("cash")})
	}
	items = append(items, item{label: fmt.Sprintf("BACK STEP %d bags, %d buried, %d cars", len(w.Pile), w.Buried, len(w.Driveway)), header: true})
	for _, e := range w.Pile {
		items = append(items, item{label: fmt.Sprintf("%-8s %-28s %dh on the step", e.Sub, trunc(sim.VictimByID(e.Victim).Name, 28), int(w.Clock-e.Born)/60), color: ui.Col("heat")})
	}
	for _, c := range w.Driveway {
		items = append(items, item{label: fmt.Sprintf("car      %-28s %dh at the curb", trunc(sim.VictimByID(c.Victim).Name, 28), int(w.Clock-c.Born)/60), color: ui.Col("heat_hi")})
	}
	a.modal = newMenu("Inventory", items)
}

func starsText(q int) string {
	return strings.Repeat("*", q) + strings.Repeat("-", 5-q)
}

// ---------------------------------------------------------------------------
// the street
// ---------------------------------------------------------------------------

func (a *app) openNeighborPicker(title string, pick func(a *app, n world.Neighbor)) {
	hood, err := a.be.Neighborhood(a.bbs, a.handle)
	if err != nil {
		a.note("bad", "Couldn't reach the street: %v", err)
		return
	}
	me := world.Key(a.bbs, a.handle)
	var items []item
	for _, n := range hood.Neighbors {
		n := n
		if n.Key == me {
			continue
		}
		items = append(items, item{label: fmt.Sprintf("%-24s %s@%s", trunc(n.Cannibal, 24), n.Handle, n.BBS),
			onSelect: func(a *app) { pick(a, n) }})
	}
	a.modal = newMenu(title, items)
}

func (a *app) openNeighborhood() {
	hood, err := a.be.Neighborhood(a.bbs, a.handle)
	if err != nil {
		a.note("bad", "Couldn't reach the street: %v", err)
		return
	}
	me := world.Key(a.bbs, a.handle)
	var items []item
	for _, n := range hood.Neighbors {
		n := n
		status := fmt.Sprintf("gen %d  day %-3d  %3d stars  %s", n.Generation, n.Day, n.Stars, sim.TierName(sim.HeatTier(float64(n.Heat))))
		col := ui.Col("text")
		if !n.Alive {
			status, col = "deceased (awaiting an heir)", ui.Col("dim")
		}
		label := fmt.Sprintf("%-22s %s", trunc(n.Cannibal, 22), status)
		if n.Key == me {
			label, col = "* "+label, ui.Col("bright")
		}
		it := item{label: label, color: col,
			desc: []sim.Line{{Kind: "info", Text: fmt.Sprintf("%s on %s. %d livestock, %d Renfields, %d meals.", n.Handle, n.BBS, n.Livestock, n.Renfields, n.Meals)}}}
		if n.Key != me {
			it.desc = append(it.desc, sim.Line{Kind: "dim", Text: "Enter: anonymous tip to the HOA (+10 Heat for them, may backfire). Casseroles go from the Inventory (I)."})
			it.onSelect = func(a *app) {
				a.modal = confirm("Call the HOA about "+n.Cannibal+"?", "Heat +10 for them. 35% chance the hotline's caller ID gives you +8.",
					func(a *app) {
						w, err := a.be.Tip(a.bbs, a.handle, n.Key)
						if a.take(w, err) {
							a.note("info", "You called the HOA about %s's garage. Anonymously. Mostly.", n.Cannibal)
						}
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
	var lines []sim.Line
	for i := len(hood.Feed) - 1; i >= 0; i-- {
		e := hood.Feed[i]
		lines = append(lines, sim.Line{Kind: e.Kind, Text: fmt.Sprintf("%s %s", e.Cannibal, e.Text)})
	}
	if len(lines) == 0 {
		lines = append(lines, sim.Line{Kind: "dim", Text: "Nothing posted yet. The street is suspiciously quiet."})
	}
	p := textPanel("Nextdoor ─ newest first", lines)
	p.footer = "↑↓ scroll · any key closes"
	a.modal = p
}

// ---------------------------------------------------------------------------
// options, help, death, away digest
// ---------------------------------------------------------------------------

func (a *app) openOptions() {
	mode := "16-color"
	if a.buf.Mode == ui.Mode256 {
		mode = "256-color"
	}
	var anc []sim.Line
	for _, an := range a.w.Ancestors {
		anc = append(anc, sim.Line{Kind: "dim", Text: fmt.Sprintf("%s: %d days, %d meals, %d stars. %s.", an.Name, an.Days, an.Meals, an.Stars, an.Cause)})
	}
	if len(anc) == 0 {
		anc = append(anc, sim.Line{Kind: "dim", Text: "No one has died in this house. Yet."})
	}
	items := []item{
		{label: "Colors: " + mode, desc: []sim.Line{{Kind: "info", Text: "Toggle between classic 16-color (every terminal) and 256-color (SyncTERM, NetRunner, most modern terminals)."}},
			onSelect: func(a *app) {
				if a.buf.Mode == ui.Mode256 {
					a.buf.Mode = ui.Mode16
					a.do(sim.Command{Kind: "color", Name: "16"})
				} else {
					a.buf.Mode = ui.Mode256
					a.do(sim.Command{Kind: "color", Name: "256"})
				}
				a.buf.Invalidate()
				a.openOptions()
			}},
		{label: "Rename " + a.w.Cannibal, desc: []sim.Line{{Kind: "info", Text: "Type a new name for the current cannibal."}},
			onSelect: func(a *app) {
				a.modal = newPrompt("New name", a.w.Cannibal, func(a *app, v string) { a.do(sim.Command{Kind: "rename", Name: v}) })
			}},
		{label: "Journal order: " + map[bool]string{true: "oldest at the top", false: "newest at the top"}[a.w.JournalOrder == "oldest"],
			desc:     []sim.Line{{Kind: "info", Text: "Newest-first keeps the latest event right under the map; reading order runs top to bottom. J toggles it from the map too."}},
			onSelect: func(a *app) { a.toggleJournal(); a.openOptions() }},
		{label: "Family history", desc: anc, onSelect: func(a *app) { a.modal = textPanel("The dynasty", anc) }},
		{label: "Start over from nothing", color: ui.Col("bad"),
			desc: []sim.Line{{Kind: "warn", Text: "Burns the dynasty down: a new cannibal, the starter kitchen, no fixtures, no ancestors, no cash. Not the same as dying; dying keeps the house."}},
			onSelect: func(a *app) {
				a.modal = confirm("Really start over?", "Everything this family built is gone. The neighbors get a new house to gossip about.",
					func(a *app) {
						if a.do(sim.Command{Kind: "reset"}) {
							a.seenDeath = false
							a.level = sim.Ground
							a.note("info", "A fresh start. Same street, new name, empty fridge.")
						}
					})
			}},
		{label: fmt.Sprintf("Clock: %d game minutes per real minute", a.w.Scale), disabled: true,
			desc: []sim.Line{{Kind: "info", Text: "Set by the sysop (time_scale in cannibalcasa.ini). The house runs at this rate whether or not you're here."}}},
	}
	a.modal = newMenu("Options", items)
}

func deathScreen(a *app) *panel {
	w := a.w
	lines := []sim.Line{
		{Kind: "bad", Text: fmt.Sprintf("%s: %s.", w.Cannibal, w.Cause)},
		{Kind: "info", Text: ""},
		{Kind: "info", Text: fmt.Sprintf("%d days on the cul-de-sac. %d meals. %d stars. %d guests captured, %d escaped, %d dinner parties, %d items sold.", w.DaysAlive(), w.Stats.Meals, w.Stars, w.Stats.Captured, w.Stats.Escaped, w.Stats.Dinners, w.Stats.Sold)},
		{Kind: "info", Text: ""},
		{Kind: "warn", Text: "A relative will inherit the house. Fixtures stay; cash is halved; the basement, the fridge and the back step come empty; the neighbors come curious."},
		{Kind: "dim", Text: "Press any key to read the will."},
	}
	p := textPanel("In Memoriam", lines)
	p.footer = "any key"
	p.onDone = func(a *app) {
		if a.do(sim.Command{Kind: "inherit"}) {
			a.seenDeath = false
		}
	}
	return p
}

func awayDigest(a *app, ticks int) *panel {
	w := a.w
	since := w.Clock - int64(ticks)
	var lines []sim.Line
	lines = append(lines, sim.Line{Kind: "info", Text: fmt.Sprintf("The house ran for %d day(s) and %d hour(s) while you were out.", ticks/1440, ticks%1440/60)})
	lines = append(lines, sim.Line{Kind: "info", Text: ""})
	n := 0
	for _, l := range w.Ticker {
		if l.Tick < since || l.Kind == "info" {
			continue
		}
		lines = append(lines, sim.Line{Kind: l.Kind, Text: fmt.Sprintf("d%d %02d:%02d %s", l.Tick/1440+1, l.Tick%1440/60, l.Tick%60, l.Text)})
		n++
	}
	if n == 0 {
		lines = append(lines, sim.Line{Kind: "dim", Text: "Nothing worth mentioning. The machine hummed."})
	}
	p := textPanel("While you were away", lines)
	p.footer = "↑↓ scroll · any key closes"
	return p
}
