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

import "time"

// Baton implements a task interlock.
// A baton must be owned by exactly 1 task at any given time, so a release blocks until a corresponding acquire occurs.
// You can pass a value through the baton from the release to the acquire.
type Baton chan interface{}

// NewBaton returns a new Baton, with the expectation that the calling goroutine owns the baton.
func NewBaton() Baton { return make(Baton) }

// Acquire is a request to pick up the baton, it will block until another goroutine releases it.
// It returns the value passed to the release that triggers it.
func (b Baton) Acquire() interface{} {
	return <-b
}

// TryAcquire is a request to pick up the baton, it will block until another goroutine releases it or timeout passes.
// It will return the released value and  true if the baton was successfully acquired.
func (b Baton) TryAcquire(timeout time.Duration) (interface{}, bool) {
	select {
	case value := <-b:
		return value, true
	case <-time.After(timeout):
		return nil, false
	}
}

// Release is a request to relinquish the baton, it will block until another goroutine acquires it.
// The supplied value is returned from the Acquire this release triggers.
func (b Baton) Release(value interface{}) {
	b <- value
}

// TryRelease is a request to relinquish the baton, it will block until another goroutine acquires it or timeout passes.
// It will return true if the baton was successfully released.
func (b Baton) TryRelease(value interface{}, timeout time.Duration) bool {
	select {
	case b <- value:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Yield is helper that does Release followed by an Acquire.
// It waits for another goroutine to acquire the baton, and then waits for the baton to be released back to this goroutine.
func (b Baton) Yield(value interface{}) interface{} {
	b.Release(value)
	return b.Acquire()
}

// Relay is a helper that does an Acquire followed by a Release with the value that came from the Acquire.
// This waits for the baton to be available, and then immediately passes back, used as a signalling gate.
func (b Baton) Relay() {
	value := b.Acquire()
	b.Release(value)
}
