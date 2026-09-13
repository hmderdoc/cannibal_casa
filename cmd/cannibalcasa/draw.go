// Rendering the main screen: the floor plan, the sidebar of meters, the
// journal and the key footer. Everything goes through the ui.Buffer so only
// changed cells hit the wire.
package main

import (
	"fmt"
	"strings"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/ui"
)

const (
	mapW   = 48 // 46-wide plan + frame
	mapH   = 14 // tallest plan (12 rows) + frame
	minSbW = 30
)

func (a *app) render() {
	b := a.buf
	b.Clear(ui.Black)
	if a.st == nil {
		b.Print(2, 1, "No player state. Press Q.", ui.Col("bad"), ui.Black)
		a.drawLog(0, 3, b.W, b.H-4)
		return
	}
	a.drawMap(0, 0)
	a.drawSidebar(mapW, 0, b.W-mapW, mapH)
	a.drawLog(0, mapH, b.W, b.H-mapH-1)
	a.drawFooter(b.H - 1)
	if a.modal != nil {
		a.modal.draw(a, b)
	}
}

func (a *app) drawMap(ox, oy int) {
	b := a.buf
	p := a.plan()
	border := ui.Col("border")
	b.Box(ox, oy, mapW, mapH, border, ui.Black, "", ui.Black)
	title := fmt.Sprintf(" CANNIBAL CASA ─ %s ", strings.ToUpper(a.level.String()))
	b.Print(ox+2, oy, title, ui.Col("title"), ui.Black)
	b.Print(ox+mapW-14, oy, " Tab: floor ", ui.Col("dim"), ui.Black)

	wallCol, floorCol, floorCh := ui.Col("wall"), ui.Col("floor_ground"), '·'
	switch a.level {
	case game.Basement:
		wallCol, floorCol = ui.Col("wall_basement"), ui.Col("floor_basement")
	case game.Backyard:
		wallCol, floorCol, floorCh = ui.Col("wall_yard"), ui.Col("floor_yard"), ','
	case game.Attic:
		wallCol, floorCol = ui.Col("wall_attic"), ui.Col("floor_attic")
	}
	isWall := func(x, y int) bool {
		return y >= 0 && y < len(p.Rows) && x >= 0 && x < len(p.Rows[y]) && p.Rows[y][x] == '#'
	}
	cells := a.cellOccupants()
	for y, row := range p.Rows {
		for x := 0; x < len(row); x++ {
			sx, sy := ox+1+x, oy+1+y
			switch ch := row[x]; ch {
			case '#':
				if isWall(x, y-1) && isWall(x+1, y) && isWall(x, y+1) && isWall(x-1, y) &&
					isWall(x-1, y-1) && isWall(x+1, y-1) && isWall(x-1, y+1) && isWall(x+1, y+1) {
					b.Set(sx, sy, '▒', wallCol, ui.Black) // solid interior
					continue
				}
				b.Set(sx, sy, ui.WallGlyph(isWall(x, y-1), isWall(x+1, y), isWall(x, y+1), isWall(x-1, y)), wallCol, ui.Black)
			case '.', '@':
				b.Set(sx, sy, floorCh, floorCol, ui.Black)
			case '+':
				b.Set(sx, sy, '+', ui.Col("door"), ui.Black)
			default:
				f := game.FixtureAtSlot(a.level, ch)
				if f == nil {
					b.Set(sx, sy, '?', ui.Col("slot_empty"), ui.Black)
				} else if !a.st.Has(f.ID) {
					b.Set(sx, sy, '?', ui.Col("slot_empty"), ui.Black)
				} else if v, ok := cells[f.ID]; ok {
					b.Set(sx, sy, v.Glyph, ui.Col(v.Color), ui.Black)
				} else {
					b.Set(sx, sy, f.Glyph, ui.Col(f.Color), ui.Black)
				}
			}
		}
	}
	for _, r := range p.Rooms {
		for i, ch := range r.Name {
			x, y := r.X+i, r.Y
			if y < len(p.Rows) && x < len(p.Rows[y]) && p.Rows[y][x] == '.' {
				b.Set(ox+1+x, oy+1+y, ch, ui.Col("label"), ui.Black)
			}
		}
	}
	// Renfields loiter in the hall on the ground floor.
	if a.level == game.Ground {
		n := 0
		for _, c := range a.st.Captives {
			if c.Renfield && n < 2 {
				b.Set(ox+1+40+n*2, oy+1+9, 'r', ui.Col("fix_closet"), ui.Black)
				n++
			}
		}
	}
	b.Set(ox+1+a.px, oy+1+a.py, '@', ui.Col("player"), ui.Black)
}

// cellOccupants maps cell fixture ids to the victim in them, in roster order.
func (a *app) cellOccupants() map[string]*game.Victim {
	out := map[string]*game.Victim{}
	cells := []string{"cell_1", "cell_2", "cell_3", "cell_4", "cell_5"}
	i := 0
	for _, c := range a.st.Captives {
		if c.Renfield {
			continue
		}
		for i < len(cells) && !a.st.Has(cells[i]) {
			i++
		}
		if i >= len(cells) {
			break
		}
		out[cells[i]] = game.VictimByID(c.Victim)
		i++
	}
	return out
}

