// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package jdwp implements types to communicate using the the Java Debug Wire Protocol.
package jdwp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
)

var (
	handshake = []byte("JDWP-Handshake")

	defaultIDSizes = IDSizes{
		FieldIDSize:         8,
		MethodIDSize:        8,
		ObjectIDSize:        8,
		ReferenceTypeIDSize: 8,
		FrameIDSize:         8,
	}
)

type eventsID uint64

// Connection represents a JDWP connection.
type Connection struct {
	in           io.Reader
	r            binary.Reader
	w            binary.Writer
	flush        func() error
	idSizes      IDSizes
	nextPacketID packetID
	nextEventsID eventsID
	events       map[eventsID]chan<- Events
	replies      map[packetID]chan<- replyPacket
	sync.Mutex
}

// Open creates a Connection using conn for I/O.
func Open(ctx context.Context, conn io.ReadWriteCloser) (*Connection, error) {
	if err := exchangeHandshakes(conn); err != nil {
		return nil, err
	}

	buf := bufio.NewWriterSize(conn, 1024)
	r := endian.Reader(conn, device.BigEndian)
	w := endian.Writer(buf, device.BigEndian)
	events := map[eventsID]chan<- Events{}
	replies := map[packetID]chan<- replyPacket{}
	c := &Connection{
		in:      conn,
		r:       r,
		w:       w,
		flush:   buf.Flush,
		idSizes: defaultIDSizes,
		events:  events,
		replies: replies,
	}

	crash.Go(func() { c.recv(ctx) })
	var err error
	c.idSizes, err = c.GetIDSizes()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func exchangeHandshakes(conn io.ReadWriter) error {
	if _, err := conn.Write(handshake); err != nil {
		return err
	}
	ok, err := expect(conn, handshake)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Bad handshake")
	}
	return nil
}

// expect reads c.in, expecting the specfified sequence of bytes. If the read
// data doesn't match, then the function returns immediately with false.
func expect(conn io.Reader, expected []byte) (bool, error) {
	got := make([]byte, len(expected))
	for len(expected) > 0 {
		n, err := conn.Read(got)
		if err != nil {
			return false, err
		}
		for i := 0; i < n; i++ {
			if got[i] != expected[i] {
				return false, nil
			}
		}
		got, expected = got[n:], expected[n:]
	}
	return true, nil
}

// get sends the specified command and waits for a reply.
func (c *Connection) get(cmdSet cmdSet, cmd cmdID, req interface{}, out interface{}) error {
	data := bytes.Buffer{}
	if req != nil {
		e := endian.Writer(&data, device.BigEndian)
		if err := c.encode(e, reflect.ValueOf(req)); err != nil {
			return err
		}
	}

	id, replyChan := c.newReplyHandler()
	defer c.deleteReplyHandler(id)

	p := cmdPacket{id: id, cmdSet: cmdSet, cmdID: cmd, data: data.Bytes()}
	if err := p.write(c.w); err != nil {
		return err
	}
	if err := c.flush(); err != nil {
		return err
	}

	select {
	case reply := <-replyChan:
		if reply.err != ErrNone {
			return reply.err
		}
		if out == nil {
			return nil
		}
		r := bytes.NewReader(reply.data)
		d := endian.Reader(r, device.BigEndian)
		if err := c.decode(d, reflect.ValueOf(out)); err != nil {
			return err
		}
		if offset, _ := r.Seek(0, 1); offset != int64(len(reply.data)) {
			panic(fmt.Errorf("Only %d/%d bytes read from reply packet", offset, len(reply.data)))
		}
		return nil
	case <-time.After(time.Second * 120):
		return fmt.Errorf("timeout")
	}
}

func (c *Connection) newReplyHandler() (packetID, <-chan replyPacket) {
	reply := make(chan replyPacket, 1)
	c.Lock()
	id := c.nextPacketID
	c.nextPacketID++
	c.replies[id] = reply
	c.Unlock()
	return id, reply
}

func (c *Connection) deleteReplyHandler(id packetID) {
	c.Lock()
	delete(c.replies, id)
	c.Unlock()
}
