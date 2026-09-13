// Backends: the door plays against a world.Backend. With a [server] configured
// it dials the neighborhood server and speaks proto over one TCP connection;
// otherwise (or if the server is unreachable) it runs the same rules
// in-process against the data directory, which every node on this BBS shares.
package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/proto"
	"cannibalcasa/internal/world"
)

type remote struct {
	nc     net.Conn
	c      *proto.Conn
	mu     sync.Mutex
	bbs    string
	handle string
}

func dialRemote(cfg config, bbs, handle string) (*remote, error) {
	nc, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.Server, cfg.Port), 3*time.Second)
	if err != nil {
		return nil, err
	}
	r := &remote{nc: nc, c: proto.NewConn(nc), bbs: bbs, handle: handle}
	nc.SetDeadline(time.Now().Add(5 * time.Second))
	if err := r.c.Write(proto.Request{Type: proto.Hello, Version: proto.Version, Token: cfg.Token, BBS: bbs, Handle: handle}); err != nil {
		nc.Close()
		return nil, err
	}
	resp, err := r.c.ReadResponse()
	if err != nil {
		nc.Close()
		return nil, err
	}
	if !resp.OK {
		nc.Close()
		return nil, errors.New(resp.Err)
	}
	logf("connected to neighborhood server %s:%s (server %s)", cfg.Server, cfg.Port, resp.Server)
	return r, nil
}

func (r *remote) call(req proto.Request) (proto.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nc.SetDeadline(time.Now().Add(15 * time.Second))
	if err := r.c.Write(req); err != nil {
		return proto.Response{}, err
	}
	resp, err := r.c.ReadResponse()
	if err != nil {
		return proto.Response{}, err
	}
	if !resp.OK {
		return resp, errors.New(resp.Err)
	}
	return resp, nil
}

func (r *remote) Join(bbs, handle string) (*game.State, []game.Event, error) {
	resp, err := r.call(proto.Request{Type: proto.Join})
	if err != nil {
		return nil, nil, err
	}
	return resp.State, resp.Events, nil
}

func (r *remote) Do(bbs, handle string, a game.Action) (*game.State, game.Result, error) {
	resp, err := r.call(proto.Request{Type: proto.Do, Action: a})
	if err != nil {
		return nil, game.Result{}, err
	}
	return resp.State, deref(resp.Result), nil
}

func (r *remote) Neighborhood(bbs, handle string) (world.Neighborhood, error) {
	resp, err := r.call(proto.Request{Type: proto.Hood})
	if err != nil || resp.Hood == nil {
		return world.Neighborhood{}, err
	}
	return *resp.Hood, nil
}

func (r *remote) Casserole(bbs, handle, toKey string, dishID int) (*game.State, game.Result, error) {
	resp, err := r.call(proto.Request{Type: proto.Casserole, To: toKey, ID: dishID})
	if err != nil {
		return nil, game.Result{}, err
	}
	return resp.State, deref(resp.Result), nil
}

func (r *remote) Tip(bbs, handle, toKey string) (*game.State, game.Result, error) {
	resp, err := r.call(proto.Request{Type: proto.Tip, To: toKey})
	if err != nil {
		return nil, game.Result{}, err
	}
	return resp.State, deref(resp.Result), nil
}

func (r *remote) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.c.Write(proto.Request{Type: proto.Bye})
	return r.nc.Close()
}

func deref(r *game.Result) game.Result {
	if r == nil {
		return game.Result{}
	}
	return *r
}

// pickBackend prefers the configured server and falls back to local play,
// reporting which one it chose so the player knows whose street they're on.
func pickBackend(cfg config, bbs, handle string, forceLocal bool) (world.Backend, string) {
	if cfg.Server != "" && !forceLocal {
		if r, err := dialRemote(cfg, bbs, handle); err == nil {
			return r, "Connected to the shared neighborhood at " + cfg.Server + "."
		} else {
			logf("server %s:%s unreachable: %v; falling back to local", cfg.Server, cfg.Port, err)
			return world.NewLocal(cfg.DataDir, cfg.Rules), "Neighborhood server unreachable (" + err.Error() + "). Playing the local street."
		}
	}
	return world.NewLocal(cfg.DataDir, cfg.Rules), "Local neighborhood: everyone on this BBS shares the street."
}
