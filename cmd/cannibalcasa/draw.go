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

func (a *app) levelBg(l sim.Level) ui.Attr {
	switch l {
	case sim.Basement:
		return ui.Col("bg_basement")
	case sim.Backyard:
		return ui.Col("bg_yard")
	case sim.Attic:
		return ui.Col("bg_attic")
	}
	return ui.Col("bg_ground")
}

// fixtureCell picks glyph, fg and bg for an owned fixture by what it's doing.
func (a *app) fixtureCell(f *sim.Fixture, floorBg ui.Attr, occupants map[string]occupant) (rune, ui.Attr, ui.Attr) {
	if o, ok := occupants[f.ID]; ok {
		return o.glyph, o.color, ui.Col("bg_cell")
	}
	m := a.w.Machines[f.ID]
	fg, bg := ui.Col(f.Color), floorBg
	if m != nil && m.Off {
		return f.Glyph, ui.Col("dim"), floorBg
	}
	switch f.Kind {
	case sim.KindStation, sim.KindDisposal, sim.KindChair:
		if m != nil && (m.Busy > 0 || m.Output != nil) {
			bg = ui.Col("bg_busy")
			fg = ui.Col("bright")
		}
	case sim.KindContainer:
		switch f.ID {
		case "back_step":
			n := len(a.w.Pile)
			switch {
			case n == 0:
				fg = ui.Col("dim")
			case n >= 5:
				fg, bg = ui.Col("bright"), ui.Col("bg_full")
			default:
				fg = ui.Col("heat_mid")
			}
		case "fridge", "fridge_big":
			if fresh, _ := fridgeUsed(a.w); fresh >= a.w.FridgeCap() {
				bg = ui.Col("bg_full")
			}
		case "pantry", "pantry_big":
			if len(a.w.Pantry) >= a.w.PantryCap() {
				bg = ui.Col("bg_full")
			}
		}
	case sim.KindStand:
		if len(a.w.Stand) > 0 {
			bg = ui.Col("bg_busy")
		}
	case sim.KindCell:
		fg = ui.Col("dim")
	}
	return f.Glyph, fg, bg
}

