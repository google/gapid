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

package monitor

import (
	"sync"
)

var alreadyClosed = make(chan struct{})

func init() { close(alreadyClosed) }

// Generation is a way to handle sequences of 'updates' or 'generations',
// find out how many have happened, as well as blocking until the next one happens.
// This does not assume an 'update' to mean anything specific.
type Generation struct {
	lock    sync.Mutex
	id      uint64
	changed chan struct{}
}

// ID returns the current generation identifier under the lock
func (g *Generation) ID() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.id
}

// Get returns the generation identifier and blocking channel under the lock
func (g *Generation) Get() (uint64, chan struct{}) {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.id, g.changed
}

// Update increments the generation and notifies all blocked goroutines
func (g *Generation) Update() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	close(g.changed)
	g.changed = make(chan struct{})
	g.id++
	return g.id
}

// WaitForUpdate blocks until the generation is past previous, and returns the
// current generation.
func (g *Generation) WaitForUpdate(previous uint64) uint64 {
	current, block := g.Get()
	if previous < current {
		// we are already past this point, no need to wait
		return current
	}
	<-block
	return g.ID()
}

// After returns a channel that will be closed as soon as the generation exceeds ID.
// This is intended for use in select calls, if you just want to wait, WaitForUpdate is
// more efficient.
func (g *Generation) After(id uint64) <-chan struct{} {
	current, block := g.Get()
	if id < current {
		// we are already past this point, no need to wait
		return alreadyClosed
	}
	return block
}

// NewGeneration returns a new Generation starting with an id of 0.
func NewGeneration() *Generation {
	return &Generation{
		changed: make(chan struct{}),
	}
}
