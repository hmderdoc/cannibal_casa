// Package proto is the JSON-lines wire protocol between the door and the
// neighborhood server: one JSON object per line, request then response, over
// a persistent TCP connection that begins with a token-gated hello. The wire
// carries state, never pixels; the door renders.
package proto

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"

	"cannibalcasa/internal/game"
	"cannibalcasa/internal/world"
)

// Version bumps when the wire shape changes incompatibly.
const Version = 1

// Request types.
const (
	Hello     = "hello"
	Join      = "join"
	Do        = "do"
	Hood      = "hood"
	Casserole = "casserole"
	Tip       = "tip"
	Bye       = "bye"
)

type Request struct {
	Type    string      `json:"type"`
	Version int         `json:"version,omitempty"`
	Token   string      `json:"token,omitempty"`
	BBS     string      `json:"bbs,omitempty"`
	Handle  string      `json:"handle,omitempty"`
	Action  game.Action `json:"action,omitempty"`
	To      string      `json:"to,omitempty"`
	ID      int         `json:"id,omitempty"`
}

type Response struct {
	OK     bool                `json:"ok"`
	Err    string              `json:"err,omitempty"`
	State  *game.State         `json:"state,omitempty"`
	Result *game.Result        `json:"result,omitempty"`
	Events []game.Event        `json:"events,omitempty"`
	Hood   *world.Neighborhood `json:"hood,omitempty"`
	Server string              `json:"server,omitempty"` // server version, in the hello reply
}

const maxLine = 4 * 1024 * 1024

// Conn frames JSON lines over any stream.
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
