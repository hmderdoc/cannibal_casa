// Rendering: the floor plan with everything moving on it, the sidebar of
// meters and workers, the ticker, and the key footer.
package main

import (
	"fmt"
	"sort"
	"strings"

	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/ui"
)

const (
	mapW = 48 // 46-wide plan + frame
	mapH = 16 // tallest plan (14 rows) + frame
)

func (a *app) render() {
	b := a.buf
	b.Clear(ui.Black)
	if a.w == nil {
		b.Print(2, 1, "No world. Press Q.", ui.Col("bad"), ui.Black)
		a.drawTicker(0, 3, b.W, b.H-4)
		return
	}
	a.drawMap(0, 0)
	a.drawSidebar(mapW, 0, b.W-mapW, mapH)
	a.drawTicker(0, mapH, b.W, b.H-mapH-1)
	a.drawFooter(b.H - 1)
	if a.modal != nil {
		a.modal.draw(a, b)
	}
}

// machineColor tints an owned fixture by what it's doing.
func (a *app) machineColor(f *sim.Fixture) ui.Attr {
	m := a.w.Machines[f.ID]
	if m != nil && m.Off {
		return ui.Col("dim")
	}
	switch f.Kind {
	case sim.KindStation, sim.KindDisposal, sim.KindChair:
		if m != nil && (m.Busy > 0 || m.Output != nil) {
			return ui.Col("busy")
		}
	case sim.KindContainer:
		switch f.ID {
		case "back_step":
			n := len(a.w.Pile)
			switch {
			case n == 0:
				return ui.Col("dim")
			case n >= 5:
				return ui.Col("heat_hi")
			}
			return ui.Col("heat_mid")
		case "fridge", "fridge_big":
			fresh, _ := fridgeUsed(a.w)
			if fresh >= a.w.FridgeCap() {
				return ui.Col("full")
			}
		case "pantry", "pantry_big":
			if len(a.w.Pantry) >= a.w.PantryCap() {
				return ui.Col("full")
			}
		}
	case sim.KindStand:
		if len(a.w.Stand) > 0 {
			return ui.Col("busy")
		}
	}
	return ui.Col(f.Color)
}

func fridgeUsed(w *sim.World) (fresh, frozen int) {
	for _, c := range w.Fridge {
		if c.Frozen {
			frozen++
		} else {
			fresh++
		}
	}
	return
}

