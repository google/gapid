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

package log

import "github.com/google/gapid/core/app/crash"

// Channel is a log handler that passes log messages to another Handler through
// a chan.
// This makes this Handler safe to use from multiple threads.
func Channel(to Handler, size int) Handler {
	c := make(chan *Message, size)
	done := make(chan struct{})
	crash.Go(func() {
		defer func() {
			to.Close()
			close(done)
		}()
		for m := range c {
			if m == nil {
				return
			}
			to.Handle(m)
		}
	})
	handle := func(m *Message) {
		if m == nil {
			return
		}
		select {
		case c <- m: // Message sent.
		case <-done: // Handler closed. Message dropped on floor.
		}
	}
	close := func() {
		select {
		case <-done: // Already stopped.
		case c <- nil: // Stop requested.
			<-done // Wait for flush of existing messages.
		}
	}
	return &handler{handle, close}
}
