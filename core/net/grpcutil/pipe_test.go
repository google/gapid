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

package grpcutil_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"net"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/net/grpcutil"
)

func TestPipeListener(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	data := []byte(strings.Repeat("TestPipeListener data", 100))
	listener := func(addr string) {
		l := grpcutil.NewPipeListener(addr)
		for i := 0; i < 2; i++ {
			s, err := l.Accept()
			assert.For("accept").ThatError(err).Succeeded()
			s.Write(data[0:2])
			s.Write(data[2:])
			s.Close()
		}
		l.Close()
	}

	check := func(c net.Conn, err error) {
		assert.For("dial").ThatError(err).Succeeded()
		buf := make([]byte, len(data))
		n, err := io.ReadFull(c, buf)
		assert.For("read full").ThatError(err).Succeeded()
		assert.For("all data read").That(n).Equals(len(buf))
		assert.For("data matches").ThatSlice(buf).Equals(data)
		c.Close()
	}

	go listener("pipe:1")
	go listener("pipe:2")
	go listener("pipe:2")
	for i := 0; i < 2; i++ {
		check(grpcutil.GetDialer(ctx)("pipe:1", 0))
		check(grpcutil.DialPipe("pipe:2"))
		check(grpcutil.DialPipeCtx(ctx, "pipe:2"))
	}
}

func TestPipeListener_Close(t *testing.T) {
	assert := assert.To(t)
	l := grpcutil.NewPipeListener("pipe:3")

	// Close while an Accept is being executed.
	sig := make(chan struct{}, 1)
	go func() {
		time.Sleep(1 * time.Millisecond)
		l.Close()
		sig <- struct{}{}
	}()
	_, err := l.Accept()
	assert.For("accept").ThatError(err).Failed()
	<-sig

	// At this point, the listener is already closed.
	// The Accept should just fail, even if there is a
	// client waiting.
	sig = make(chan struct{}, 1)
	go func() {
		_, err := l.Accept()
		assert.For("accept").ThatError(err).Failed()
		// Unblock client.
		grpcutil.NewPipeListener("pipe:3").Accept()
		sig <- struct{}{}
	}()
	c, err := grpcutil.DialPipe("pipe:3")
	assert.For("dial").ThatError(err).Succeeded()
	c.Close()
	<-sig
}

func TestDialPipeCtxTimeout(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	ctxt, _ := task.WithTimeout(ctx, 1*time.Nanosecond)
	_, err := grpcutil.DialPipeCtx(ctxt, "pipe:5")
	assert.For("dial").ThatError(err).Failed()
}

func TestDialPipeCtxCancel(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	ctxt, cancel := task.WithCancel(ctx)
	go func() {
		cancel()
	}()
	_, err := grpcutil.DialPipeCtx(ctxt, "pipe:6")
	assert.For("dial").ThatError(err).Failed()
}