func (a *app) drawMap(ox, oy int) {
	b := a.buf
	w := a.w
	p := a.plan()
	border := ui.Col("border")
	b.Box(ox, oy, mapW, mapH, border, ui.Black, "", ui.Black)
	b.Print(ox+2, oy, fmt.Sprintf(" CANNIBAL CASA ─ %s ", strings.ToUpper(a.level.String())), ui.Col("title"), ui.Black)
	b.Print(ox+mapW-14, oy, " Tab: floor ", ui.Col("dim"), ui.Black)

	wallCol, floorCol, floorCh := ui.Col("wall"), ui.Col("floor_ground"), '·'
	switch a.level {
	case sim.Basement:
		wallCol, floorCol = ui.Col("wall_basement"), ui.Col("floor_basement")
	case sim.Backyard:
		wallCol, floorCol, floorCh = ui.Col("wall_yard"), ui.Col("floor_yard"), ','
		if w.Has("fence") {
			wallCol = ui.Col("fix_yard")
		}
	case sim.Attic:
		wallCol, floorCol = ui.Col("wall_attic"), ui.Col("floor_attic")
	}
	isWall := func(x, y int) bool {
		return y >= 0 && y < len(p.Rows) && x >= 0 && x < len(p.Rows[y]) && p.Rows[y][x] == '#'
	}
	occupants := a.cellOccupants()
	for y, row := range p.Rows {
		for x := 0; x < len(row); x++ {
			sx, sy := ox+1+x, oy+1+y
			switch ch := row[x]; ch {
			case '#':
				if isWall(x, y-1) && isWall(x+1, y) && isWall(x, y+1) && isWall(x-1, y) &&
					isWall(x-1, y-1) && isWall(x+1, y-1) && isWall(x-1, y+1) && isWall(x+1, y+1) {
					b.Set(sx, sy, '▒', wallCol, ui.Black)
					continue
				}
				b.Set(sx, sy, ui.WallGlyph(isWall(x, y-1), isWall(x+1, y), isWall(x, y+1), isWall(x-1, y)), wallCol, ui.Black)
			case '.', '@':
				b.Set(sx, sy, floorCh, floorCol, ui.Black)
			case '+':
				b.Set(sx, sy, '+', ui.Col("door"), ui.Black)
			case '=':
				b.Set(sx, sy, '═', ui.Col("street"), ui.Black)
			case '-':
				b.Set(sx, sy, '─', ui.Col("sidewalk"), ui.Black)
			case '>':
				b.Set(sx, sy, '>', ui.Col("door"), ui.Black)
			case '<':
				b.Set(sx, sy, '<', ui.Col("door"), ui.Black)
			default:
				var owned *sim.Fixture
				for _, f := range sim.FixturesAtSlot(a.level, ch) {
					if w.Has(f.ID) {
						owned = f
					}
				}
				if owned == nil {
					b.Set(sx, sy, '?', ui.Col("slot_empty"), ui.Black)
				} else if v, ok := occupants[owned.ID]; ok {
					b.Set(sx, sy, v.glyph, v.color, ui.Black)
				} else {
					b.Set(sx, sy, owned.Glyph, a.machineColor(owned), ui.Black)
				}
			}
		}
	}
	for _, r := range p.Rooms {
		for i, ch := range r.Name {
			x, y := r.X+i, r.Y
			if p.At(x, y) == '.' {
				b.Set(ox+1+x, oy+1+y, ch, ui.Col("label"), ui.Black)
			}
		}
	}
	// Cars at the curb.
	if a.level == sim.Ground {
		for i := range w.Driveway {
			if i > 3 {
				break
			}
			b.Set(ox+1+sim.DrivewayX+i*2, oy+1+sim.StreetY, 'c', ui.Col("heat_hi"), ui.Black)
		}
	}
	// Entities: passers-by, visitors, workers.
	for _, ps := range w.Passers {
		if ps.Pos.L != a.level {
			continue
		}
		v := sim.VictimByID(ps.Victim)
		col := ui.Col(v.Color)
		if ps.State == "lured" {
			col = ui.Col("bright")
		} else if ps.State == "flee" {
			col = ui.Col("heat_hi")
		}
		b.Set(ox+1+ps.Pos.X, oy+1+ps.Pos.Y, v.Glyph, col, ui.Black)
	}
	for _, vs := range w.Visitors {
		if vs.Pos.L != a.level {
			continue
		}
		g, col := 'I', ui.Col("victim_heat")
		switch vs.Kind {
		case "dog":
			g, col = 'D', ui.Col("door")
		case "police":
			g, col = 'P', ui.Col("fix_cell")
		}
		b.Set(ox+1+vs.Pos.X, oy+1+vs.Pos.Y, g, col, ui.Black)
	}
	for _, wk := range w.Workers {
		if wk.Pos.L != a.level {
			continue
		}
		g, col := '@', ui.Col("player")
		if wk.Kind == "renfield" {
			g, col = 'r', ui.Col("fix_closet")
		}
		if wk.Asleep {
			col = ui.Col("dim")
		}
		b.Set(ox+1+wk.Pos.X, oy+1+wk.Pos.Y, g, col, ui.Black)
	}
	// Cursor: invert the cell.
	c := b.Get(ox+1+a.cx, oy+1+a.cy)
	b.Set(ox+1+a.cx, oy+1+a.cy, c.Ch, ui.Black, ui.Col("cursor"))
}

type occupant struct {
	glyph rune
	color ui.Attr
}

func (a *app) cellOccupants() map[string]occupant {
	out := map[string]occupant{}
	for _, l := range a.w.Livestock {
		if l.Cell == "" {
			continue
		}
		v := sim.VictimByID(l.Victim)
		col := ui.Col(v.Color)
		if l.Hunger > 50 {
			col = ui.Col("heat_hi")
		}
		out[l.Cell] = occupant{v.Glyph, col}
	}
	if m := a.w.Machines["chair"]; m != nil && m.Load != 0 {
		out["chair"] = occupant{'&', ui.Col("busy")}
	}
	return out
}

