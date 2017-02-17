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

	"github.com/google/gapid/core/log"
)

// CancelFunc is a function type that can be used to stop a context.
type CancelFunc context.CancelFunc

// WithCancel returns a copy of ctx with a new Done channel.
// See context.WithCancel for more details.
func WithCancel(ctx log.Context) (log.Context, CancelFunc) {
	c, cancel := context.WithCancel(ctx.Unwrap())
	return log.Wrap(c), CancelFunc(cancel)
}

// WithDeadline returns a copy of ctx with the deadline adjusted to be no later than deadline.
// See context.WithDeadline for more details.
func WithDeadline(ctx log.Context, deadline time.Time) (log.Context, CancelFunc) {
	c, cancel := context.WithDeadline(ctx.Unwrap(), deadline)
	return log.Wrap(c), CancelFunc(cancel)
}

// WithTimeout is shorthand for ctx.WithDeadline(time.Now().Add(duration)).
// See context.Context.WithTimeout for more details.
func WithTimeout(ctx log.Context, duration time.Duration) (log.Context, CancelFunc) {
	return WithDeadline(ctx, time.Now().Add(duration))
}

// ShouldStop returns a chan that's closed when work done on behalf of this
// context should be stopped.
// See context.Context.Done for more details.
func ShouldStop(ctx log.Context) <-chan struct{} {
	return ctx.Unwrap().Done()
}

// StopReason returns a non-nil error value after Done is closed.
// See context.Context.Err for more details.
func StopReason(ctx log.Context) error {
	return ctx.Unwrap().Err()
}

// Stopped is shorthand for StopReason(ctx) != nil because it increases the readability of common use cases.
func Stopped(ctx log.Context) bool {
	return ctx.Unwrap().Err() != nil
}
