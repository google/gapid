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
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/gapis/config"
)

// NewInMemory builds a new in memory database.
func NewInMemory(ctx context.Context) Database {
	m := &memory{}
	m.records = map[id.ID]*record{}
	m.resolveCtx = Put(ctx, m)
	return m
}

type record struct {
	proto        proto.Message
	object       interface{}
	resolveState *resolveState
	created      callstack
}

type resolveState struct {
	ctx        context.Context // Context for the resolve
	err        error           // Error raised when resolving
	finished   chan struct{}   // Signal that resolve has finished. Set to nil when done.
	waiting    uint32          // Number of go-routines waiting for the resolve
	cancel     func()          // Cancels ctx
	callstacks []callstack
}

func (r *record) resolve(ctx context.Context) error {
	// Deserialize the object from the proto if we don't have the object already.
	if r.object == nil {
		obj, err := protoconv.ToObject(ctx, r.proto)
		switch err := err.(type) {
		case protoconv.ErrNoConverterRegistered:
			if err.Object != r.proto {
				// We got a ErrNoConverterRegistered error, but it wasn't for the outermost object!
				return err
			}
			r.object = r.proto
		case nil:
			r.object = obj
		default:
			return err
		}
	}
	for {
		// If the object implements resolvable, then we need to resolve it.
		// Is the database value resolvable?
		resolvable, isResolvable := r.object.(Resolvable)
		if !isResolvable {
			return nil
		}
		resolved, err := resolvable.Resolve(ctx)
		if err != nil {
			return err
		}
		r.object = resolved
	}
}

type memory struct {
	mutex      sync.Mutex
	records    map[id.ID]*record
	resolveCtx context.Context
}

// Implements Database
func (d *memory) Store(ctx context.Context, id id.ID, v interface{}, m proto.Message) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.storeLocked(ctx, id, v, m)
}

// store function must be called with a locked mutex
func (d *memory) storeLocked(ctx context.Context, id id.ID, v interface{}, m proto.Message) error {
	if v == nil && m == nil {
		panic(fmt.Errorf("Store nil in database (that is bad), id '%v'", id))
	}
	r, got := d.records[id]
	if !got {
		d.records[id] = &record{object: v, proto: m, created: getCallstack(4)}
	} else if config.DebugDatabaseVerify {
		if !reflect.DeepEqual(m, r.proto) {
			return fmt.Errorf("Duplicate object id %v", id)
		}
	}
	return nil
}

// Implements Database
func (d *memory) Resolve(ctx context.Context, id id.ID) (interface{}, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.resolveLocked(ctx, id)
}

// resolve function must be called with a locked mutex and returns with a locked
// mutex.
func (d *memory) resolveLocked(ctx context.Context, id id.ID) (interface{}, error) {
	// Look up the record with the provided identifier.
	r, got := d.records[id]
	if !got {
		// Database doesn't recognise this identifier.
		return nil, fmt.Errorf("Resource '%v' not found", id)
	}

	rs := r.resolveState
	if rs == nil {
		// First request for this resolvable.

		// Grab the resolve chain from the caller's context.
		rc := &resolveChain{r, getResolveChain(ctx)}

		// Build a cancellable context for the resolve from database's resolve
		// context. We use this as we don't to cancel the resolve if a single
		// caller cancel's their context.
		resolveCtx, cancel := task.WithCancel(d.resolveCtx)

		rs = &resolveState{
			ctx:      rc.bind(resolveCtx),
			finished: make(chan struct{}),
			cancel:   cancel,
		}
		r.resolveState = rs

		// Build the resolvable on a separate go-routine.
		go func(ctx context.Context) {
			defer d.resolvePanicHandler(ctx)
			err := r.resolve(ctx)

			// Signal that the resolvable has finished.
			d.mutex.Lock()
			close(rs.finished)
			rs.err, rs.finished = err, nil
			d.mutex.Unlock()
		}(rs.ctx)
	}

	if finished := rs.finished; finished != nil {
		// Buildable has not yet finished.
		// Increment the waiting go-routine counter.
		rs.waiting++
		rs.callstacks = append(rs.callstacks, getCallstack(4))
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
	return r.object, nil // Done.
}

// Implements Database
func (d *memory) Contains(ctx context.Context, id id.ID) (res bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	_, got := d.records[id]
	return got
}
