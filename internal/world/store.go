package world

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cannibalcasa/internal/game"
)

// Store is the on-disk neighborhood: one JSON per player, an append-only feed
// and an append-only mailbox per player, all under one directory. Appends are
// single writes with O_APPEND so nodes on one BBS can share the directory
// without a lock; a player's own file is only ever written by their session.
type Store struct {
	Dir string
}

const feedKeep = 60

// Key is the stable id of a player across BBSes: "<bbs>/<handle>", lowercased.
func Key(bbs, handle string) string {
	return strings.ToLower(strings.TrimSpace(bbs)) + "/" + strings.ToLower(strings.TrimSpace(handle))
}

func safe(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '.':
			b.WriteRune(c)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

func (st *Store) playerPath(key string) string {
	bbs, handle, _ := strings.Cut(key, "/")
	return filepath.Join(st.Dir, "players", safe(bbs)+"__"+safe(handle)+".json")
}

func (st *Store) mailPath(key string) string {
	return st.playerPath(key) + ".mail"
}

func (st *Store) feedPath() string { return filepath.Join(st.Dir, "feed.jsonl") }

// Load returns the player's state, or nil when they've never played.
func (st *Store) Load(key string) (*game.State, error) {
	data, err := os.ReadFile(st.playerPath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s game.State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Fixtures == nil {
		s.Fixtures = map[string]bool{}
	}
	if s.Recipes == nil {
		s.Recipes = map[string]bool{}
	}
	return &s, nil
}

// Save writes the state atomically (temp file + rename).
func (st *Store) Save(key string, s *game.State) error {
	p := st.playerPath(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// All lists every player on the street, plus the key each file belongs to.
func (st *Store) All() ([]Neighbor, error) {
	entries, err := os.ReadDir(filepath.Join(st.Dir, "players"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Neighbor
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(st.Dir, "players", e.Name()))
		if err != nil {
			continue
		}
		var s game.State
		if json.Unmarshal(data, &s) != nil || s.Handle == "" {
			continue
		}
		out = append(out, Neighbor{Key: Key(s.BBS, s.Handle), Summary: s.Summary()})
	}
	return out, nil
}

func appendLine(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

// Post adds a line to the neighborhood feed.
func (st *Store) Post(e FeedEntry) error {
	if e.Time == 0 {
		e.Time = time.Now().Unix()
	}
	return appendLine(st.feedPath(), e)
}

// Feed returns the newest entries, oldest first.
func (st *Store) Feed(n int) ([]FeedEntry, error) {
	f, err := os.Open(st.feedPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var all []FeedEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var e FeedEntry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			all = append(all, e)
		}
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}

// SendMail drops mail on a neighbor's porch.
func (st *Store) SendMail(toKey string, m Mail) error {
	if m.Time == 0 {
		m.Time = time.Now().Unix()
	}
	return appendLine(st.mailPath(toKey), m)
}

// TakeMail collects and clears a player's mailbox.
func (st *Store) TakeMail(key string) ([]Mail, error) {
	p := st.mailPath(key)
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	os.Remove(p)
	var out []Mail
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m Mail
		if json.Unmarshal([]byte(line), &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}
