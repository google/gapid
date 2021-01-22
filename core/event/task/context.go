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
	"time"
)

// Provide a mean to let a context ignore cancellation. Use unexported type and
// variable to force other packages to use IgnoreCancellation().
type key int

var ignoreCancellationKey key

// IgnoreCancellation returns a context that will pretend to never be canceled.
func IgnoreCancellation(ctx context.Context) context.Context {
	return context.WithValue(ctx, ignoreCancellationKey, struct{}{})
}

// CancelFunc is a function type that can be used to stop a context.
type CancelFunc context.CancelFunc

// WithCancel returns a copy of ctx with a new Done channel.
// See context.WithCancel for more details.
func WithCancel(ctx context.Context) (context.Context, CancelFunc) {
	c, cancel := context.WithCancel(ctx)
	return c, CancelFunc(cancel)
}

// WithDeadline returns a copy of ctx with the deadline adjusted to be no later than deadline.
// See context.WithDeadline for more details.
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, CancelFunc) {
	c, cancel := context.WithDeadline(ctx, deadline)
	return c, CancelFunc(cancel)
}

// WithTimeout is shorthand for ctx.WithDeadline(time.Now().Add(duration)).
// See context.Context.WithTimeout for more details.
func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, CancelFunc) {
	return WithDeadline(ctx, time.Now().Add(duration))
}

// ShouldStop returns a chan that's closed when work done on behalf of this
// context should be stopped.
// See context.Context.Done for more details.
func ShouldStop(ctx context.Context) <-chan struct{} {
	if ctx.Value(ignoreCancellationKey) != nil {
		// return the nil channel from which reading blocks forever
		return nil
	}
	return ctx.Done()
}

// StopReason returns a non-nil error value after Done is closed.
// See context.Context.Err for more details.
func StopReason(ctx context.Context) error {
	if ctx.Value(ignoreCancellationKey) != nil {
		// pretend to never be canceled
		return nil
	}
	return ctx.Err()
}

// Stopped is shorthand for StopReason(ctx) != nil because it increases the readability of common use cases.
func Stopped(ctx context.Context) bool {
	return ctx.Err() != nil
}
