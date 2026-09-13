// Package world is the cul-de-sac: every player's World, the shared Nextdoor
// feed, and neighbor-to-neighbor mail (casseroles and anonymous tips). Local
// runs it in-process against a directory, which is what the door uses when no
// server is configured and what the server wraps over TCP.
package world

import (
	"cannibalcasa/internal/sim"
)

// FeedEntry is one line of the neighborhood feed.
type FeedEntry struct {
	Time     int64  `json:"t"`
	Key      string `json:"key"`
	Handle   string `json:"handle"`
	BBS      string `json:"bbs"`
	Cannibal string `json:"cannibal"`
	Text     string `json:"text"`
	Kind     string `json:"kind"`
}

// Mail is something a neighbor left on your porch, delivered when you next
// join: a casserole (a dish) or an anonymous tip to the HOA (Heat).
type Mail struct {
	Time    int64  `json:"t"`
	From    string `json:"from"`
	FromKey string `json:"from_key"`
	Kind    string `json:"kind"` // casserole | tip
	Recipe  string `json:"recipe,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// Neighbor is a player as seen from the street.
type Neighbor struct {
	Key string `json:"key"`
	sim.Summary
}

// Neighborhood is the street view.
type Neighborhood struct {
	Neighbors []Neighbor  `json:"neighbors"`
	Feed      []FeedEntry `json:"feed"`
}

// Backend is what the door talks to. The clock is driven by Sync: the door
// calls it about once a second with its speed (0 paused, 1, 2, 4) and the
// backend advances the world by the wall-clock time elapsed, at the world's
// scale times that speed. Join does the long catch-up from the last session.
type Backend interface {
	Join(bbs, handle string) (*sim.World, []sim.Event, int, error)
	Sync(bbs, handle string, speed int) (*sim.World, error)
	Do(bbs, handle string, c sim.Command) (*sim.World, error)
	Neighborhood(bbs, handle string) (Neighborhood, error)
	Casserole(bbs, handle, toKey string, dishID int) (*sim.World, error)
	Tip(bbs, handle, toKey string) (*sim.World, error)
	Leave(bbs, handle string) error
	Close() error
}