func (a *app) drawSidebar(ox, oy, w, h int) {
	b := a.buf
	s := a.st
	if w < 20 {
		return
	}
	b.Box(ox, oy, w, h, ui.Col("border"), ui.Black, "", ui.Black)
	x, y := ox+2, oy+1
	iw := w - 4 // inner width
	name := s.Cannibal
	if !s.Alive {
		name += " (deceased)"
	}
	b.PrintN(x, y, trunc(name, iw), iw, ui.Col("bright"), ui.Black)
	y++
	b.PrintN(x, y, fmt.Sprintf("Gen %d  Day %d  %s", s.Generation, s.Day, s.Phase), iw, ui.Col("text"), ui.Black)
	y++
	bankCol := ui.Col("dim")
	if s.Bank == 0 {
		bankCol = ui.Col("warn")
	}
	b.PrintN(x, y, fmt.Sprintf("Days banked: %d", s.Bank), iw, bankCol, ui.Black)
	y++
	barW := iw - 11
	if barW < 6 {
		barW = 6
	}
	b.Print(x, y, "Hunger", ui.Col("text"), ui.Black)
	b.Bar(x+7, y, barW, s.Hunger, 100, hungerColor(s.Hunger), ui.Col("bar_off"), ui.Black)
	b.Print(x+8+barW, y, fmt.Sprintf("%3d", s.Hunger), hungerColor(s.Hunger), ui.Black)
	y++
	heat := s.Heat
	if heat > 100 {
		heat = 100
	}
	b.Print(x, y, "Heat  ", ui.Col("text"), ui.Black)
	b.Bar(x+7, y, barW, heat, 100, tierColor(heat), ui.Col("bar_off"), ui.Black)
	b.Print(x+8+barW, y, fmt.Sprintf("%3d", heat), tierColor(heat), ui.Black)
	y++
	b.PrintN(x+7, y, game.TierName(s.Tier()), iw-7, tierColor(heat), ui.Black)
	y++
	b.Print(x, y, fmt.Sprintf("Cash $%d", s.Cash), ui.Col("cash"), ui.Black)
	b.Print(x+iw/2, y, fmt.Sprintf("Stars %d", s.Stars), ui.Col("stars"), ui.Black)
	y++
	fresh, frozen := 0, 0
	for _, c := range s.Fridge {
		if c.Frozen {
			frozen++
		} else {
			fresh++
		}
	}
	b.PrintN(x, y, fmt.Sprintf("Fridge %d/%d  Frz %d/%d  Meals %d", fresh, s.FridgeCap(), frozen, s.FreezerCap(), len(s.Dishes)), iw, ui.Col("food"), ui.Black)
	y++
	b.PrintN(x, y, fmt.Sprintf("Livestock %d/%d  Renfields %d", s.Livestock(), s.CellCount(), s.Renfields()), iw, ui.Col("fix_cell"), ui.Black)
	y++
	ev := len(s.Evidence) - s.Buried()
	evCol := ui.Col("dim")
	if ev > 0 {
		evCol = ui.Col("heat")
	}
	b.PrintN(x, y, fmt.Sprintf("Evidence %d loose, %d buried", ev, s.Buried()), iw, evCol, ui.Black)
	y++
	switch {
	case s.Mission != nil:
		m := game.MissionByID(s.Mission.ID)
		b.PrintN(x, y, trunc(fmt.Sprintf("Away: %s (%d)", m.Name, s.Mission.Phases), iw), iw, ui.Col("warn"), ui.Black)
	case s.Inspection != nil:
		b.PrintN(x, y, trunc(fmt.Sprintf("VISITOR in %d phase(s)!", s.Inspection.Phases), iw), iw, ui.Col("bad"), ui.Black)
	case s.Phase == game.Night && len(s.Prey) > 0:
		b.PrintN(x, y, fmt.Sprintf("%d out walking. [H]unt", len(s.Prey)), iw, ui.Col("good"), ui.Black)
	default:
		b.PrintN(x, y, "[T] to pass the time", iw, ui.Col("dim"), ui.Black)
	}
	y++
	if y < oy+h-1 {
		b.PrintN(x, y, fmt.Sprintf("Dynasty stars: %d (gen %d)", s.DynastyStars(), s.Generation), iw, ui.Col("dim"), ui.Black)
	}
}

func (a *app) drawLog(ox, oy, w, h int) {
	b := a.buf
	if h < 3 {
		return
	}
	b.Box(ox, oy, w, h, ui.Col("border"), ui.Black, "Journal", ui.Col("dim"))
	lines := make([]game.Event, 0, h)
	// Walk backwards, wrapping, until the box is full.
	for i := len(a.log) - 1; i >= 0 && len(lines) < h-2; i-- {
		ws := wrap(a.log[i].Text, w-4)
		for j := len(ws) - 1; j >= 0 && len(lines) < h-2; j-- {
			lines = append(lines, game.Event{Kind: a.log[i].Kind, Text: ws[j]})
		}
	}
	y := oy + h - 2
	for _, l := range lines {
		b.PrintN(ox+2, y, l.Text, w-4, kindColor(l.Kind), ui.Black)
		y--
	}
}

func (a *app) drawFooter(y int) {
	b := a.buf
	keys := []struct{ k, v string }{
		{"Tab", "floor"}, {"T", "next"}, {"B", "build"}, {"C", "cook"}, {"E", "eat"},
		{"L", "stock"}, {"X", "evid"}, {"M", "jobs"}, {"N", "street"}, {"?", "help"},
	}
	x := 1
	for i, kv := range keys {
		if x+len(kv.k)+len(kv.v)+2 > b.W {
			break
		}
		if i > 0 {
			b.Set(x, y, '·', ui.Col("dim"), ui.Black)
			x++
		}
		x = b.Print(x, y, kv.k, ui.Col("key"), ui.Black)
		x = b.Print(x, y, " "+kv.v, ui.Col("dim"), ui.Black)
	}
}
