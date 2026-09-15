package main

import (
	"strings"
	"testing"

	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/ui"
)

func journalText(a *app) string {
	b := a.buf
	var sb strings.Builder
	for y := 17; y < 22; y++ {
		var row []rune
		for x := 2; x < 78; x++ {
			row = append(row, b.Get(x, y).Ch)
		}
		sb.WriteString(strings.TrimRight(string(row), " ") + "\n")
	}
	return sb.String()
}

func TestJournalOrderKeepsWrappedLines(t *testing.T) {
	w := sim.New("t", "b", 1)
	w.Ticker = nil
	long := "first line of a long entry that will certainly wrap onto a second line because it goes on and on"
	w.Ticker = append(w.Ticker, sim.Line{Tick: 100, Kind: "info", Text: "old short entry"}, sim.Line{Tick: 200, Kind: "info", Text: long})
	for _, order := range []string{"newest", "oldest"} {
		w.JournalOrder = order
		a := &app{w: w, buf: ui.NewBuffer(80, 24)}
		a.render()
		txt := journalText(a)
		i1, i2 := strings.Index(txt, "first line"), strings.Index(txt, "on and on")
		if i1 < 0 || i2 < 0 || i1 > i2 {
			t.Fatalf("%s: wrapped lines out of order:\n%s", order, txt)
		}
		short := strings.Index(txt, "old short")
		if order == "newest" && short < i1 || order == "oldest" && short > i1 {
			t.Fatalf("%s: entries out of order:\n%s", order, txt)
		}
	}
}
