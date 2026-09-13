// Config: cannibalcasa.ini next to the binary (seeded from
// cannibalcasa.example.ini by install-xtrn). [settings] is pacing and display,
// [server] is the optional cross-BBS neighborhood server.
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cannibalcasa/internal/game"
)

type config struct {
	DataDir   string
	Rules     game.Rules
	ColorMode string // "16" | "256"
	UTF8      bool
	Server    string
	Port      string
	Token     string
	BBSName   string
}

func iniPath() string {
	p := filepath.Join(exeDir(), "cannibalcasa.ini")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if _, err := os.Stat("cannibalcasa.ini"); err == nil {
		return "cannibalcasa.ini"
	}
	return p
}

// defaultDataDir is SBBSDATA/cannibalcasa when the BBS exports it (the shared
// per-BBS neighborhood every node sees), else ./data next to the binary.
func defaultDataDir() string {
	if d := os.Getenv("SBBSDATA"); d != "" {
		return filepath.Join(d, "cannibalcasa")
	}
	return filepath.Join(exeDir(), "data")
}

func loadConfig(path string) config {
	cfg := config{DataDir: defaultDataDir(), Rules: game.DefaultRules, ColorMode: "16", Port: "7710"}
	f, err := os.Open(path)
	if err != nil {
		logf("config: %v (using defaults)", err)
		return cfg
	}
	defer f.Close()
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if i := strings.Index(v, " ;"); i >= 0 { // trailing comment
			v = strings.TrimSpace(v[:i])
		}
		switch section + "." + k {
		case "settings.data_dir":
			cfg.DataDir = v
		case "settings.days_per_day":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.Rules.DaysPerDay = n
			}
		case "settings.day_bank_max":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.Rules.BankMax = n
			}
		case "settings.color_mode":
			if v == "256" {
				cfg.ColorMode = "256"
			}
		case "settings.charset":
			cfg.UTF8 = strings.EqualFold(v, "utf8") || strings.EqualFold(v, "utf-8")
		case "server.server":
			cfg.Server = v
		case "server.port":
			cfg.Port = v
		case "server.token":
			cfg.Token = v
		case "server.bbsname":
			cfg.BBSName = v
		}
	}
	return cfg
}
