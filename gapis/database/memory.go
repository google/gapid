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

package database

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/config"
)

// NewInMemory builds a new in memory database.
func NewInMemory(ctx log.Context) Database {
	m := &memory{}
	m.records = map[id.ID]*record{}
	m.resolveCtx = Put(ctx, m)
	return m
}

type record struct {
	value        interface{}
	resolveState *resolveState
}

type resolveState struct {
	ctx      log.Context   // Context for the resolve
	valID    id.ID         // Identifier of the resolved result
	err      error         // Error raised when resolving
	finished chan struct{} // Signal that resolve has finished. Set to nil when done.
	waiting  uint32        // Number of go-routines waiting for the resolve
	cancel   func()        // Cancels ctx
}

type memory struct {
	mutex      sync.Mutex
	records    map[id.ID]*record
	resolveCtx log.Context
}

// Implements Database
func (d *memory) Store(ctx log.Context, id id.ID, v interface{}) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.store(ctx, id, v)
}

// store function must be called with a locked mutex
func (d *memory) store(ctx log.Context, id id.ID, v interface{}) error {
	if v == nil {
		panic(fmt.Errorf("Store nil in database (that is bad), id '%v'", id))
	}
	r, got := d.records[id]
	if !got {
		d.records[id] = &record{value: v}
	} else if config.DebugDatabaseVerify {
		if !reflect.DeepEqual(v, r.value) {
			return fmt.Errorf("Duplicate object id %v", id)
		}
	}
	return nil
}

// Implements Database
func (d *memory) Resolve(ctx log.Context, id id.ID) (interface{}, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.resolve(ctx, id)
}

// resolve function must be called with a locked mutex and returns with a locked
// mutex.
func (d *memory) resolve(ctx log.Context, id id.ID) (interface{}, error) {
	// Look up the record with the provided identifier.
	r, got := d.records[id]
	if !got {
		// Database doesn't recognise this identifier.
		return nil, fmt.Errorf("Resource '%v' not found", id)
	}

	// Is the database value resolvable?
	resolvable, isResolvable := r.value.(Resolvable)
	if !isResolvable {
		// Non-resolvable object. Just return the value.
		return r.value, nil
	}

	rs := r.resolveState
	if rs == nil {
		// First request for this resolvable.

		// Mutate the resolvable identifier to get the result value identifier.
		valID := resolvedID(id)

		// Build a cancellable context for the resolve.
		resolveCtx, cancel := task.WithCancel(d.resolveCtx)

		rs = &resolveState{
			ctx:      resolveCtx,
			valID:    valID,
			finished: make(chan struct{}),
			cancel:   cancel,
		}
		r.resolveState = rs

		// Build the resolvable on a separate go-routine.
		go func() {
			val, err := resolvable.Resolve(rs.ctx)
			if err == nil {
				// Resolved without error. Store the resulting values.
				err = d.Store(ctx, rs.valID, val)
			}
			// Signal that the resolvable has finished.
			d.mutex.Lock()
			close(rs.finished)
			rs.err, rs.finished = err, nil
			d.mutex.Unlock()
		}()
	}

	if finished := rs.finished; finished != nil {
		// Buildable has not yet finished.
		// Increment the waiting go-routine counter.
		rs.waiting++

		// Wait for either the resolve to finish or ctx to be cancelled.
		d.mutex.Unlock()
		select {
		case <-finished:
		case <-task.ShouldStop(ctx):
		}
		d.mutex.Lock()

		// Decrement the waiting go-routine counter.
		rs.waiting--
		if rs.waiting == 0 && rs.finished != nil {
			// There's no more go-routines waiting for this resolvable and it
			// hasn't finished yet. Cancel it and remove the resolve state from
			// the record.
			rs.cancel()
			r.resolveState = nil
			d.records[id] = r
		}
	}

	if err := task.StopReason(ctx); err != nil {
		return nil, err // Context was cancelled.
	}
	if rs.err != nil {
		return nil, rs.err // Resolve errored.
	}
	// Resolve was successful.
	// Resolve the value identifier to get the goods.
	return d.resolve(ctx, rs.valID)
}

// Implements Database
func (d *memory) Contains(ctx log.Context, id id.ID) (res bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	_, got := d.records[id]
	return got
}
