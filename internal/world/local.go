package world

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"cannibalcasa/internal/game"
)

// Local is the in-process Backend over a Store. One mutex serializes every
// call: the rules run in microseconds, so contention is a non-issue and the
// feed/mail files stay consistent.
type Local struct {
	Store Store
	Rules game.Rules
	Now   func() time.Time // injectable for tests
	mu    sync.Mutex
}

// NewLocal opens (or creates) a neighborhood directory.
func NewLocal(dir string, rules game.Rules) *Local {
	return &Local{Store: Store{Dir: dir}, Rules: rules, Now: time.Now}
}

func (l *Local) today() string { return l.Now().Format("2006-01-02") }

func (l *Local) load(bbs, handle string) (*game.State, string, error) {
	key := Key(bbs, handle)
	s, err := l.Store.Load(key)
	if err != nil {
		return nil, key, err
	}
	if s == nil {
		return nil, key, fmt.Errorf("no such player: %s", key)
	}
	return s, key, nil
}

func (l *Local) Join(bbs, handle string) (*game.State, []game.Event, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := Key(bbs, handle)
	s, err := l.Store.Load(key)
	if err != nil {
		return nil, nil, err
	}
	var events []game.Event
	if s == nil {
		s = game.NewState(handle, bbs, l.Now().UnixNano()^rand.Int63())
		events = append(events, game.Event{Kind: "info", Text: "A new family moves onto the cul-de-sac. The welcome basket is muffins. Yours will not be."})
		l.Store.Post(FeedEntry{Key: key, Handle: handle, BBS: bbs, Cannibal: s.Cannibal, Kind: "info",
			Text: "moved in. Waved at everyone. Asked about the neighbors' 'dietary restrictions'."})
	}
	if game.GrantAllowance(s, l.Rules, l.today()) {
		events = append(events, game.Event{Kind: "good", Text: fmt.Sprintf("A new real day: +%d in-game days to spend (%d banked).", l.Rules.DaysPerDay, s.Bank)})
	}
	mail, _ := l.Store.TakeMail(key)
	for _, m := range mail {
		switch m.Kind {
		case "casserole":
			rc := game.RecipeByID(m.Recipe)
			if rc == nil {
				continue
			}
			s.NextID++
			s.Dishes = append(s.Dishes, game.Dish{ID: s.NextID, Recipe: m.Recipe, Quality: m.Quality, Shelf: 3})
			events = append(events, game.Event{Kind: "food", Text: fmt.Sprintf("%s left a %s on the porch. Neighborly. Suspiciously neighborly.", m.From, rc.Name)})
		case "tip":
			s.Heat += 10
			if s.Heat > 120 {
				s.Heat = 120
			}
			events = append(events, game.Event{Kind: "heat", Text: "Somebody phoned the HOA about 'smells' from your garage. Anonymous. Heat +10."})
		}
	}
	if err := l.Store.Save(key, s); err != nil {
		return nil, nil, err
	}
	return s, events, nil
}

func (l *Local) Do(bbs, handle string, a game.Action) (*game.State, game.Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, key, err := l.load(bbs, handle)
	if err != nil {
		return nil, game.Result{}, err
	}
	r := game.Apply(s, a)
	if r.Err != "" {
		return s, r, nil
	}
	for _, p := range r.Posts {
		l.Store.Post(FeedEntry{Key: key, Handle: handle, BBS: bbs, Cannibal: s.Cannibal, Text: p.Text, Kind: p.Kind})
	}
	if err := l.Store.Save(key, s); err != nil {
		return nil, game.Result{}, err
	}
	return s, r, nil
}

func (l *Local) Neighborhood(bbs, handle string) (Neighborhood, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
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

func (l *Local) Casserole(bbs, handle, toKey string, dishID int) (*game.State, game.Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, key, err := l.load(bbs, handle)
	if err != nil {
		return nil, game.Result{}, err
	}
	r := game.Result{}
	if toKey == key {
		r.Err = "You can't send yourself a casserole. You can just eat it."
		return s, r, nil
	}
	to, err := l.Store.Load(toKey)
	if err != nil || to == nil {
		r.Err = "No such neighbor."
		return s, r, nil
	}
	for i, d := range s.Dishes {
		if d.ID != dishID {
			continue
		}
		rc := game.RecipeByID(d.Recipe)
		if !rc.Food {
			r.Err = "That's not food. Even by your standards."
			return s, r, nil
		}
		s.Dishes = append(s.Dishes[:i], s.Dishes[i+1:]...)
		l.Store.SendMail(toKey, Mail{From: s.Cannibal, FromKey: key, Kind: "casserole", Recipe: d.Recipe, Quality: d.Quality})
		r.Events = append(r.Events, game.Event{Kind: "good", Text: fmt.Sprintf("Left a %s on %s's porch. What a neighbor you are.", rc.Name, to.Cannibal)})
		l.Store.Post(FeedEntry{Key: key, Handle: handle, BBS: bbs, Cannibal: s.Cannibal, Kind: "good",
			Text: fmt.Sprintf("dropped off a %s at %s's place. 'Family recipe.'", rc.Name, to.Cannibal)})
		if err := l.Store.Save(key, s); err != nil {
			return nil, game.Result{}, err
		}
		return s, r, nil
	}
	r.Err = "No such dish."
	return s, r, nil
}

func (l *Local) Tip(bbs, handle, toKey string) (*game.State, game.Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, key, err := l.load(bbs, handle)
	if err != nil {
		return nil, game.Result{}, err
	}
	r := game.Result{}
	if toKey == key {
		r.Err = "Reporting yourself is a bold strategy."
		return s, r, nil
	}
	if s.LastTip == l.today() {
		r.Err = "The HOA hotline knows your voice by now. One tip a day."
		return s, r, nil
	}
	to, err := l.Store.Load(toKey)
	if err != nil || to == nil {
		r.Err = "No such neighbor."
		return s, r, nil
	}
	s.LastTip = l.today()
	l.Store.SendMail(toKey, Mail{From: s.Cannibal, FromKey: key, Kind: "tip"})
	r.Events = append(r.Events, game.Event{Kind: "info", Text: fmt.Sprintf("You called the HOA about %s's garage. Anonymously. Mostly.", to.Cannibal)})
	if rand.Intn(100) < 35 {
		s.Heat += 8
		r.Events = append(r.Events, game.Event{Kind: "bad", Text: "The hotline has caller ID. Heat +8. Worth it?"})
		l.Store.Post(FeedEntry{Key: key, Handle: handle, BBS: bbs, Cannibal: s.Cannibal, Kind: "heat",
			Text: fmt.Sprintf("tried to snitch on %s and got traced. Awkward at the mailbox now.", to.Cannibal)})
	}
	if err := l.Store.Save(key, s); err != nil {
		return nil, game.Result{}, err
	}
	return s, r, nil
}

func (l *Local) Close() error { return nil }
