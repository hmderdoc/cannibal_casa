package world

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cannibalcasa/internal/sim"
)

// Local is the in-process Backend over a Store. Worlds stay loaded between
// calls while a player is present, and are saved on every command, every
// saveEvery of wall-clock time, and on Leave.
type Local struct {
	Store Store
	Scale int
	Now   func() time.Time
	mu    sync.Mutex
	live  map[string]*live
}

type live struct {
	w     *sim.World
	saved time.Time
}

const saveEvery = 20 * time.Second

func NewLocal(dir string, scale int) *Local {
	if scale <= 0 {
		scale = sim.DefaultScale
	}
	return &Local{Store: Store{Dir: dir}, Scale: scale, Now: time.Now, live: map[string]*live{}}
}

func (l *Local) today() string { return l.Now().Format("2006-01-02") }

// get returns the loaded world for a key, loading it if needed.
func (l *Local) get(bbs, handle string) (*live, string, error) {
	key := Key(bbs, handle)
	if lv, ok := l.live[key]; ok {
		return lv, key, nil
	}
	w, err := l.Store.Load(key)
	if err != nil {
		return nil, key, err
	}
	if w == nil {
		return nil, key, fmt.Errorf("no such player: %s", key)
	}
	lv := &live{w: w, saved: l.Now()}
	l.live[key] = lv
	return lv, key, nil
}

// flush posts pending feed lines and saves.
func (l *Local) flush(key string, lv *live, force bool) error {
	w := lv.w
	for _, p := range w.Posts {
		l.Store.Post(FeedEntry{Key: key, Handle: w.Handle, BBS: w.BBS, Cannibal: w.Cannibal, Text: p.Text, Kind: p.Kind})
	}
	w.Posts = nil
	if force || l.Now().Sub(lv.saved) >= saveEvery {
		lv.saved = l.Now()
		return l.Store.Save(key, w)
	}
	return nil
}

func (l *Local) Join(bbs, handle string) (*sim.World, []sim.Event, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := Key(bbs, handle)
	var events []sim.Event
	lv, ok := l.live[key]
	if !ok {
		w, err := l.Store.Load(key)
		if err != nil {
			return nil, nil, 0, err
		}
		if w == nil {
			w = sim.New(handle, bbs, l.Now().UnixNano()^rand.Int63())
			w.Scale = l.Scale
			events = append(events, sim.Event{Kind: "info", Text: "A new family moves onto the cul-de-sac. The welcome basket is muffins. Yours will not be."})
			l.Store.Post(FeedEntry{Key: key, Handle: handle, BBS: bbs, Cannibal: w.Cannibal, Kind: "info",
				Text: "moved in. Waved at everyone. Asked about the neighbors' 'dietary restrictions'."})
		}
		lv = &live{w: w, saved: l.Now()}
		l.live[key] = lv
	}
	w := lv.w
	ticks := w.Catchup(l.Now())
	mail, _ := l.Store.TakeMail(key)
	for _, m := range mail {
		switch m.Kind {
		case "casserole":
			rc := sim.RecipeByID(m.Recipe)
			if rc == nil {
				continue
			}
			w.NextID++
			if len(w.Pantry) < w.PantryCap() {
				w.Pantry = append(w.Pantry, sim.Item{ID: w.NextID, Kind: "dish", Sub: m.Recipe, Quality: m.Quality, Born: w.Clock})
				events = append(events, sim.Event{Kind: "food", Text: fmt.Sprintf("%s left a %s on the porch. Neighborly. Suspiciously neighborly.", m.From, rc.Name)})
			}
		case "tip":
			w.Heat += 10
			if w.Heat > 120 {
				w.Heat = 120
			}
			events = append(events, sim.Event{Kind: "heat", Text: "Somebody phoned the HOA about 'smells' from your garage. Anonymous. Heat +10."})
		}
	}
	if err := l.flush(key, lv, true); err != nil {
		return nil, nil, 0, err
	}
	return w, events, ticks, nil
}

