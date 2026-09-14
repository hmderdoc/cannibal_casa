// Package ui is the door's drawing layer: a cell framebuffer where every cell
// carries a glyph, a foreground and a background; a diff renderer that emits
// the minimal ANSI to bring the terminal up to date; a semantic palette that
// resolves to 16 or 256 colors; and CP437 output for BBS terminals (UTF-8 on
// request for a local shell).
package ui

import (
	"bytes"
	"io"
	"strconv"
)

// Attr is one color in both depths: C16 is the classic 0..15 (8..15 bright),
// C256 the xterm index used when the caller's terminal can take it.
type Attr struct {
	C16  uint8
	C256 uint8
}

// Cell is one screen position.
type Cell struct {
	Ch    rune
	Fg    Attr
	Bg    Attr
	Blink bool
}

// Color depth modes.
const (
	Mode16 = iota
	Mode256
)

// Buffer is a double-buffered screen. Draw into it, then Flush the diff.
type Buffer struct {
	W, H  int
	Mode  int
	UTF8  bool
	cells []Cell
	prev  []Cell
	dirty bool // prev is invalid: next Flush repaints everything
}

func NewBuffer(w, h int) *Buffer {
	b := &Buffer{}
	b.Resize(w, h)
	return b
}

// Resize discards both buffers; the next Flush is a full repaint.
func (b *Buffer) Resize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	b.W, b.H = w, h
	b.cells = make([]Cell, w*h)
	b.prev = make([]Cell, w*h)
	b.dirty = true
	b.Clear(Black)
}

// Invalidate forces the next Flush to repaint every cell.
func (b *Buffer) Invalidate() { b.dirty = true }

func (b *Buffer) Clear(bg Attr) {
	for i := range b.cells {
		b.cells[i] = Cell{Ch: ' ', Fg: White, Bg: bg}
	}
}

func (b *Buffer) Set(x, y int, ch rune, fg, bg Attr) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	b.cells[y*b.W+x] = Cell{Ch: ch, Fg: fg, Bg: bg}
}

// SetBlink is Set with the blink attribute (SGR 5). Use it for one thing at
// a time: on terminals with iCE colors enabled it reads as a bright
// background instead of blinking, which is still an accent.
func (b *Buffer) SetBlink(x, y int, ch rune, fg, bg Attr) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	b.cells[y*b.W+x] = Cell{Ch: ch, Fg: fg, Bg: bg, Blink: true}
}

// PrintBlink writes a blinking string and returns the x after it.
func (b *Buffer) PrintBlink(x, y int, s string, fg, bg Attr) int {
	for _, ch := range s {
		if x >= b.W {
			break
		}
		b.SetBlink(x, y, ch, fg, bg)
		x++
	}
	return x
}

// Get returns the cell at x,y (a blank cell when out of range).
func (b *Buffer) Get(x, y int) Cell {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return Cell{Ch: ' '}
	}
	return b.cells[y*b.W+x]
}

// Print writes a string left to right, clipping at the right edge, and
// returns the x after the last glyph.
func (b *Buffer) Print(x, y int, s string, fg, bg Attr) int {
	for _, ch := range s {
		if x >= b.W {
			break
		}
		b.Set(x, y, ch, fg, bg)
		x++
	}
	return x
}

// PrintN prints at most n glyphs, padding with spaces to exactly n.
func (b *Buffer) PrintN(x, y int, s string, n int, fg, bg Attr) {
	i := 0
	for _, ch := range s {
		if i >= n {
			break
		}
		b.Set(x+i, y, ch, fg, bg)
		i++
	}
	for ; i < n; i++ {
		b.Set(x+i, y, ' ', fg, bg)
	}
}

func (b *Buffer) Fill(x, y, w, h int, ch rune, fg, bg Attr) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			b.Set(xx, yy, ch, fg, bg)
		}
	}
}

// Box draws a single-line frame with an optional title on the top edge.
func (b *Buffer) Box(x, y, w, h int, fg, bg Attr, title string, titleFg Attr) {
	if w < 2 || h < 2 {
		return
	}
	b.Set(x, y, '┌', fg, bg)
	b.Set(x+w-1, y, '┐', fg, bg)
	b.Set(x, y+h-1, '└', fg, bg)
	b.Set(x+w-1, y+h-1, '┘', fg, bg)
	for xx := x + 1; xx < x+w-1; xx++ {
		b.Set(xx, y, '─', fg, bg)
		b.Set(xx, y+h-1, '─', fg, bg)
	}
	for yy := y + 1; yy < y+h-1; yy++ {
		b.Set(x, yy, '│', fg, bg)
		b.Set(x+w-1, yy, '│', fg, bg)
	}
	if title != "" && w > 4 {
		t := " " + title + " "
		if len([]rune(t)) > w-2 {
			t = string([]rune(t)[:w-2])
		}
		b.Print(x+1, y, t, titleFg, bg)
	}
}

// Bar draws a meter: label, then a filled/empty track w cells wide.
func (b *Buffer) Bar(x, y, w int, val, max int, fill, empty, bg Attr) {
	if max <= 0 {
		max = 1
	}
	if val < 0 {
		val = 0
	}
	if val > max {
		val = max
	}
	n := val * w / max
	for i := 0; i < w; i++ {
		if i < n {
			b.Set(x+i, y, '█', fill, bg)
		} else {
			b.Set(x+i, y, '░', empty, bg)
		}
	}
}

