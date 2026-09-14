package ui

// The 16 classic colors, with a hand-picked xterm-256 twin for each.
var (
	Black         = Attr{0, 16}
	Red           = Attr{1, 124}
	Green         = Attr{2, 28}
	Brown         = Attr{3, 130}
	Blue          = Attr{4, 19}
	Magenta       = Attr{5, 90}
	Cyan          = Attr{6, 30}
	White         = Attr{7, 250}
	Gray          = Attr{8, 240}
	BrightRed     = Attr{9, 196}
	BrightGreen   = Attr{10, 82}
	Yellow        = Attr{11, 220}
	BrightBlue    = Attr{12, 33}
	BrightMagenta = Attr{13, 201}
	BrightCyan    = Attr{14, 51}
	BrightWhite   = Attr{15, 231}
)

// Extra hues that only exist in 256-color mode; they fall back to the nearest
// classic color on a 16-color terminal.
var (
	Orange   = Attr{11, 208}
	Gold     = Attr{11, 178}
	Pink     = Attr{13, 213}
	Blood    = Attr{1, 88}
	Meat     = Attr{9, 167}
	Grass    = Attr{2, 34}
	Dirt     = Attr{3, 94}
	Concrete = Attr{8, 238}
	Wood     = Attr{3, 137}
	Dusk     = Attr{4, 17}
	Slate    = Attr{8, 236}
	Teal     = Attr{6, 37}
	Lime     = Attr{10, 118}
	Lavender = Attr{13, 141}
)

// base16 lists the classic colors by index so a hue's counterpart can be found.
var base16 = [16]Attr{Black, Red, Green, Brown, Blue, Magenta, Cyan, White,
	Gray, BrightRed, BrightGreen, Yellow, BrightBlue, BrightMagenta, BrightCyan, BrightWhite}

// Counterpart is the same hue at the other intensity: red for light red,
// light red for red, brown for yellow, and so on. Extra 256-color hues map
// through their classic twin.
func (a Attr) Counterpart() Attr {
	return base16[(a.C16^8)&15]
}

// Palette is the semantic color table. Game data names colors by role
// ("victim_prime", "fix_cold"); the door looks them up here so a re-theme is
// one table edit.
var Palette = map[string]Attr{
	// world
	"wall":           White,
	"wall_basement":  Concrete,
	"wall_yard":      Wood,
	"wall_attic":     Dirt,
	"floor_ground":   Slate,
	"floor_basement": Dusk,
	"floor_yard":     Grass,
	"floor_attic":    Dirt,
	"door":           Brown,
	"street":         Slate,
	"sidewalk":       Concrete,
	"cursor":         BrightWhite,
	"busy":           Yellow,
	"full":           Orange,
	// backgrounds (16-color mode can only use the 8 dark colors)
	"bg_ground":   Attr{0, 235},
	"bg_basement": Attr{4, 17},
	"bg_yard":     Attr{2, 22},
	"bg_attic":    Attr{3, 58},
	"bg_porch":    Attr{0, 236},
	"bg_busy":     Attr{4, 24},
	"bg_full":     Attr{1, 88},
	"bg_cell":     Attr{1, 52},
	"bg_ok":       Attr{2, 28},
	"bg_bad":      Attr{1, 124},
	"bg_sidewalk": Attr{0, 237},
	"player":      Yellow,
	"slot_empty":  Gray,
	"label":       Gray,
	// fixtures
	"fix_kitchen": BrightWhite,
	"fix_cold":    BrightCyan,
	"fix_living":  Lavender,
	"fix_closet":  BrightMagenta,
	"fix_garage":  White,
	"fix_lure":    Gold,
	"fix_cell":    BrightBlue,
	"fix_yard":    Lime,
	"fix_attic":   Orange,
	// victims
	"victim_easy":  BrightGreen,
	"victim_mid":   Yellow,
	"victim_hard":  BrightRed,
	"victim_prime": Pink,
	"victim_heat":  BrightCyan,
	// chrome
	"title":    BrightRed,
	"text":     BrightWhite, // content
	"bright":   BrightWhite,
	"dim":      Gray,        // hints and labels only
	"flavor":   Lavender,    // the joke line
	"rule":     Gold,        // placement rule, footers of menus
	"strip_bg": Attr{4, 17}, // title and status strips
	"strip_fg": BrightWhite,
	"alert_bg": Attr{1, 88},
	// modals: their own world, not the map's
	"modal_bg":     Attr{4, 18},
	"modal_fg":     BrightWhite,
	"modal_border": BrightWhite,
	"modal_title":  Yellow,
	"modal_tbg":    Attr{4, 18},
	"modal_sel_bg": Attr{6, 37},
	"modal_sel_fg": Black,
	"modal_desc":   Attr{0, 234},
	"modal_hint":   BrightCyan,
	"calm_bg":      Attr{2, 22},
	"border":       Blood,
	"key":          Yellow,
	"sel_bg":       Blood,
	"sel_fg":       BrightWhite,
	"hunger":       Meat,
	"cash":         Gold,
	"stars":        Yellow,
	"heat_low":     BrightGreen,
	"heat_mid":     Orange,
	"heat_hi":      BrightRed,
	"bar_off":      Slate,
	// log kinds
	"info":    BrightWhite,
	"good":    BrightGreen,
	"bad":     BrightRed,
	"heat":    BrightMagenta,
	"cash_ev": Gold,
	"food":    BrightCyan,
	"warn":    Orange,
}

// Col resolves a palette name, falling back to plain white.
func Col(name string) Attr {
	if a, ok := Palette[name]; ok {
		return a
	}
	return White
}
