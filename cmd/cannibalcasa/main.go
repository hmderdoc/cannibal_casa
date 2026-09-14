// Cannibal Casa - a BBS door (Synchronet/DOOR32) where the house is a machine
// for processing the neighbors: lure them in, keep them in cells, butcher,
// cook, sell, and dispose of what's left before the street notices. The
// simulation runs on one clock whether you're watching or not. Multiplayer:
// every player lives on one cul-de-sac, per BBS or across boards through
// cannibalcasa-server.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/ui"
)

var version = "dev"

func main() {
	logf("=== cannibal casa %s start: args=%v pid=%d ===", version, os.Args, os.Getpid())
	dropfile := "DOOR32.SYS"
	forceLocal, utf8, snapshot, snapMenu := false, false, false, false
	handleArg := ""
	cols, rows, snapDays, snapLevel := 0, 0, 0, 0
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch {
		case a == "-dropfile" && i+1 < len(os.Args):
			i++
			dropfile = os.Args[i]
		case strings.HasPrefix(a, "-dropfile="):
			dropfile = a[len("-dropfile="):]
		case a == "-handle" && i+1 < len(os.Args):
			i++
			handleArg = os.Args[i]
		case a == "-local":
			forceLocal = true
		case a == "-utf8":
			utf8 = true
		case a == "-snapshot":
			snapshot = true
		case a == "-snapshot-menu":
			snapshot = true
			snapMenu = true
		case a == "-snapshot-days" && i+1 < len(os.Args):
			i++
			snapDays, _ = strconv.Atoi(os.Args[i])
		case a == "-snapshot-level" && i+1 < len(os.Args):
			i++
			snapLevel, _ = strconv.Atoi(os.Args[i])
		case a == "-size" && i+1 < len(os.Args):
			i++
			if c, r, ok := strings.Cut(os.Args[i], "x"); ok {
				cols, _ = strconv.Atoi(c)
				rows, _ = strconv.Atoi(r)
			}
		case a == "-version":
			fmt.Println("cannibalcasa", version)
			return
		}
	}
	cfg := loadConfig(iniPath())
	if utf8 {
		cfg.UTF8 = true
	}
	bbs, handle := door32Identity(dropfile)
	if cfg.BBSName != "" {
		bbs = cfg.BBSName
	}
	if handleArg != "" {
		handle = handleArg
	}
	if handle == "" {
		handle = "guest"
	}

	if snapshot {
		runSnapshot(cfg, bbs, handle, cols, rows, snapDays, snapMenu, snapLevel)
		return
	}

	term, restore, err := openTerm(dropfile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannibalcasa: cannot open terminal:", err)
		os.Exit(1)
	}
	defer term.Close()
	if cols == 0 || rows == 0 {
		cols, rows = 80, 24
		if c, r, ok := localTermSize(); ok {
			cols, rows = c, r
		} else if c, r, ok := probeSize(term, 600*time.Millisecond); ok {
			cols, rows = c, r
		}
	}
	if cols < 80 {
		cols = 80
	}
	if rows < 24 {
		rows = 24
	}
	logf("terminal %dx%d bbs=%q handle=%q", cols, rows, bbs, handle)

	w := bufio.NewWriterSize(term, 64*1024)
	w.WriteString("\x1b[?25l\x1b[?7l")
	defer func() { w.WriteString("\x1b[0m\x1b[?25h\x1b[?7h\x1b[2J\x1b[H"); w.Flush(); restore() }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, shutdownSignals...)

	buf := ui.NewBuffer(cols, rows)
	buf.UTF8 = cfg.UTF8
	if cfg.ColorMode == "256" {
		buf.Mode = ui.Mode256
	}
	in := newInput(term)

	be, status := pickBackend(cfg, bbs, handle, forceLocal)
	defer be.Close()
	runSplash(w, buf, in, status)

	a := newApp(be, bbs, handle, cfg, buf)
	a.note("info", "%s", status)
	a.join()
	defer be.Leave(bbs, handle)
	buf.Invalidate()

	syncTick := time.NewTicker(1000 * time.Millisecond)
	defer syncTick.Stop()
	resizeTick := time.NewTicker(3 * time.Second)
	defer resizeTick.Stop()
	for !a.quit {
		a.render()
		buf.Flush(w)
		w.Flush()
		select {
		case k := <-in.events:
			a.handleKey(k)
		case <-syncTick.C:
			a.sync()
		case <-in.quitCh:
			logf("input closed; exiting")
			return
		case <-sig:
			logf("signal; exiting")
			return
		case sz := <-in.resizeCh:
			if sz[0] != buf.W || sz[1] != buf.H {
				logf("resize to %dx%d", sz[0], sz[1])
				buf.Resize(max(sz[0], 80), max(sz[1], 24))
			}
		case <-resizeTick.C:
			w.WriteString("\x1b[18t")
			w.Flush()
		}
	}
	logf("clean exit")
}

// runSnapshot renders one frame of the main screen as UTF-8 ANSI to stdout,
// after running the house for -snapshot-days, for eyeballing the layout.
func runSnapshot(cfg config, bbs, handle string, cols, rows, days int, menu bool, level int) {
	if cols == 0 {
		cols, rows = 80, 24
	}
	buf := ui.NewBuffer(cols, rows)
	buf.UTF8 = true
	be, status := pickBackend(cfg, bbs, handle, true)
	defer be.Close()
	a := newApp(be, bbs, handle, cfg, buf)
	a.note("info", "%s", status)
	a.join()
	a.modal = nil
	if days > 0 && a.w != nil {
		a.w.Cash = 2000
		for _, id := range []string{"lure_couch", "cell", "grinder", "microwave", "compost", "fence", "chair", "cot", "smoker", "booth"} {
			f := sim.FixtureByID(id)
			p := sim.Plans[f.Level]
			for y := 0; y < len(p.Rows); y++ {
				placed := false
				for x := 0; x < len(p.Rows[y]); x++ {
					if a.w.Build(id, sim.Pos{L: f.Level, X: x, Y: y}) == nil {
						placed = true
						break
					}
				}
				if placed {
					break
				}
			}
		}
		a.w.Policy.Brainwash = true
		for d := 0; d < days; d++ {
			a.w.Advance(1440)
		}
		a.w.Advance(sim.DefaultScale) // land at a random-ish minute
	}
	if menu && a.w != nil {
		a.openBuild("")
	}
	a.level = sim.Level(level)
	w := bufio.NewWriter(os.Stdout)
	a.render()
	buf.Flush(w)
	w.WriteString("\x1b[0m\n")
	w.Flush()
}
