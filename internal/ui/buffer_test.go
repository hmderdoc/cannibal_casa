package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestFlushDiffsAndEncodes(t *testing.T) {
	b := NewBuffer(10, 3)
	var out bytes.Buffer
	b.Print(0, 0, "hi", White, Black)
	if err := b.Flush(&out); err != nil {
		t.Fatal(err)
	}
	first := out.String()
	if !strings.HasPrefix(first, "\x1b[0m\x1b[2J\x1b[H") || !strings.Contains(first, "hi") {
		t.Fatalf("first flush should clear and draw: %q", first)
	}
	out.Reset()
	if err := b.Flush(&out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("unchanged frame should emit nothing, got %q", out.String())
	}
	b.Set(5, 1, '─', Yellow, Black)
	out.Reset()
	b.Flush(&out)
	s := out.String()
	if !strings.Contains(s, "\x1b[2;6H") || !strings.Contains(s, "\x1b[0;1;33;40m") || !strings.Contains(s, "\xc4") {
		t.Fatalf("diff should move, color, and CP437-encode: %q", s)
	}
	b.UTF8 = true
	b.Mode = Mode256
	b.Set(6, 1, '─', Yellow, Black)
	out.Reset()
	b.Flush(&out)
	if !strings.Contains(out.String(), "─") || !strings.Contains(out.String(), "38;5;220") {
		t.Fatalf("utf8/256 output wrong: %q", out.String())
	}
}

func TestWallGlyph(t *testing.T) {
	if WallGlyph(true, true, true, true) != '┼' || WallGlyph(false, true, false, true) != '─' || WallGlyph(true, false, true, false) != '│' {
		t.Fatal("wall junctions wrong")
	}
}

func TestBlink(t *testing.T) {
	b := NewBuffer(10, 2)
	var out bytes.Buffer
	b.PrintBlink(0, 0, "d1", Yellow, Black)
	b.Print(2, 0, " x", White, Black)
	b.Flush(&out)
	s := out.String()
	if !strings.Contains(s, "\x1b[0;5;1;33;40md1") {
		t.Fatalf("blink SGR missing: %q", s)
	}
	if !strings.Contains(s, "\x1b[0;37;40m x") {
		t.Fatalf("blink should reset for the next cell: %q", s)
	}
}
