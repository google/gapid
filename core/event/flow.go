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

package event

import (
	"context"
	"sync"
)

// Filter returns a handler that only forwards events to the underlying handler
// if pred is true for that event.
func Filter(ctx context.Context, pred Predicate, handler Handler) Handler {
	return func(ctx context.Context, event interface{}) error {
		if !pred(ctx, event) {
			return nil
		}
		return handler(ctx, event)
	}
}

// FilterAny returns a handler that only forwards events to the underlying handler
// if pred is true for that event.
// It obeys the same rules as AsHandler and AsPredicate when dealing with pred and dst.
func FilterAny(ctx context.Context, pred interface{}, dst interface{}) Handler {
	return Filter(ctx, AsPredicate(ctx, pred), AsHandler(ctx, dst))
}

// Feed passes events from a producer to the handler.
// It returns only when either src returns nil or dst returns an error.
// It will not pass the nil onto dst.
// It converts a pull interface to a push one.
func Feed(ctx context.Context, dst Handler, src Producer) error {
	for {
		event := src(ctx)
		if event == nil {
			return nil
		}
		if err := dst(ctx, event); err != nil {
			return err
		}
	}
}

// Connect adds a handler to a listener in a way that blocks until the handler is
// done.
func Connect(ctx context.Context, listener Listener, handler Handler) error {
	h, done := Drain(ctx, handler)
	listener(ctx, h)
	return <-done
}

// Drain returns a handler wrapper and a channel that will block until
// the event stream is done.
func Drain(ctx context.Context, handler Handler) (Handler, <-chan error) {
	done := make(chan error)
	return func(ctx context.Context, event interface{}) error {
		err := handler(ctx, event)
		if event == nil || err != nil {
			done <- err
			close(done)
		}
		return err
	}, done
}

// Monitor feeds an initial producer to the handler, and then adds a handler to the listener.
// It does all of that under a lock, and then waits for the event stream to drain after the
// lock is released.
// This is commonly used for things that want to feed an existing data set and then keep watching
// for new entries arriving.
func Monitor(ctx context.Context, lock sync.Locker, listener Listener, producer Producer, handler Handler) error {
	var done <-chan error
	if err := func() error { // To scope the lock correctly
		lock.Lock()
		defer lock.Unlock()
		if err := Feed(ctx, handler, producer); err != nil {
			return err
		}
		handler, done = Drain(ctx, handler)
		listener(ctx, handler)
		return nil
	}(); err != nil {
		return err
	}
	return <-done
}
