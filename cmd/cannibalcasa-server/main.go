// cannibalcasa-server: the neighborhood server. It owns the cul-de-sac (every
// player's world, the feed, the mail) and runs the simulation; doors on any
// BBS connect out with a shared token, sync the clock, and send commands.
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

	"cannibalcasa/internal/proto"
	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/world"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":7710", "listen address")
	token := flag.String("token", "", "shared secret every door must present (required)")
	dataDir := flag.String("data", "", "neighborhood data directory (default: SBBSDATA/cannibalcasa or ./data)")
	scale := flag.Int("time-scale", sim.DefaultScale, "game minutes per real minute")
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
	w := world.NewLocal(dir, *scale)
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	log.Printf("cannibalcasa-server %s listening on %s, data in %s, scale %d", version, *addr, dir, *scale)
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
	defer func() {
		w.Leave(bbs, handle)
		log.Printf("%s: %s@%s left", nc.RemoteAddr(), handle, bbs)
	}()
	for {
		nc.SetDeadline(time.Now().Add(10 * time.Minute))
		req, err := c.ReadRequest()
		if err != nil {
			return
		}
		var resp proto.Response
		switch req.Type {
		case proto.Join:
			st, ev, ticks, err := w.Join(bbs, handle)
			resp = wrap(err, st)
			resp.Events, resp.Ticks = ev, ticks
		case proto.Sync:
			st, err := w.Sync(bbs, handle, req.Speed)
			resp = wrap(err, st)
		case proto.Do:
			st, err := w.Do(bbs, handle, req.Command)
			resp = wrap(err, st)
		case proto.Hood:
			h, err := w.Neighborhood(bbs, handle)
			resp = wrap(err, nil)
			if err == nil {
				resp.Hood = &h
			}
		case proto.Casserole:
			st, err := w.Casserole(bbs, handle, req.To, req.ID)
			resp = wrap(err, st)
		case proto.Tip:
			st, err := w.Tip(bbs, handle, req.To)
			resp = wrap(err, st)
		case proto.Raid:
			st, err := w.Raid(bbs, handle, req.To)
			resp = wrap(err, st)
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

// wrap turns a backend result into a response. A command error still carries
// the world so the door can show the message without losing sync.
func wrap(err error, st *sim.World) proto.Response {
	if err != nil {
		return proto.Response{Err: err.Error(), World: st}
	}
	return proto.Response{OK: true, World: st}
}
