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

import "sync"

// Broadcaster forwards all messages to all supplied handlers.
// Broadcaster implements the Handler interface.
type Broadcaster struct {
	m sync.RWMutex
	l []Handler
}

// Broadcast forwards all messages sent to Broadcast to all supplied handlers.
// Additional handlers can be added with Listen.
func Broadcast(handlers ...Handler) *Broadcaster {
	b := &Broadcaster{}
	for _, h := range handlers {
		b.Listen(h)
	}
	return b
}

// Listen calls adds h to the list of handlers that are informed of each log
// message passed to Handle.
func (b *Broadcaster) Listen(h Handler) (unlisten func()) {
	b.m.Lock()
	defer b.m.Unlock()
	b.l = append(b.l, h)
	return func() { b.remove(h) }
}

// Handle broadcasts the page to all the listening handlers.
func (b *Broadcaster) Handle(m *Message) {
	b.m.RLock()
	defer b.m.RUnlock()
	for _, h := range b.l {
		h.Handle(m)
	}
}

// Count returns the number of registered handlers.
func (b *Broadcaster) Count() int {
	b.m.RLock()
	defer b.m.RUnlock()
	return len(b.l)
}

// Close calls Close on all the listening handlers and removes them from the
// broadcaster.
func (b *Broadcaster) Close() {
	b.m.Lock()
	defer b.m.Unlock()
	for _, h := range b.l {
		h.Close()
	}
	b.l = []Handler{}
}

func (b *Broadcaster) remove(h Handler) {
	b.m.Lock()
	defer b.m.Unlock()
	for i, t := range b.l {
		if h == t {
			copy(b.l[i:], b.l[i+1:])
			b.l = b.l[:len(b.l)-1]
			return
		}
	}
}