// Flush writes the difference between the last flushed frame and the current
// one. The cursor is left wherever the last change was; callers hide it.
func (b *Buffer) Flush(w io.Writer) error {
	var out bytes.Buffer
	full := b.dirty
	if full {
		out.WriteString("\x1b[0m\x1b[2J\x1b[H")
	}
	curFg, curBg := Attr{255, 255}, Attr{255, 255}
	curBlink := false
	curX, curY := -1, -1
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			i := y*b.W + x
			c := b.cells[i]
			if !full && c == b.prev[i] {
				continue
			}
			// Never write the bottom-right cell: on many terminals it scrolls.
			if y == b.H-1 && x == b.W-1 {
				continue
			}
			if curX != x || curY != y {
				out.WriteString("\x1b[")
				out.WriteString(strconv.Itoa(y + 1))
				out.WriteByte(';')
				out.WriteString(strconv.Itoa(x + 1))
				out.WriteByte('H')
			}
			if c.Fg != curFg || c.Bg != curBg || c.Blink != curBlink {
				out.WriteString(b.sgr(c.Fg, c.Bg, c.Blink))
				curFg, curBg, curBlink = c.Fg, c.Bg, c.Blink
			}
			b.encode(&out, c.Ch)
			curX, curY = x+1, y
			b.prev[i] = c
		}
	}
	b.dirty = false
	if out.Len() == 0 {
		return nil
	}
	_, err := w.Write(out.Bytes())
	return err
}

// sgr builds the color escape for a fg/bg pair in the buffer's depth. In 16
// color mode bright foregrounds use bold (the portable spelling) and
// backgrounds are limited to the 8 dark colors, which every BBS terminal has.
func (b *Buffer) sgr(fg, bg Attr, blink bool) string {
	bl := ""
	if blink {
		bl = "5;"
	}
	if b.Mode == Mode256 {
		return "\x1b[0;" + bl + "38;5;" + strconv.Itoa(int(fg.C256)) + ";48;5;" + strconv.Itoa(int(bg.C256)) + "m"
	}
	s := "\x1b[0;" + bl
	if fg.C16 >= 8 {
		s += "1;"
	}
	s += strconv.Itoa(30+int(fg.C16&7)) + ";" + strconv.Itoa(40+int(bg.C16&7)) + "m"
	return s
}

func (b *Buffer) encode(out *bytes.Buffer, ch rune) {
	if ch < 0x20 {
		ch = ' '
	}
	if ch < 0x7f {
		out.WriteByte(byte(ch))
		return
	}
	if b.UTF8 {
		out.WriteRune(ch)
		return
	}
	if c, ok := cp437[ch]; ok {
		out.WriteByte(c)
		return
	}
	out.WriteByte('?')
}

// cp437 maps the non-ASCII glyphs the door draws to their CP437 byte.
var cp437 = map[rune]byte{
	'─': 0xC4, '│': 0xB3, '┌': 0xDA, '┐': 0xBF, '└': 0xC0, '┘': 0xD9,
	'├': 0xC3, '┤': 0xB4, '┬': 0xC2, '┴': 0xC1, '┼': 0xC5,
	'═': 0xCD, '║': 0xBA, '╔': 0xC9, '╗': 0xBB, '╚': 0xC8, '╝': 0xBC,
	'╠': 0xCC, '╣': 0xB9, '╦': 0xCB, '╩': 0xCA, '╬': 0xCE,
	'█': 0xDB, '▄': 0xDC, '▀': 0xDF, '▌': 0xDD, '▐': 0xDE,
	'░': 0xB0, '▒': 0xB1, '▓': 0xB2, '■': 0xFE, '·': 0xFA, '∙': 0xF9,
	'≡': 0xF0, '±': 0xF1, '°': 0xF8, '√': 0xFB, '²': 0xFD, '£': 0x9C, '¥': 0x9D, '¢': 0x9B,
	'½': 0xAB, '¼': 0xAC, '«': 0xAE, '»': 0xAF, '¡': 0xAD, '¿': 0xA8,
	'α': 0xE0, 'ß': 0xE1, 'Γ': 0xE2, 'π': 0xE3, 'Σ': 0xE4, 'σ': 0xE5, 'µ': 0xE6, 'τ': 0xE7,
	'Φ': 0xE8, 'Θ': 0xE9, 'Ω': 0xEA, 'δ': 0xEB, '∞': 0xEC, 'φ': 0xED, 'ε': 0xEE, '∩': 0xEF,
	'é': 0x82, 'ñ': 0xA4, 'ü': 0x81, 'ö': 0x94, 'ä': 0x84,
}

// WallGlyph picks the box-drawing piece for a wall tile from which of its four
// neighbors are also walls.
func WallGlyph(n, e, s, w bool) rune {
	switch {
	case n && s && e && w:
		return '┼'
	case n && s && e:
		return '├'
	case n && s && w:
		return '┤'
	case e && w && s:
		return '┬'
	case e && w && n:
		return '┴'
	case s && e:
		return '┌'
	case s && w:
		return '┐'
	case n && e:
		return '└'
	case n && w:
		return '┘'
	case n || s:
		return '│'
	case e || w:
		return '─'
	}
	return '■'
}