func (a *app) drawMap(ox, oy int) {
	b := a.buf
	w := a.w
	p := a.plan()
	border := ui.Col("border")
	b.Box(ox, oy, mapW, mapH, border, ui.Black, "", ui.Black)
	// Title strip across the top of the map frame.
	b.Fill(ox+1, oy, mapW-2, 1, ' ', ui.Col("strip_fg"), ui.Col("strip_bg"))
	b.Print(ox+2, oy, "CANNIBAL CASA", ui.Col("title"), ui.Col("strip_bg"))
	b.Print(ox+16, oy, strings.ToUpper(a.level.String()), ui.Col("strip_fg"), ui.Col("strip_bg"))
	b.Print(ox+mapW-12, oy, "Tab", ui.Col("key"), ui.Col("strip_bg"))
	b.Print(ox+mapW-8, oy, "floor", ui.Col("label"), ui.Col("strip_bg"))

	floorBg := a.levelBg(a.level)
	wallCol := ui.Col("wall")
	switch a.level {
	case sim.Basement:
		wallCol = ui.Col("wall_basement")
	case sim.Backyard:
		wallCol = ui.Col("wall_yard")
		if w.Has("fence") {
			wallCol = ui.Col("fix_yard")
		}
	case sim.Attic:
		wallCol = ui.Col("wall_attic")
	}
	isWall := func(x, y int) bool {
		return y >= 0 && y < len(p.Rows) && x >= 0 && x < len(p.Rows[y]) && p.Rows[y][x] == '#'
	}
	for y, row := range p.Rows {
		for x := 0; x < len(row); x++ {
			sx, sy := ox+1+x, oy+1+y
			switch ch := row[x]; ch {
			case '#':
				if isWall(x, y-1) && isWall(x+1, y) && isWall(x, y+1) && isWall(x-1, y) &&
					isWall(x-1, y-1) && isWall(x+1, y-1) && isWall(x-1, y+1) && isWall(x+1, y+1) {
					b.Set(sx, sy, ' ', wallCol, ui.Black)
					continue
				}
				b.Set(sx, sy, ui.WallGlyph(isWall(x, y-1), isWall(x+1, y), isWall(x, y+1), isWall(x-1, y)), wallCol, ui.Black)
			case '.':
				bg := floorBg
				if a.level == sim.Ground && y <= 4 {
					bg = ui.Col("bg_porch")
				}
				b.Set(sx, sy, ' ', ui.White, bg)
			case '+':
				b.Set(sx, sy, '+', ui.Col("door"), floorBg)
			case '=':
				b.Set(sx, sy, '═', ui.Col("street"), ui.Black)
			case '-':
				b.Set(sx, sy, ' ', ui.Col("sidewalk"), ui.Col("bg_sidewalk"))
			case '>', '<':
				b.Set(sx, sy, rune(ch), ui.Col("key"), floorBg)
			}
		}
	}
	for _, r := range p.Rooms {
		for i, ch := range r.Name {
			x, y := r.X+i, r.Y
			if p.At(x, y) == '.' {
				bg := floorBg
				if a.level == sim.Ground && y <= 4 {
					bg = ui.Col("bg_porch")
				}
				b.Set(ox+1+x, oy+1+y, ch, ui.Col("label"), bg)
			}
		}
	}
	// Fixtures where the player put them.
	occupants := a.cellOccupants()
	for id, at := range w.Placed {
		f := sim.FixtureByID(id)
		if f == nil || !w.Has(id) || at.L != a.level {
			continue
		}
		g, fg, bg := a.fixtureCell(f, b.Get(ox+1+at.X, oy+1+at.Y).Bg, occupants)
		b.Set(ox+1+at.X, oy+1+at.Y, g, fg, bg)
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
	// Entities keep the background of the tile they stand on.
	put := func(pos sim.Pos, g rune, fg ui.Attr) {
		if pos.L != a.level {
			return
		}
		c := b.Get(ox+1+pos.X, oy+1+pos.Y)
		b.Set(ox+1+pos.X, oy+1+pos.Y, g, fg, c.Bg)
	}
	for _, ps := range w.Passers {
		v := sim.VictimByID(ps.Victim)
		col := ui.Col(v.Color)
		if ps.State == "lured" {
			col = ui.Col("bright")
		} else if ps.State == "flee" {
			col = ui.Col("heat_hi")
		}
		put(ps.Pos, v.Glyph, col)
	}
	for _, vs := range w.Visitors {
		g, col := 'I', ui.Col("victim_heat")
		switch vs.Kind {
		case "dog":
			g, col = 'D', ui.Col("door")
		case "police":
			g, col = 'P', ui.Col("fix_cell")
		}
		put(vs.Pos, g, col)
	}
	for _, wk := range w.Workers {
		if wk.Away != nil {
			continue
		}
		g, col := '@', ui.Col("player")
		switch wk.Kind {
		case "renfield":
			g, col = 'r', ui.Col("fix_closet")
		case "convert":
			g, col = '&', ui.Col("player")
		}
		if wk.Asleep {
			col = ui.Col("dim")
		}
		put(wk.Pos, g, col)
	}
	// Cursor, or the placement preview.
	if pl, ok := a.modal.(*placer); ok {
		f := sim.FixtureByID(pl.id)
		bg := ui.Col("bg_ok")
		if a.w.CanPlace(f, sim.Pos{L: a.level, X: a.cx, Y: a.cy}) != nil {
			bg = ui.Col("bg_bad")
		}
		b.Set(ox+1+a.cx, oy+1+a.cy, f.Glyph, ui.Col("bright"), bg)
	} else {
		c := b.Get(ox+1+a.cx, oy+1+a.cy)
		b.Set(ox+1+a.cx, oy+1+a.cy, c.Ch, ui.Black, ui.Col("cursor"))
	}
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
	// Header strip: who and when.
	b.Fill(ox+1, y, w-2, 2, ' ', ui.Col("strip_fg"), ui.Col("strip_bg"))
	b.PrintN(x, y, trunc(name, iw-7), iw-7, ui.Col("strip_fg"), ui.Col("strip_bg"))
	b.Print(x+iw-6, y, fmt.Sprintf("gen %d", s.Generation), ui.Col("label"), ui.Col("strip_bg"))
	y++
	speed := fmt.Sprintf("%dx", a.speed)
	speedCol := ui.Col("good")
	if a.speed == 0 {
		speed, speedCol = "PAUSED", ui.Col("warn")
	} else if _, ok := a.modal.(*menu); ok {
		speed, speedCol = "menu", ui.Col("label")
	}
	b.PrintN(x, y, s.TimeString(), iw, ui.Col("strip_fg"), ui.Col("strip_bg"))
	b.Print(x+iw-len(speed), y, speed, speedCol, ui.Col("strip_bg"))
	y++
	b.Print(x, y, fmt.Sprintf("Cash $%.0f", s.Cash), ui.Col("cash"), ui.Black)
	b.Print(x+iw/2, y, fmt.Sprintf("Stars %d", s.Stars), ui.Col("stars"), ui.Black)
	y++
	barW := iw - 11
	heat := s.Heat
	if heat > 100 {
		heat = 100
	}
	b.Print(x, y, "Heat", ui.Col("label"), ui.Black)
	b.Bar(x+7, y, barW, int(heat), 100, tierColor(heat), ui.Col("bar_off"), ui.Black)
	b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", heat), tierColor(heat), ui.Black)
	y++
	tierBg := ui.Black
	if s.Tier() >= 3 {
		tierBg = ui.Col("alert_bg")
	}
	b.PrintN(x+7, y, sim.TierName(s.Tier()), iw-7, tierColor(heat), tierBg)
	y++
	if p := s.Player(); p != nil {
		b.Print(x, y, "Hunger", ui.Col("label"), ui.Black)
		b.Bar(x+7, y, barW, int(p.Hunger), 100, needColor(p.Hunger), ui.Col("bar_off"), ui.Black)
		b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", p.Hunger), needColor(p.Hunger), ui.Black)
		y++
		b.Print(x, y, "Energy", ui.Col("label"), ui.Black)
		b.Bar(x+7, y, barW, int(p.Energy), 100, needColor(p.Energy), ui.Col("bar_off"), ui.Black)
		b.Print(x+8+barW, y, fmt.Sprintf("%3.0f", p.Energy), needColor(p.Energy), ui.Black)
		y++
	}
	// Stock lines: label in gray, values colored by how they're doing.
	fresh, frozen := fridgeUsed(s)
	hungry := 0
	for _, l := range s.Livestock {
		if l.Hunger > 50 {
			hungry++
		}
	}
	x2 := stat(b, x, y, "Cells", fmt.Sprintf("%d/%d", len(s.Livestock), len(s.Cells())), pick(hungry > 0, ui.Col("warn"), ui.Col("food")))
	stat(b, x2+2, y, "Fridge", fmt.Sprintf("%d/%d+%d", fresh, s.FridgeCap(), frozen), pick(fresh >= s.FridgeCap(), ui.Col("bad"), ui.Col("food")))
	y++
	x2 = stat(b, x, y, "Shelf", fmt.Sprintf("%d/%d", len(s.Pantry), s.PantryCap()), pick(len(s.Pantry) == 0, ui.Col("warn"), pick(len(s.Pantry) >= s.PantryCap(), ui.Col("bad"), ui.Col("food"))))
	stat(b, x2+2, y, "Stand", fmt.Sprintf("%d/%d", len(s.Stand), s.StandCap()), ui.Col("cash"))
	y++
	// Evidence line: green when clean, yellow when there's work, red on a background when it's a problem.
	pile, cars := len(s.Pile), len(s.Driveway)
	lineBg := ui.Black
	if pile >= 6 || cars >= 2 {
		lineBg = ui.Col("alert_bg")
	}
	b.Fill(x, y, iw, 1, ' ', ui.White, lineBg)
	x2 = statBg(b, x, y, "Step", fmt.Sprint(pile), pick(pile == 0, ui.Col("good"), pick(pile < 4, ui.Col("warn"), ui.Col("bad"))), lineBg)
	x2 = statBg(b, x2+2, y, "Cars", fmt.Sprint(cars), pick(cars == 0, ui.Col("good"), ui.Col("bad")), lineBg)
	statBg(b, x2+2, y, "Buried", fmt.Sprintf("%d/%d", s.Buried, s.PlotCap()), pick(s.Buried >= s.PlotCap() && s.PlotCap() > 0, ui.Col("warn"), ui.Col("text")), lineBg)
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
		if wk.Kind == "convert" {
			g = "&"
		}
		label := wk.JobLabel()
		if wk.Away != nil {
			left := wk.Away.Until - s.Clock
			label = fmt.Sprintf("away, back in %dh%02dm", left/60, left%60)
			col = ui.Col("warn")
		}
		b.PrintN(x, y, trunc(fmt.Sprintf("%s %s: %s", g, firstName(wk.Name), label), iw), iw, col, ui.Black)
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
		bg := ui.Col("alert_bg")
		if alertCol == ui.Col("good") {
			bg = ui.Col("calm_bg")
		}
		b.PrintN(x, y, " "+alert, iw, ui.Col("strip_fg"), bg)
	}
}

