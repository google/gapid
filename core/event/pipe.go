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
	"io"
	"reflect"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/log"
)

// Buffer returns a handler that will feed the supplied handler, but inserts a unbounded buffer
// and goroutine switch between the two handlers so that the input handler always returns immediatly
// without waiting for the underlying handler to be invoked and complete.
// This is a *very* dangerous function to use, as it fully uncouples the handlers, and can result in
// unbounded memory consumption with no ability to flow control.
func Buffer(ctx context.Context, handler Handler) Handler {
	in := make(chan interface{})  // returned handler to buffer
	out := make(chan interface{}) // buffer to passed in handler
	closer := make(chan struct{}) // close signal
	events := []interface{}{}
	crash.Go(func() {
		// This is responsible for managing the buffer itself
		for {
			output := out
			if len(events) == 0 {
				output = nil
			}
			select {
			case output <- events[0]: // write an event to the out channel
				events = events[1:] // and remove it
			case event := <-in: // read an even from the in channel
				if event == nil {
					// TODO: teardown
				}
				events = append(events, event) // and store it
			case <-ctx.Done():
				// TODO: teardown
			}
		}
	})
	crash.Go(func() {
		// This is responsible for feeding the buffer to the passed in handler
		for {
			select {
			case event := <-out: // read an even from the in channel
				if event == nil {
					close(closer)
					return
				}
				handler(ctx, out)
			case <-ctx.Done():
				close(closer)
				return
			}
		}
	})
	// Now return a handler that feeds the buffer input channel
	return chanHandler(in, closer)
}

func chanProducer(from <-chan interface{}) Producer {
	return func(ctx context.Context) interface{} {
		select {
		case event := <-from:
			return event
		case <-ctx.Done():
			return nil
		}
	}
}

func chanHandler(to chan<- interface{}, closer <-chan struct{}) Handler {
	return func(ctx context.Context, event interface{}) error {
		select {
		case to <- event:
			if event == nil {
				close(to)
				return io.EOF
			}
			return nil
		case <-closer:
			close(to)
			return io.EOF
		case <-ctx.Done():
			close(to)
			return ctx.Err()
		}
	}
}

func chanToHandler(ctx context.Context, f reflect.Value) Handler {
	log.F(ctx, "Typed channels not yet supported")
	return nil
}

func chanToProducer(ctx context.Context, f reflect.Value) Producer {
	log.F(ctx, "Typed channels not yet supported")
	return nil
}
