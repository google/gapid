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

// FiredSignal is a signal that is always in the fired state.
var FiredSignal Signal

func init() {
	fired := make(chan struct{})
	close(fired)
	FiredSignal = fired
}

// Signal is used to notify that a task has completed.
// Nothing is ever sent through a signal, it is closed to indicate signalled.
type Signal <-chan struct{}

// NewSignal builds a new signal, and then returns the signal and a Task that is used to fire the signal.
// The returned fire Task must only be called once.
func NewSignal() (Signal, Task) {
	c := make(chan struct{})
	return c, func(context.Context) error { close(c); return nil }
}

// Fired returns true if the signal has been fired.
func (s Signal) Fired() bool {
	select {
	case <-s:
		return true
	default:
		return false
	}
}

// Wait blocks until the signal has been fired or the context has been
// cancelled.
// Returns true if the signal was fired, false if the context was cancelled.
func (s Signal) Wait(ctx context.Context) bool {
	select {
	case <-s:
		return true
	case <-ShouldStop(ctx):
		return false
	}
}

// TryWait waits for the signal to fire, the context to be cancelled or the
// timeout, whichever comes first.
// Returns true if the signal was fired, false if the context was cancelled or
// the timeout was reached.
func (s Signal) TryWait(ctx context.Context, timeout time.Duration) bool {
	select {
	case <-s:
		return true
	case <-ShouldStop(ctx):
		return false
	case <-time.After(timeout):
		return false
	}
}
