// The door's application state and key handling. The app holds the last
// World the backend returned, the cursor, the speed, and at most one modal.
// The world advances on its own; the app just asks for it once a second.
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/sim"
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
	w         *sim.World
	buf       *ui.Buffer
	cfg       config
	level     sim.Level
	cx, cy    int
	speed     int // 0 paused, 1, 2, 4
	notes     []sim.Line
	modal     modal
	quit      bool
	seenDeath bool
	status    string
}

func newApp(be world.Backend, bbs, handle string, cfg config, buf *ui.Buffer) *app {
	a := &app{be: be, bbs: bbs, handle: handle, cfg: cfg, buf: buf, speed: 1}
	a.cx, a.cy = 8, 7
	return a
}

// note adds a door-side line to the journal (errors, connection news).
func (a *app) note(kind, format string, args ...any) {
	t := int64(0)
	if a.w != nil {
		t = a.w.Clock
	}
	a.notes = append(a.notes, sim.Line{Tick: t, Kind: kind, Text: fmt.Sprintf(format, args...)})
	if len(a.notes) > 100 {
		a.notes = a.notes[len(a.notes)-100:]
	}
}

func (a *app) take(w *sim.World, err error) bool {
	if w != nil {
		a.w = w
	}
	if err != nil {
		a.note("warn", "%s", capitalize(err.Error()))
		return false
	}
	a.checkDeath()
	return true
}

func (a *app) do(c sim.Command) bool {
	w, err := a.be.Do(a.bbs, a.handle, c)
	return a.take(w, err)
}

func (a *app) join() {
	w, ev, ticks, err := a.be.Join(a.bbs, a.handle)
	if err != nil {
		a.note("bad", "Could not join the neighborhood: %v", err)
		return
	}
	a.w = w
	for _, e := range ev {
		a.note(e.Kind, "%s", e.Text)
	}
	if w.ColorMode == "256" {
		a.buf.Mode = ui.Mode256
	}
	if ticks >= 60 {
		a.modal = awayDigest(a, ticks)
	}
	a.checkDeath()
}

func (a *app) sync() {
	if a.w == nil {
		return
	}
	speed := a.speed
	if a.modal != nil {
		if _, ok := a.modal.(*panel); ok {
			// reading a panel doesn't stop the world; menus do (you're at the controls)
		} else {
			speed = 0
		}
	}
	if !a.w.Alive {
		speed = 0
	}
	w, err := a.be.Sync(a.bbs, a.handle, speed)
	if err != nil {
		a.note("bad", "Lost the neighborhood: %v", err)
		return
	}
	a.w = w
	a.checkDeath()
}

func (a *app) checkDeath() {
	if a.w != nil && !a.w.Alive && !a.seenDeath {
		a.seenDeath = true
		a.modal = deathScreen(a)
	}
}

func (a *app) plan() sim.Plan { return sim.Plans[a.level] }

func (a *app) handleKey(k key) {
	if a.modal != nil {
		a.modal.key(a, k)
		return
	}
	if a.w == nil {
		if k == keyEsc || k == 'q' || k == 'Q' {
			a.quit = true
		}
		return
	}
	switch k {
	case keyUp, 'w', 'W':
		a.moveCursor(0, -1)
	case keyDown, 's', 'S':
		a.moveCursor(0, 1)
	case keyLeft, 'a', 'A':
		a.moveCursor(-1, 0)
	case keyRight, 'd', 'D':
		a.moveCursor(1, 0)
	case keyEnter:
		a.inspectCursor()
	case keyTab, '>', '.':
		a.switchLevel(1)
	case '<', ',':
		a.switchLevel(-1)
	case ' ':
		if a.speed == 0 {
			a.speed = 1
		} else {
			a.speed = 0
		}
	case '1':
		a.speed = 1
	case '2':
		a.speed = 2
	case '3', '4':
		a.speed = 4
	case 'b', 'B':
		a.openBuild("")
	case 'g', 'G':
		a.openGoOut()
	case 'p', 'P':
		a.openPolicy()
	case 'l', 'L':
		a.openRoster()
	case 'i', 'I':
		a.openInventory()
	case 'n', 'N':
		a.openNeighborhood()
	case 'o', 'O':
		a.openOptions()
	case 'j', 'J':
		a.toggleJournal()
	case '?', 'h', 'H':
		a.openHelp()
	case keyEsc, 'q', 'Q':
		a.modal = confirm("Leave the house?", "The machine keeps running while you're gone. The neighbors will assume you're at work.", func(a *app) { a.quit = true })
	}
}

