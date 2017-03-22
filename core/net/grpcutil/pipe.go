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

package grpcutil

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
)

const (
	ErrListenerClosed = fault.Const("pipeListener closed")
)

var pchans = func() func(string) chan net.Conn {
	m := make(map[string]chan net.Conn)
	var l sync.Mutex
	return func(addr string) chan net.Conn {
		l.Lock()
		defer l.Unlock()
		if ch, found := m[addr]; found {
			return ch
		} else {
			m[addr] = make(chan net.Conn)
			return m[addr]
		}
	}
}()

type pipeListener struct {
	ch     chan net.Conn
	l      sync.Mutex
	closed bool
	done   chan struct{}
	addr   pipeAddr
}

func (f *pipeListener) Accept() (net.Conn, error) {
	select {
	case <-f.done:
		return nil, ErrListenerClosed
	default:
	}
	left, right := net.Pipe()
	select {
	case f.ch <- left:
		return right, nil
	case <-f.done:
		return nil, ErrListenerClosed
	}
}

func (f *pipeListener) Close() error {
	f.l.Lock()
	defer f.l.Unlock()
	if !f.closed {
		close(f.done)
		f.closed = true
	}
	return nil
}

func (f *pipeListener) Addr() net.Addr {
	return f.addr
}

// DialPipe connects to listeners obtained with NewPipeListener.
func DialPipe(addr string) (net.Conn, error) {
	return <-pchans(addr), nil
}

// DialPipeCtx connects to listeners obtained with NewPipeListener, and
// errors if the context is canceled.
func DialPipeCtx(ctx context.Context, addr string) (net.Conn, error) {
	select {
	case <-task.ShouldStop(ctx):
		return nil, errors.New("context cancelled")
	case conn := <-pchans(addr):
		return conn, nil
	}
}

// GetDialer takes a context and returns a grpc-compatible dialer function
// that will fail if the context is stopped or if the specified timeout
// passes.
func GetDialer(ctx context.Context) func(string, time.Duration) (net.Conn, error) {
	return func(addr string, timeout time.Duration) (net.Conn, error) {
		dialerCtx := ctx
		if timeout != 0 {
			dialerCtx, _ = task.WithTimeout(ctx, timeout)
		}
		return DialPipeCtx(dialerCtx, addr)
	}
}

// NewPipeListener returns a net.Listener that accepts connections from DialPipe.
// This listener will use net.Pipe() to create a pair of endpoints and match
// them together. addr is used to distinguish between multiple listeners.
// Multiple listeners can accept connections on the same address.
func NewPipeListener(addr string) net.Listener {
	return &pipeListener{
		done: make(chan struct{}),
		addr: pipeAddr{addr},
		ch:   pchans(addr),
	}
}

type pipeAddr struct {
	Name string
}

func (pipeAddr) Network() string {
	return "pipe"
}
func (f pipeAddr) String() string {
	return f.Name
}
