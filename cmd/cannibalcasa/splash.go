// Title screen: a block-letter logo in a blood gradient, a tagline, and the
// backend status line, until a key is pressed (or a few seconds pass).
package main

import (
	"bufio"
	"time"

	"cannibalcasa/internal/ui"
)

// A 5-row block font for the letters the logo needs.
var blockFont = map[rune][]string{
	'C': {" ███", "█   ", "█   ", "█   ", " ███"},
	'A': {" ██ ", "█  █", "████", "█  █", "█  █"},
	'N': {"█  █", "██ █", "█ ██", "█  █", "█  █"},
	'I': {"███", " █ ", " █ ", " █ ", "███"},
	'B': {"███ ", "█  █", "███ ", "█  █", "███ "},
	'L': {"█   ", "█   ", "█   ", "█   ", "████"},
	'S': {" ███", "█   ", " ██ ", "   █", "███ "},
	' ': {"  ", "  ", "  ", "  ", "  "},
}

func logoWidth(s string) int {
	w := 0
	for _, ch := range s {
		w += len([]rune(blockFont[ch][0])) + 1
	}
	return w
}

func drawLogo(b *ui.Buffer, x, y int, s string, rows []ui.Attr) {
	for _, ch := range s {
		g := blockFont[ch]
		for r, line := range g {
			for i, c := range line {
				if c == '█' {
					b.Set(x+i, y+r, '█', rows[r%len(rows)], ui.Black)
				}
			}
		}
		x += len([]rune(g[0])) + 1
	}
}

func runSplash(w *bufio.Writer, b *ui.Buffer, in *input, status string) {
	deadline := time.After(6 * time.Second)
	frame := 0
	tick := time.NewTicker(120 * time.Millisecond)
	defer tick.Stop()
	for {
		b.Clear(ui.Black)
		grad := []ui.Attr{ui.BrightRed, ui.BrightRed, ui.Red, ui.Red, ui.Blood}
		cy := b.H/2 - 7
		if cy < 1 {
			cy = 1
		}
		lw := logoWidth("CANNIBAL")
		if lw+2 < b.W {
			drawLogo(b, (b.W-lw)/2, cy, "CANNIBAL", grad)
			lw2 := logoWidth("CASA")
			drawLogo(b, (b.W-lw2)/2, cy+6, "CASA", grad)
		} else {
			b.Print((b.W-13)/2, cy+3, "CANNIBAL CASA", ui.BrightRed, ui.Black)
		}
		// Drips under the logo, animated.
		for i := 0; i < b.W/6; i++ {
			dx := 4 + i*6 + (i*7)%5
			dl := (i*3 + frame/3) % 4
			for d := 0; d < dl; d++ {
				b.Set(dx, cy+11+d, '▄', ui.Red, ui.Black)
			}
		}
		tag := "a builder game about eating the neighbors"
		b.Print((b.W-len(tag))/2, cy+13, tag, ui.Col("dim"), ui.Black)
		b.PrintN(2, b.H-3, trunc(status, b.W-4), b.W-4, ui.Col("food"), ui.Black)
		press := "press any key"
		col := ui.Col("key")
		if frame%8 < 4 {
			col = ui.Col("dim")
		}
		b.Print((b.W-len(press))/2, b.H-2, press, col, ui.Black)
		b.Flush(w)
		w.Flush()
		select {
		case <-in.events:
			return
		case <-in.quitCh:
			return
		case <-deadline:
			return
		case <-tick.C:
			frame++
		}
	}
}
