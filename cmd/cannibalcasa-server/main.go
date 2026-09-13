// cannibalcasa-server: the neighborhood server. It owns the cul-de-sac (every
// player's state, the feed, the mail) and applies the rules; doors on any BBS
// connect out to it with a shared token and send actions. Run it yourself,
// e.g. via a systemd unit; see cannibalcasa-server.service.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/proto"
	"cannibalcasa/internal/world"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":7710", "listen address")
	token := flag.String("token", "", "shared secret every door must present (required)")
	dataDir := flag.String("data", "", "neighborhood data directory (default: SBBSDATA/cannibalcasa or ./data)")
	daysPerDay := flag.Int("days-per-day", game.DefaultRules.DaysPerDay, "in-game days granted per real day")
	bankMax := flag.Int("day-bank-max", game.DefaultRules.BankMax, "ceiling on banked days")
	flag.Parse()
	if *token == "" {
		fmt.Fprintln(os.Stderr, "cannibalcasa-server: -token is required")
		os.Exit(2)
	}
	dir := *dataDir
	if dir == "" {
		if d := os.Getenv("SBBSDATA"); d != "" {
			dir = filepath.Join(d, "cannibalcasa")
		} else {
			dir = "data"
		}
	}
	w := world.NewLocal(dir, game.Rules{DaysPerDay: *daysPerDay, BankMax: *bankMax})
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	log.Printf("cannibalcasa-server %s listening on %s, data in %s", version, *addr, dir)
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go serve(c, w, *token)
	}
}

func serve(nc net.Conn, w *world.Local, token string) {
	defer nc.Close()
	c := proto.NewConn(nc)
	nc.SetDeadline(time.Now().Add(10 * time.Second))
	hello, err := c.ReadRequest()
	if err != nil || hello.Type != proto.Hello {
		return
	}
	if hello.Token != token {
		c.Write(proto.Response{Err: "Bad token. Check cannibalcasa.ini against the server's -token."})
		log.Printf("%s: bad token", nc.RemoteAddr())
		return
	}
	if hello.Version != proto.Version {
		c.Write(proto.Response{Err: fmt.Sprintf("Protocol mismatch: door speaks v%d, server v%d. Update the door.", hello.Version, proto.Version)})
		return
	}
	c.Write(proto.Response{OK: true, Server: version})
	bbs, handle := strings.TrimSpace(hello.BBS), strings.TrimSpace(hello.Handle)
	log.Printf("%s: %s@%s connected", nc.RemoteAddr(), handle, bbs)
	for {
		nc.SetDeadline(time.Now().Add(30 * time.Minute)) // idle door sessions eventually free the slot
		req, err := c.ReadRequest()
		if err != nil {
			return
		}
		var resp proto.Response
		switch req.Type {
		case proto.Join:
			st, ev, err := w.Join(bbs, handle)
			resp = wrap(err, st, nil, ev, nil)
		case proto.Do:
			st, r, err := w.Do(bbs, handle, req.Action)
			resp = wrap(err, st, &r, nil, nil)
		case proto.Hood:
			h, err := w.Neighborhood(bbs, handle)
			resp = wrap(err, nil, nil, nil, &h)
		case proto.Casserole:
			st, r, err := w.Casserole(bbs, handle, req.To, req.ID)
			resp = wrap(err, st, &r, nil, nil)
		case proto.Tip:
			st, r, err := w.Tip(bbs, handle, req.To)
			resp = wrap(err, st, &r, nil, nil)
		case proto.Bye:
			return
		default:
			resp = proto.Response{Err: "unknown request " + req.Type}
		}
		if err := c.Write(resp); err != nil {
			return
		}
	}
}

func wrap(err error, st *game.State, r *game.Result, ev []game.Event, h *world.Neighborhood) proto.Response {
	if err != nil {
		return proto.Response{Err: err.Error()}
	}
	return proto.Response{OK: true, State: st, Result: r, Events: ev, Hood: h}
}
