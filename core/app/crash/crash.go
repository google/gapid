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

// Package crash provides functions for reporting application crashes (uncaught
// panics).
//
// crash does not offer any sort of global crash recovery mechanism.
package crash

import (
	"sync"

	"github.com/google/gapid/core/fault/stacktrace"
)

var (
	mutex     sync.RWMutex
	reporters []Reporter
)

// Reporter is a function that reports an uncaught panic that will crash the
// application.
type Reporter func(e interface{}, s stacktrace.Callstack)

// Register adds r to the list of functions that gets called when an uncaught
// panic is thrown.
func Register(r Reporter) {
	mutex.Lock()
	defer mutex.Unlock()
	reporters = append(reporters, r)
}

// handler catches any uncaught panics, reporting these to the registered
// crash handlers, and then rethrows the panic.
func handler() {
	if e := recover(); e != nil {
		Crash(e)
	}
}

// Go calls f on a new go-routine, reporting any uncaught panics to the
// registered crash handlers.
func Go(f func()) {
	go func() {
		defer handler()
		f()
	}()
}

var crashOnce = sync.Once{}

// Crash invokes each of the registered crash reporters with e, then panics with
// e.
func Crash(e interface{}) {
	stack := stacktrace.Capture()
	crashOnce.Do(func() {
		mutex.RLock()
		defer mutex.RUnlock()
		for _, r := range reporters {
			r(e, stack)
		}
		panic(e)
	})
}
