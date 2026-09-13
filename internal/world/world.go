// Package world is the cul-de-sac: every player's State, the shared Nextdoor
// feed, and neighbor-to-neighbor mail (casseroles and anonymous tips). Local
// runs it in-process against a directory, which is what the door uses when no
// server is configured and what the server wraps over TCP. Same rules, same
// files, either way.
package world

import (
	"cannibalcasa/internal/game"
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
	From    string `json:"from"` // sender's cannibal name
	FromKey string `json:"from_key"`
	Kind    string `json:"kind"` // casserole | tip
	Recipe  string `json:"recipe,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

// Neighbor is a player as seen from the street.
type Neighbor struct {
	Key string `json:"key"`
	game.Summary
}

// Neighborhood is the street view: who lives here and what's been said.
type Neighborhood struct {
	Neighbors []Neighbor  `json:"neighbors"`
	Feed      []FeedEntry `json:"feed"`
}

// Backend is what the door talks to. Local implements it in-process; the
// door's remote client implements it over the wire; the server bridges the two.
type Backend interface {
	// Join loads (or creates) the player, grants the daily allowance and
	// delivers any mail. Events describe what changed since last time.
	Join(bbs, handle string) (*game.State, []game.Event, error)
	// Do applies one action and persists the result.
	Do(bbs, handle string, a game.Action) (*game.State, game.Result, error)
	// Neighborhood returns the street: everyone's summary plus the feed.
	Neighborhood(bbs, handle string) (Neighborhood, error)
	// Casserole sends one of the player's dishes to a neighbor by key.
	Casserole(bbs, handle, toKey string, dishID int) (*game.State, game.Result, error)
	// Tip phones in an anonymous complaint about a neighbor. Once a day; can backfire.
	Tip(bbs, handle, toKey string) (*game.State, game.Result, error)
	Close() error
}