func (a *app) drawSidebar(ox, oy, w, h int) {
	b := a.buf
	s := a.w
	if w < 20 {
		return
	}
	b.Box(ox, oy, w, h, ui.Col("border"), ui.Black, "", ui.Black)
	x, y := ox+2, oy+1
	iw := w - 4
	name := s.Cannibal
	if !s.Alive {
		name += " (deceased)"
	}
	b.PrintN(x, y, trunc(fmt.Sprintf("%s  gen %d", name, s.Generation), iw), iw, ui.Col("bright"), ui.Black)
	y++
	speed := fmt.Sprintf("%dx", a.speed)
	speedCol := ui.Col("good")
	if a.speed == 0 {
		speed, speedCol = "PAUSED", ui.Col("warn")
	}
	b.PrintN(x, y, s.TimeString(), iw, ui.Col("text"), ui.Black)
	b.Print(x+iw-len(speed), y, speed, speedCol, ui.Black)
	y++
	b.Print(x, y, fmt.Sprintf("Cash $%.0f", s.Cash), ui.Col("cash"), ui.Black)
	b.Print(x+iw/2, y, fmt.Sprintf("Stars %d", s.Stars), ui.Col("stars"), ui.Black)
	y++
	barW := iw - 11
	heat := s.Heat
	if heat > 100 {
		heat = 100
	}
	b.Print(x, y, "Heat  ", ui.Col("text"), ui.Black)
	b.Bar(x+7, y, barW, int(heat), 100, tierColor(heat), ui.Col("bar_off"), ui.Black)
	b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", heat), tierColor(heat), ui.Black)
	y++
	b.PrintN(x+7, y, sim.TierName(s.Tier()), iw-7, tierColor(heat), ui.Black)
	y++
	if p := s.Player(); p != nil {
		b.Print(x, y, "Hunger", ui.Col("text"), ui.Black)
		b.Bar(x+7, y, barW, int(p.Hunger), 100, needColor(p.Hunger), ui.Col("bar_off"), ui.Black)
		b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", p.Hunger), needColor(p.Hunger), ui.Black)
		y++
		b.Print(x, y, "Energy", ui.Col("text"), ui.Black)
		b.Bar(x+7, y, barW, int(p.Energy), 100, needColor(p.Energy), ui.Col("bar_off"), ui.Black)
		b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", p.Energy), needColor(p.Energy), ui.Black)
		y++
	}
	fresh, frozen := fridgeUsed(s)
	b.PrintN(x, y, fmt.Sprintf("Cells %d/%d  Fridge %d/%d +%d", len(s.Livestock), len(s.Cells()), fresh, s.FridgeCap(), frozen), iw, ui.Col("food"), ui.Black)
	y++
	b.PrintN(x, y, fmt.Sprintf("Shelf %d/%d  Stand %d/%d", len(s.Pantry), s.PantryCap(), len(s.Stand), s.StandCap()), iw, ui.Col("food"), ui.Black)
	y++
	pileCol := ui.Col("dim")
	if len(s.Pile) > 0 || len(s.Driveway) > 0 {
		pileCol = ui.Col("heat")
	}
	b.PrintN(x, y, fmt.Sprintf("Step %d  Cars %d  Buried %d/%d", len(s.Pile), len(s.Driveway), s.Buried, s.PlotCap()), iw, pileCol, ui.Black)
	y++
	for _, wk := range s.Workers {
		if y >= oy+h-2 {
			break
		}
		g := "@"
		col := ui.Col("player")
		if wk.Kind == "renfield" {
			g, col = "r", ui.Col("fix_closet")
		}
		b.PrintN(x, y, trunc(fmt.Sprintf("%s %s: %s", g, firstName(wk.Name), wk.JobLabel()), iw), iw, col, ui.Black)
		y++
	}
	alert := ""
	alertCol := ui.Col("warn")
	for _, vs := range s.Visitors {
		switch vs.Kind {
		case "police":
			alert, alertCol = "THE FBI IS HERE", ui.Col("bad")
		case "inspector":
			if alert == "" {
				alert = "VISITOR AT THE DOOR"
			}
		case "dog":
			if alert == "" {
				alert = "dog in the yard"
			}
		}
	}
	if alert == "" {
		for _, ps := range s.Passers {
			if ps.State == "lured" {
				alert, alertCol = "someone's on the porch...", ui.Col("good")
			}
		}
	}
	if alert != "" && y < oy+h-1 {
		b.PrintN(x, y, alert, iw, alertCol, ui.Black)
	}
}

func firstName(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return s
	}
	return parts[len(parts)-1]
}

// drawTicker shows the newest journal lines, world ticker and door notes
// merged, oldest at the top.
func (a *app) drawTicker(ox, oy, w, h int) {
	b := a.buf
	if h < 3 {
		return
	}
	b.Box(ox, oy, w, h, ui.Col("border"), ui.Black, "Journal", ui.Col("dim"))
	var lines []sim.Line
	if a.w != nil {
		lines = append(lines, a.w.Ticker...)
	}
	lines = append(lines, a.notes...)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Tick < lines[j].Tick })
	var rows []sim.Line
	for i := len(lines) - 1; i >= 0 && len(rows) < h-2; i-- {
		stamp := fmt.Sprintf("d%d %02d:%02d ", lines[i].Tick/1440+1, lines[i].Tick%1440/60, lines[i].Tick%60)
		ws := wrap(lines[i].Text, w-4-len(stamp))
		for j := len(ws) - 1; j >= 0 && len(rows) < h-2; j-- {
			pre := strings.Repeat(" ", len(stamp))
			if j == 0 {
				pre = stamp
			}
			rows = append(rows, sim.Line{Kind: lines[i].Kind, Text: pre + ws[j]})
		}
	}
	y := oy + h - 2
	for _, l := range rows {
		b.PrintN(ox+2, y, l.Text, w-4, kindColor(l.Kind), ui.Black)
		y--
	}
}

func (a *app) drawFooter(y int) {
	b := a.buf
	keys := []struct{ k, v string }{
		{"Spc", "pause"}, {"Tab", "floor"}, {"B", "build"}, {"P", "policy"},
		{"L", "roster"}, {"I", "stock"}, {"N", "street"}, {"?", "help"},
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