func (l *Local) Sync(bbs, handle string, speed int) (*sim.World, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lv, key, err := l.get(bbs, handle)
	if err != nil {
		return nil, err
	}
	lv.w.Sync(l.Now(), speed)
	if err := l.flush(key, lv, false); err != nil {
		return nil, err
	}
	return lv.w, nil
}

func (l *Local) Do(bbs, handle string, c sim.Command) (*sim.World, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lv, key, err := l.get(bbs, handle)
	if err != nil {
		return nil, err
	}
	if err := lv.w.Apply(c); err != nil {
		return lv.w, err
	}
	if err := l.flush(key, lv, true); err != nil {
		return nil, err
	}
	return lv.w, nil
}

func (l *Local) Neighborhood(bbs, handle string) (Neighborhood, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Make sure our own latest state is on disk so the listing includes it.
	if lv, ok := l.live[Key(bbs, handle)]; ok {
		l.flush(Key(bbs, handle), lv, true)
	}
	ns, err := l.Store.All()
	if err != nil {
		return Neighborhood{}, err
	}
	feed, err := l.Store.Feed(feedKeep)
	if err != nil {
		return Neighborhood{}, err
	}
	return Neighborhood{Neighbors: ns, Feed: feed}, nil
}

func (l *Local) Casserole(bbs, handle, toKey string, dishID int) (*sim.World, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lv, key, err := l.get(bbs, handle)
	if err != nil {
		return nil, err
	}
	w := lv.w
	if toKey == key {
		return w, fmt.Errorf("you can't send yourself a casserole; you can just eat it")
	}
	to, err := l.Store.Load(toKey)
	if err != nil || to == nil {
		return w, fmt.Errorf("no such neighbor")
	}
	for i, d := range w.Pantry {
		if d.ID != dishID {
			continue
		}
		rc := sim.RecipeByID(d.Sub)
		if !rc.Food {
			return w, fmt.Errorf("that's not food, even by your standards")
		}
		w.Pantry = append(w.Pantry[:i], w.Pantry[i+1:]...)
		l.Store.SendMail(toKey, Mail{From: w.Cannibal, FromKey: key, Kind: "casserole", Recipe: d.Sub, Quality: d.Quality})
		w.Posts = append(w.Posts, sim.Event{Kind: "good", Text: fmt.Sprintf("dropped off a %s at %s's place. 'Family recipe.'", rc.Name, to.Cannibal)})
		l.flush(key, lv, true)
		return w, nil
	}
	return w, fmt.Errorf("no such dish")
}

func (l *Local) Tip(bbs, handle, toKey string) (*sim.World, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lv, key, err := l.get(bbs, handle)
	if err != nil {
		return nil, err
	}
	w := lv.w
	if toKey == key {
		return w, fmt.Errorf("reporting yourself is a bold strategy")
	}
	to, err := l.Store.Load(toKey)
	if err != nil || to == nil {
		return w, fmt.Errorf("no such neighbor")
	}
	l.Store.SendMail(toKey, Mail{From: w.Cannibal, FromKey: key, Kind: "tip"})
	if rand.Intn(100) < 35 {
		w.Heat += 8
		w.Posts = append(w.Posts, sim.Event{Kind: "heat", Text: fmt.Sprintf("tried to snitch on %s and got traced. Awkward at the mailbox now.", to.Cannibal)})
		l.flush(key, lv, true)
		return w, fmt.Errorf("the hotline has caller ID: Heat +8 for you")
	}
	l.flush(key, lv, true)
	return w, nil
}

func (l *Local) Leave(bbs, handle string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := Key(bbs, handle)
	lv, ok := l.live[key]
	if !ok {
		return nil
	}
	err := l.flush(key, lv, true)
	delete(l.live, key)
	return err
}

func (l *Local) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var first error
	for key, lv := range l.live {
		if err := l.flush(key, lv, true); err != nil && first == nil {
			first = err
		}
	}
	l.live = map[string]*live{}
	return first
}
