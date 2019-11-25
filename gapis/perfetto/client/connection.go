// Copyright (C) 2019 Google Inc.
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

package client

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"

	wire "protos/perfetto/wire"
)

// Connection is a connection to the Perfetto service.
type Connection struct {
	conn    net.Conn
	out     binary.Writer
	in      binary.Reader
	reqID   uint64
	cleanup app.Cleanup

	closed   chan struct{}
	packets  chan packet
	lock     sync.Mutex
	handlers map[uint64]handler
}

// Method represents an RPC method that can be called on the Perfetto service.
type Method struct {
	Name      string
	serviceID uint32
	methodID  uint32
}

type packet struct {
	Frame *wire.IPCFrame
	Err   error
}

type handler func(frame *wire.IPCFrame, err error) bool

// Connect creates a new connection that uses the established socket to
// communicate with the Perfetto service.
func Connect(ctx context.Context, conn net.Conn, cleanup app.Cleanup) (*Connection, error) {
	c := &Connection{
		conn:    conn,
		out:     endian.Writer(conn, device.LittleEndian),
		in:      endian.Reader(conn, device.LittleEndian),
		cleanup: cleanup,

		closed:   make(chan struct{}),
		packets:  make(chan packet, 1),
		handlers: map[uint64]handler{},
	}

	// Reads frames from the wire and pushes them into the channel. Will exit on
	// error (including if the connection is closed).
	crash.Go(func() {
		for {
			frame, err := c.readFrame(ctx)
			select {
			case c.packets <- packet{frame, err}:
				if err != nil {
					close(c.packets)
					return
				}
			case <-c.closed:
				close(c.packets)
				return
			}
		}
	})

	// Collects frames from the channel and passes them onto the registered
	// handlers. Drops unknown frames on the floor.
	crash.Go(func() {
		for {
			select {
			case pkt := <-c.packets:
				if pkt.Err != nil {
					// Broadcast the error to all the registered handlers.
					c.lock.Lock()
					handlers := c.handlers
					c.handlers = nil
					c.lock.Unlock()

					for _, h := range handlers {
						h(nil, pkt.Err)
					}
					return
				}

				c.lock.Lock()
				h, ok := c.handlers[pkt.Frame.GetRequestId()]
				c.lock.Unlock()

				if ok {
					if !h(pkt.Frame, nil) {
						c.unregisterHandler(pkt.Frame.GetRequestId())
					}
				} else {
					log.W(ctx, "Got a Perfetto packet with an unkown request ID: %v", pkt.Frame)
				}
			case <-c.closed:
				return
			}
		}
	})

	return c, nil
}

// BindHandler is the callback invoked when the Bind completes.
type BindHandler func(methods map[string]*Method, err error)

// Bind binds to the given proto service.
func (c *Connection) Bind(ctx context.Context, service string, handler BindHandler) error {
	f := c.newFrame()
	f.Msg = &wire.IPCFrame_MsgBindService{
		MsgBindService: &wire.IPCFrame_BindService{
			ServiceName: proto.String(service),
		},
	}
	return c.writeFrame(ctx, f, func(frame *wire.IPCFrame, err error) bool {
		if err != nil {
			handler(nil, err)
			return false
		}

		if !frame.GetMsgBindServiceReply().GetSuccess() {
			handler(nil, fmt.Errorf("Failed to bind to the %s service", service))
			return false
		}

		serviceID := frame.GetMsgBindServiceReply().GetServiceId()
		methods := map[string]*Method{}
		for _, m := range frame.GetMsgBindServiceReply().GetMethods() {
			methods[m.GetName()] = &Method{m.GetName(), serviceID, m.GetId()}
		}
		handler(methods, nil)
		return false
	})
}

// InvokeHandler is the callback invoked when an Invoke gets a response. For
// streaming RPCs it may be invoked multiple times.
type InvokeHandler func(data []byte, more bool, err error)

// Invoke invokes the given method and calls the handler on a response.
func (c *Connection) Invoke(ctx context.Context, m *Method, args proto.Message, handler InvokeHandler) error {
	argsBuf, err := proto.Marshal(args)
	if err != nil {
		return err
	}

	f := c.newFrame()
	f.Msg = &wire.IPCFrame_MsgInvokeMethod{
		MsgInvokeMethod: &wire.IPCFrame_InvokeMethod{
			ServiceId: proto.Uint32(m.serviceID),
			MethodId:  proto.Uint32(m.methodID),
			ArgsProto: argsBuf,
		},
	}
	return c.writeFrame(ctx, f, func(frame *wire.IPCFrame, err error) bool {
		if err != nil {
			handler(nil, false, err)
			return false
		}

		reply := frame.GetMsgInvokeMethodReply()
		if !reply.GetSuccess() {
			handler(nil, false, fmt.Errorf("Failed to invoke consumer method %s", m.Name))
			return false
		}

		more := reply.GetHasMore()
		handler(reply.GetReplyProto(), more, nil)
		return more
	})
}

// Close closes this connection and the underlying socket.
func (c *Connection) Close(ctx context.Context) {
	if c != nil {
		close(c.closed)
		c.conn.Close()
		c.cleanup.Invoke(ctx)
	}
}

func (c *Connection) registerHandler(reqID uint64, handler handler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.handlers[reqID] = handler
}

func (c *Connection) unregisterHandler(reqID uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.handlers, reqID)
}

func (c *Connection) newFrame() *wire.IPCFrame {
	reqID := atomic.AddUint64(&c.reqID, 1)
	return &wire.IPCFrame{
		RequestId: proto.Uint64(reqID - 1),
	}
}

func (c *Connection) writeFrame(ctx context.Context, frame *wire.IPCFrame, handler handler) error {
	buf, err := proto.Marshal(frame)
	if err != nil {
		return err
	}

	reqID := frame.GetRequestId()
	c.lock.Lock()
	defer c.lock.Unlock()

	c.handlers[reqID] = handler
	c.out.Uint32(uint32(len(buf)))
	c.out.Data(buf)

	if err := c.out.Error(); err != nil {
		delete(c.handlers, reqID)
		return err
	}
	return nil
}

func (c *Connection) readFrame(ctx context.Context) (*wire.IPCFrame, error) {
	size := c.in.Uint32()
	buf := make([]byte, size)
	c.in.Data(buf)
	if err := c.in.Error(); err != nil {
		return nil, err
	}

	frame := &wire.IPCFrame{}
	if err := proto.Unmarshal(buf, frame); err != nil {
		return nil, err
	}

	switch m := frame.Msg.(type) {
	case *wire.IPCFrame_MsgRequestError:
		return nil, fmt.Errorf("Request Error: %s", m.MsgRequestError.GetError())
	}
	return frame, nil
}