// stat prints "Label value" with the label in gray and returns the x after it.
func stat(b *ui.Buffer, x, y int, label, value string, col ui.Attr) int {
	return statBg(b, x, y, label, value, col, ui.Black)
}

func statBg(b *ui.Buffer, x, y int, label, value string, col, bg ui.Attr) int {
	x = b.Print(x, y, label+" ", ui.Col("label"), bg)
	return b.Print(x, y, value, col, bg)
}

func pick(cond bool, a, b ui.Attr) ui.Attr {
	if cond {
		return a
	}
	return b
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
	b.Box(ox, oy, w, h, ui.Col("border"), ui.Black, "", ui.Black)
	b.Print(ox+2, oy, " Journal ", ui.Col("strip_fg"), ui.Col("strip_bg"))
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
		// "dN HH:MM " stamps in gray, the line itself in its kind's color.
		stamp := 0
		if strings.HasPrefix(l.Text, "d") {
			if i := strings.Index(l.Text, ":"); i > 0 && i+3 < len(l.Text) {
				stamp = i + 4
			}
		}
		if stamp > 0 {
			b.Print(ox+2, y, l.Text[:stamp], ui.Col("label"), ui.Black)
			b.PrintN(ox+2+stamp, y, l.Text[stamp:], w-4-stamp, kindColor(l.Kind), ui.Black)
		} else {
			b.PrintN(ox+2, y, l.Text, w-4, kindColor(l.Kind), ui.Black)
		}
		y--
	}
}

func (a *app) drawFooter(y int) {
	b := a.buf
	keys := []struct{ k, v string }{
		{"Spc", "pause"}, {"Tab", "floor"}, {"B", "build"}, {"G", "out"}, {"P", "policy"},
		{"L", "roster"}, {"I", "stock"}, {"N", "street"}, {"?", "guide"},
	}
	bg := ui.Col("strip_bg")
	b.Fill(0, y, b.W, 1, ' ', ui.White, bg)
	x := 1
	for _, kv := range keys {
		if x+len(kv.k)+len(kv.v)+2 > b.W {
			break
		}
		x = b.Print(x, y, kv.k, ui.Col("key"), bg)
		x = b.Print(x, y, " "+kv.v+" ", ui.Col("strip_fg"), bg)
	}
}