// toggleJournal flips the journal between newest-first and reading order.
// The choice is saved with the world so it follows the caller.
func (a *app) toggleJournal() {
	order := "oldest"
	if a.w.JournalOrder == "oldest" {
		order = "newest"
	}
	a.do(sim.Command{Kind: "journal", Name: order})
	if order == "oldest" {
		a.note("info", "Journal: oldest at the top, newest at the bottom. J flips it back.")
	} else {
		a.note("info", "Journal: newest at the top. J flips it back.")
	}
}

func (a *app) moveCursor(dx, dy int) {
	p := a.plan()
	nx, ny := a.cx+dx, a.cy+dy
	if ny < 0 || ny >= len(p.Rows) || nx < 0 || nx >= len(p.Rows[0]) {
		return
	}
	a.cx, a.cy = nx, ny
}

func (a *app) switchLevel(d int) {
	a.level = sim.Level((int(a.level) + d + int(sim.NumLevels)) % int(sim.NumLevels))
	p := a.plan()
	if a.cy >= len(p.Rows) {
		a.cy = len(p.Rows) / 2
	}
}

// inspectCursor opens whatever is under the cursor: a fixture panel, a build
// offer for an empty slot, or a description of an entity.
func (a *app) inspectCursor() {
	p := a.plan()
	pos := sim.Pos{L: a.level, X: a.cx, Y: a.cy}
	for _, wk := range a.w.Workers {
		if wk.Pos == pos {
			a.modal = textPanel(wk.Name, describeWorker(wk))
			return
		}
	}
	for _, ps := range a.w.Passers {
		if ps.Pos == pos {
			v := sim.VictimByID(ps.Victim)
			var who []string
			for _, t := range v.Traits {
				who = append(who, sim.Traits[t])
			}
			a.modal = textPanel(v.Name, []sim.Line{{Kind: "flavor", Text: v.Flavor},
				{Kind: "info", Text: "One of " + strings.Join(who, ", ") + ". Marbling " + strings.Repeat("*", v.Marbling) + ", fight " + strings.Repeat("!", v.Fight) + "."}})
			return
		}
	}
	for _, vs := range a.w.Visitors {
		if vs.Pos == pos {
			a.modal = textPanel(capitalize(vs.Name), []sim.Line{{Kind: "warn", Text: visitorText(vs)}})
			return
		}
	}
	if f := a.w.FixtureAt(pos); f != nil {
		a.openFixture(f)
		return
	}
	if p.Floor(a.cx, a.cy) {
		a.openBuild("")
	}
}

// placer is placement mode: the cursor carries a fixture; Enter drops it.
type placer struct {
	id   string
	move bool
}

func (pl *placer) key(a *app, k key) {
	switch k {
	case keyEsc, 'q', 'Q':
		a.modal = nil
	case keyUp, 'w', 'W':
		a.moveCursor(0, -1)
	case keyDown, 's', 'S':
		a.moveCursor(0, 1)
	case keyLeft, 'a', 'A':
		a.moveCursor(-1, 0)
	case keyRight, 'd', 'D':
		a.moveCursor(1, 0)
	case keyTab, '>', '.':
		a.switchLevel(1)
	case '<', ',':
		a.switchLevel(-1)
	case keyEnter, ' ':
		at := sim.Pos{L: a.level, X: a.cx, Y: a.cy}
		kind := "build"
		if pl.move {
			kind = "move"
		}
		if a.do(sim.Command{Kind: kind, Name: pl.id, At: &at}) {
			a.modal = nil
		}
	}
}

func (pl *placer) draw(a *app, b *ui.Buffer) {
	f := sim.FixtureByID(pl.id)
	verb := "Place"
	if pl.move {
		verb = "Move"
	}
	where := f.Level.String()
	if f.Front {
		where = "the porch"
	}
	msg := fmt.Sprintf(" %s the %s (%s): arrows move, Enter drops it, Esc cancels. Goes on %s. ", verb, f.Name, category(f), where)
	if err := a.w.CanPlace(f, sim.Pos{L: a.level, X: a.cx, Y: a.cy}); err != nil {
		msg = fmt.Sprintf(" %s the %s: %s. Esc cancels. ", verb, f.Name, capitalize(err.Error()))
	}
	b.PrintN(0, b.H-1, msg, b.W, ui.Col("bright"), ui.Col("sel_bg"))
}

