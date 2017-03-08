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

package task

import (
	"context"
	"sync"
	"time"
)

// Event is the interface to things that can be used to wait for conditions to occur.
type Event interface {
	// Fired returns true if Wait would not block.
	Fired() bool
	// Wait blocks until the signal has been fired or the context has been
	// cancelled.
	// Returns true if the signal was fired, false if the context was cancelled.
	Wait(ctx context.Context) bool
	// TryWait waits for the signal to fire, the context to be cancelled or the
	// timeout, whichever comes first.
	TryWait(ctx context.Context, timeout time.Duration) bool
}

// Events is a thread safe list of Event entries.
// It can be used to collect a list of events so you can Wait for them all to complete.
// The list of events may be purged of already completed entries at any time, this is an invisible
// optimization to the semantics of the API.
type Events struct {
	mutex   sync.Mutex
	pending []Event
}

// Add new events to the set to wait for completion on.
func (e *Events) Add(events ...Event) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pending = append(e.pending, events...)
}

// purge the wait list of already completed ones.
// This must only be called while the list mutex is locked.
func (e *Events) purge() {
	next := 0
	for _, event := range e.pending {
		if !event.Fired() {
			e.pending[next] = event
			next++
		}
	}
	e.pending = e.pending[:next]
}

// Pending returns the count of still pending events.
func (e *Events) Pending() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.purge()
	return len(e.pending)
}

// Join the current list of events into a signal you can wait on.
// Subsequent calls to Add will not affect the returned signal.
func (e *Events) Join(ctx context.Context) Signal {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.purge()
	// Make a copy of the pending list now, so later mutations don't affect us
	joined := append([]Event(nil), e.pending...)
	// build the signal we are going to return
	done, fire := NewSignal()
	go func() {
		defer fire(ctx)
		for _, signal := range joined {
			signal.Wait(ctx)
		}
	}()
	return done
}

// Wait blocks until the all events in the list have been fired.
// This is a helper for e.Join(ctx).Wait(ctx)
func (e *Events) Wait(ctx context.Context) bool {
	return e.Join(ctx).Wait(ctx)
}

// TryWait waits for either timeout or all the events to fire, whichever comes first.
// This is a helper for e.Join(ctx).Wait(timeout)
func (e *Events) TryWait(ctx context.Context, timeout time.Duration) bool {
	return e.Join(ctx).TryWait(ctx, timeout)
}
