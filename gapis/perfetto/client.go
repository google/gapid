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

package perfetto

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/gapis/perfetto/client"

	common "protos/perfetto/common"
	config "protos/perfetto/config"
	ipc "protos/perfetto/ipc"
)

const (
	consumerService = "ConsumerPort"
	queryMethod     = "QueryServiceState"
	traceMethod     = "EnableTracing"
	startMethod     = "StartTracing"
	stopMethod      = "DisableTracing"
	readMethod      = "ReadBuffers"
)

// Client is a client ("consumer") of a Perfetto service.
type Client struct {
	conn    *client.Connection
	methods map[string]*client.Method
}

// NewClient returns a new client using the provided socket connection. The
// client takes ownership of the connection and invokes the provided cleanup
// on shutdown.
func NewClient(ctx context.Context, conn net.Conn, cleanup app.Cleanup) (*Client, error) {
	c, err := client.Connect(ctx, conn, cleanup)
	if err != nil {
		conn.Close()
		cleanup.Invoke(ctx)
		return nil, err
	}

	bind := client.NewBindSync(ctx)
	if err := c.Bind(ctx, consumerService, bind.Handler); err != nil {
		c.Close(ctx)
		return nil, err
	}
	methods, err := bind.Wait(ctx)
	if err != nil {
		c.Close(ctx)
		return nil, err
	}

	return &Client{
		conn:    c,
		methods: methods,
	}, nil
}

// Query queries the Perfetto service for producer and data source info and
// invokes the given callback on each received result. This is a streaming,
// synchronous RPC and the callback may be invoked multiple times.
func (c *Client) Query(ctx context.Context, cb func(*common.TracingServiceState) error) error {
	m, ok := c.methods[queryMethod]
	if !ok {
		return errors.New("Remote service doesn't have a query method")
	}

	query := client.NewQuerySync(ctx, cb)
	if err := c.conn.Invoke(ctx, m, &ipc.QueryServiceStateRequest{}, query.Handler); err != nil {
		return err
	}
	return query.Wait(ctx)
}

// TraceSession is the interface to a currently running Perfetto trace.
type TraceSession struct {
	conn  *client.Connection
	wait  task.Signal
	done  task.Task
	start *client.Method
	stop  *client.Method
	err   error
}

// Trace initiates a new Perfetto trace session using the given config. The
// trace buffers will be serialized to the given writer. This is an asynchronous
// RPC that can be controlled/waited on using the returned trace session.
func (c *Client) Trace(ctx context.Context, cfg *config.TraceConfig, out io.Writer) (*TraceSession, error) {
	trace, okTrace := c.methods[traceMethod]
	start, _ := c.methods[startMethod] // ignore missing start
	stop, okStop := c.methods[stopMethod]
	read, okRead := c.methods[readMethod]
	if !okTrace || !okStop || !okRead {
		return nil, errors.New("Remote service doesn't have the trace methods")
	}

	if !cfg.GetDeferredStart() {
		start = nil
	}

	wait, done := task.NewSignal()
	s := &TraceSession{
		conn:  c.conn,
		wait:  wait,
		done:  done,
		start: start,
		stop:  stop,
	}

	pw := client.NewPacketWriter(out)

	h := client.NewTraceHandler(ctx, func(r *ipc.EnableTracingResponse, err error) {
		if !s.onResult(ctx, err) {
			c.readBuffers(ctx, read, s, pw)
		}
	})

	if err := c.conn.Invoke(ctx, trace, &ipc.EnableTracingRequest{TraceConfig: cfg}, h); err != nil {
		return nil, err
	}

	return s, nil
}

func (c *Client) readBuffers(ctx context.Context, m *client.Method, s *TraceSession, out *client.PacketWriter) {
	h := client.NewReadHandler(ctx, func(r *ipc.ReadBuffersResponse, more bool, err error) {
		if s.onResult(ctx, err) {
			return
		}

		if s.onResult(ctx, out.Write(r.Slices)) {
			return
		}

		if !more {
			s.done(ctx)
		}
	})
	s.onResult(ctx, c.conn.Invoke(ctx, m, &ipc.ReadBuffersRequest{}, h))
}

// Start starts the currently deferred trace of this session. Does nothing, if
// the Perfetto service doesn't support deferred tracing or the trace was not
// started in deferred mode.
func (s *TraceSession) Start(ctx context.Context) {
	if s.start == nil || s.wait.Fired() {
		// Ignore any starts if the trace was not defferred or after we've
		// already marked this session as done
		return
	}

	h := client.NewIgnoreHandler(ctx, func(err error) {
		s.onResult(ctx, err)
	})
	s.onResult(ctx, s.conn.Invoke(ctx, s.start, &ipc.StartTracingRequest{}, h))
}

// Stop stops the currently running trace of this session.
func (s *TraceSession) Stop(ctx context.Context) {
	if s.wait.Fired() {
		// Ignore any stops after we've already marked this session as done.
		return
	}

	h := client.NewIgnoreHandler(ctx, func(err error) {
		s.onResult(ctx, err)
	})
	s.onResult(ctx, s.conn.Invoke(ctx, s.stop, &ipc.DisableTracingRequest{}, h))
}

// Wait waits for this trace session to finish and returns any error encountered
// during the trace.
func (s *TraceSession) Wait(ctx context.Context) error {
	if !s.wait.Wait(ctx) {
		return task.StopReason(ctx)
	}
	return s.err
}

func (s *TraceSession) onResult(ctx context.Context, err error) bool {
	if err != nil {
		if s.err == nil {
			s.err = err
		}
		s.done(ctx)
		return true
	}
	return false
}

// Close closes the underlying connection to the Perfetto service of this client.
func (c *Client) Close(ctx context.Context) {
	c.conn.Close(ctx)
	c.conn = nil
}