// startPlacing enters placement mode on the fixture's level, cursor on a
// sensible first tile.
func (a *app) startPlacing(id string, move bool) {
	f := sim.FixtureByID(id)
	a.level = f.Level
	if move {
		at := a.w.Pos(id)
		a.cx, a.cy = at.X, at.Y
	} else if f.Front {
		a.cx, a.cy = 2, 3
	} else {
		p := a.plan()
		found := false
		for y := 0; y < len(p.Rows) && !found; y++ {
			for x := 0; x < len(p.Rows[y]) && !found; x++ {
				if a.w.CanPlace(f, sim.Pos{L: f.Level, X: x, Y: y}) == nil {
					a.cx, a.cy = x, y
					found = true
				}
			}
		}
	}
	a.modal = &placer{id: id, move: move}
}

func category(f *sim.Fixture) string {
	switch f.Kind {
	case sim.KindLure:
		return "lure"
	case sim.KindTrap:
		return "trap"
	case sim.KindStand:
		return "shop"
	case sim.KindStation:
		return "cooking station"
	case sim.KindDisposal:
		return "evidence disposal"
	case sim.KindPlot:
		return "burial"
	case sim.KindContainer:
		return "storage"
	case sim.KindCell:
		return "cell"
	case sim.KindBed:
		return "sleep"
	case sim.KindChair:
		return "brainwashing"
	case sim.KindBlock:
		return "butchering"
	case sim.KindVan:
		return "vehicle"
	}
	return "upgrade"
}

func trapOdds(w *sim.World, v *sim.Victim) int {
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
	c -= v.Fight * 8
	if c < 10 {
		c = 10
	}
	if c > 95 {
		c = 95
	}
	return c
}

func visitorText(v *sim.Visitor) string {
	switch v.Kind {
	case "inspector":
		return fmt.Sprintf("Tier %d visitor. They'll knock, look around from the front door, and leave. Anything visible (%s) counts against you.", v.Tier, "cars at the curb, bags on the step without a fence, screaming without soundproofing")
	case "dog":
		return "Gladys's labradoodle. Digs where things are buried. A fence keeps it out."
	case "police":
		return "It's over."
	}
	return v.Name
}

func describeWorker(wk *sim.Worker) []sim.Line {
	kind := "The cannibal of the house."
	switch wk.Kind {
	case "renfield":
		kind = fmt.Sprintf("A Renfield (%s), formerly a %s. Loyalty %.0f. %d of %d meals toward becoming one of the family.", strings.ToLower(sim.RoleName(wk.Role)), strings.ToLower(sim.VictimByID(wk.Victim).Name), wk.Loyalty, wk.Meals, sim.ConvertMeals)
	case "convert":
		kind = fmt.Sprintf("A fellow cannibal (%s), formerly a %s. Loyal for life. Eats like family, hunts at night on their own.", strings.ToLower(sim.RoleName(wk.Role)), strings.ToLower(sim.VictimByID(wk.Victim).Name))
	}
	if wk.Away != nil {
		return []sim.Line{{Kind: "info", Text: kind}, {Kind: "warn", Text: fmt.Sprintf("Out: %s.", wk.Away.Label)}}
	}
	return []sim.Line{
		{Kind: "info", Text: kind},
		{Kind: "food", Text: fmt.Sprintf("Hunger %.0f/100, energy %.0f/100.", wk.Hunger, wk.Energy)},
		{Kind: "good", Text: "Doing: " + wk.JobLabel() + "."},
		{Kind: "dim", Text: fmt.Sprintf("Carrying %d item(s).", len(wk.Carry))},
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func tierColor(heat float64) ui.Attr {
	switch {
	case heat >= 60:
		return ui.Col("heat_hi")
	case heat >= 30:
		return ui.Col("heat_mid")
	}
	return ui.Col("heat_low")
}

func needColor(v float64) ui.Attr {
	switch {
	case v <= 25:
		return ui.Col("heat_hi")
	case v <= 50:
		return ui.Col("heat_mid")
	}
	return ui.Col("heat_low")
}

func kindColor(kind string) ui.Attr {
	switch kind {
	case "cash":
		return ui.Col("cash_ev")
	case "dim":
		return ui.Col("dim")
	case "title":
		return ui.Col("title")
	case "flavor":
		return ui.Col("flavor")
	case "rule":
		return ui.Col("rule")
	}
	return ui.Col(kind)
}

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
