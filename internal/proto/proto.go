// Package proto is the JSON-lines wire protocol between the door and the
// neighborhood server: one JSON object per line, request then response, over
// a persistent TCP connection that begins with a token-gated hello. The wire
// carries the world, never pixels; the door renders.
package proto

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"

	"cannibalcasa/internal/sim"
	"cannibalcasa/internal/world"
)

const Version = 2

const (
	Hello     = "hello"
	Join      = "join"
	Sync      = "sync"
	Do        = "do"
	Hood      = "hood"
	Casserole = "casserole"
	Tip       = "tip"
	Raid      = "raid"
	Bye       = "bye"
)

type Request struct {
	Type    string      `json:"type"`
	Version int         `json:"version,omitempty"`
	Token   string      `json:"token,omitempty"`
	BBS     string      `json:"bbs,omitempty"`
	Handle  string      `json:"handle,omitempty"`
	Command sim.Command `json:"command,omitempty"`
	Speed   int         `json:"speed,omitempty"`
	To      string      `json:"to,omitempty"`
	ID      int         `json:"id,omitempty"`
}

type Response struct {
	OK     bool                `json:"ok"`
	Err    string              `json:"err,omitempty"`
	World  *sim.World          `json:"world,omitempty"`
	Events []sim.Event         `json:"events,omitempty"`
	Ticks  int                 `json:"ticks,omitempty"`
	Hood   *world.Neighborhood `json:"hood,omitempty"`
	Server string              `json:"server,omitempty"`
}

const maxLine = 8 * 1024 * 1024

type Conn struct {
	r *bufio.Reader
	w io.Writer
}

func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{r: bufio.NewReaderSize(rw, 64*1024), w: rw}
}

func (c *Conn) Write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.w.Write(append(data, '\n'))
	return err
}

func (c *Conn) readLine() ([]byte, error) {
	var line []byte
	for {
		part, isPrefix, err := c.r.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, part...)
		if len(line) > maxLine {
			return nil, errors.New("line too long")
		}
		if !isPrefix {
			return line, nil
		}
	}
}

func (c *Conn) ReadRequest() (Request, error) {
	var req Request
	line, err := c.readLine()
	if err != nil {
		return req, err
	}
	return req, json.Unmarshal(line, &req)
}

func (c *Conn) ReadResponse() (Response, error) {
	var resp Response
	line, err := c.readLine()
	if err != nil {
		return resp, err
	}
	return resp, json.Unmarshal(line, &resp)
}
