// Backends: the door plays against a world.Backend. With a [server] configured
// it dials the neighborhood server and speaks proto over one TCP connection;
// otherwise (or if the server is unreachable) it runs the same simulation
// in-process against the data directory every node on this BBS shares.
package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"cannibalcasa/internal/proto"
	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/world"
)

type remote struct {
	nc net.Conn
	c  *proto.Conn
	mu sync.Mutex
}

func dialRemote(cfg config, bbs, handle string) (*remote, error) {
	nc, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.Server, cfg.Port), 3*time.Second)
	if err != nil {
		return nil, err
	}
	r := &remote{nc: nc, c: proto.NewConn(nc)}
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
	r.nc.SetDeadline(time.Now().Add(20 * time.Second))
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

func (r *remote) Join(bbs, handle string) (*sim.World, []sim.Event, int, error) {
	resp, err := r.call(proto.Request{Type: proto.Join})
	if err != nil {
		return nil, nil, 0, err
	}
	return resp.World, resp.Events, resp.Ticks, nil
}

func (r *remote) Sync(bbs, handle string, speed int) (*sim.World, error) {
	resp, err := r.call(proto.Request{Type: proto.Sync, Speed: speed})
	if err != nil {
		return nil, err
	}
	return resp.World, nil
}

func (r *remote) Do(bbs, handle string, c sim.Command) (*sim.World, error) {
	resp, err := r.call(proto.Request{Type: proto.Do, Command: c})
	return resp.World, err
}

func (r *remote) Neighborhood(bbs, handle string) (world.Neighborhood, error) {
	resp, err := r.call(proto.Request{Type: proto.Hood})
	if err != nil || resp.Hood == nil {
		return world.Neighborhood{}, err
	}
	return *resp.Hood, nil
}

func (r *remote) Casserole(bbs, handle, toKey string, dishID int) (*sim.World, error) {
	resp, err := r.call(proto.Request{Type: proto.Casserole, To: toKey, ID: dishID})
	return resp.World, err
}

func (r *remote) Tip(bbs, handle, toKey string) (*sim.World, error) {
	resp, err := r.call(proto.Request{Type: proto.Tip, To: toKey})
	return resp.World, err
}

func (r *remote) Leave(bbs, handle string) error { return nil }

func (r *remote) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.c.Write(proto.Request{Type: proto.Bye})
	return r.nc.Close()
}

// pickBackend prefers the configured server and falls back to local play,
// reporting which one it chose so the player knows whose street they're on.
func pickBackend(cfg config, bbs, handle string, forceLocal bool) (world.Backend, string) {
	if cfg.Server != "" && !forceLocal {
		if r, err := dialRemote(cfg, bbs, handle); err == nil {
			return r, "Connected to the shared neighborhood at " + cfg.Server + "."
		} else {
			logf("server %s:%s unreachable: %v; falling back to local", cfg.Server, cfg.Port, err)
			return world.NewLocal(cfg.DataDir, cfg.Scale), "Neighborhood server unreachable (" + err.Error() + "). Playing the local street."
		}
	}
	return world.NewLocal(cfg.DataDir, cfg.Scale), "Local neighborhood: everyone on this BBS shares the street."
}
