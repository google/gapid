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
	"crypto/sha1"
	"fmt"
	"hash"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/event/task"
)

// NewInMemory builds a new in memory database.
func NewInMemory(ctx context.Context) Database {
	m := &memory{}
	m.records = map[id.ID]*record{}
	m.resolveCtx = Put(ctx, m)
	return m
}

var sha1Pool = sync.Pool{New: func() interface{} { return sha1.New() }}

func generateID(ty recordType, encoded []byte) id.ID {
	if ty == blobFunc {
		ty = blob
	}
	h := sha1Pool.Get().(hash.Hash)
	h.Reset()
	h.Write([]byte(ty))
	h.Write([]byte("•"))
	h.Write(encoded)

	out := id.ID{}
	copy(out[:], h.Sum(nil))

	sha1Pool.Put(h)
	return out
}

type recordType string

// blob is the record type for a raw byte slice
const blob = recordType("<blob>")

// blobFunc is the record type for a function that returns
// a raw byte slice
const blobFunc = recordType("<blobFunc>")

type record struct {
	data         []byte      // data is the encoded object
	ty           recordType  // ty is the type of the encoded object
	object       interface{} // object is the deserialized object
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

func (r *record) decode(ctx context.Context) (interface{}, error) {
	switch r.ty {
	case blob:
		return r.data, nil
	default:
		ty := proto.MessageType(string(r.ty))
		msg := reflect.New(ty).Interface().(proto.Message)
		if err := proto.Unmarshal(r.data, msg); err != nil {
			return nil, err
		}
		return msg, nil
	}
}

func (r *record) resolve(ctx context.Context) error {
	// Decode the object if we don't have the object already.
	if r.object == nil {
		obj, err := r.decode(ctx)
		if err != nil {
			return err
		}
		r.object = obj
	}

	// Convert protos to Go objects if we can.
	if msg, ok := r.object.(proto.Message); ok {
		obj, err := protoconv.ToObject(ctx, msg)
		switch err := err.(type) {
		case nil:
			r.object = obj
		case protoconv.ErrNoConverterRegistered:
			if err.Object != msg {
				// We got a ErrNoConverterRegistered error, but it wasn't for
				// the outermost object!
				return err
			}
		default:
			return err
		}
	}

	// Keep on resolving until the type no longer implements Resolvable.
	for {
		// If the object implements resolvable, then we need to resolve it.
		// Is the database value resolvable?
		resolvable, isResolvable := r.object.(Resolvable)
		if !isResolvable {
			return nil
		}
		ctx = status.Start(ctx, "DB Resolve<%T> %p", resolvable, r.resolveState)
		defer status.Finish(ctx)
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
func (d *memory) Store(ctx context.Context, val interface{}) (id.ID, error) {
	var data []byte
	var ty recordType
	dontStoreData := false

	switch val := val.(type) {
	case nil:
		panic(fmt.Errorf("Attemping to store nil in database"))
	case []byte:
		data, ty = val, blob
	case func(ctx context.Context) ([]byte, error):
		dat, err := val(ctx)
		if err != nil {
			return id.ID{}, err
		}
		data, ty = dat, blobFunc
		dontStoreData = true
	default:
		m, err := toProto(ctx, val)
		if err != nil {
			return id.ID{}, err
		}
		data, err = proto.Marshal(m)
		if err != nil {
			return id.ID{}, err
		}
		ty = recordType(proto.MessageName(m))
	}

	id := generateID(ty, data)

	d.mutex.Lock()
	defer d.mutex.Unlock()
	if _, got := d.records[id]; !got {
		if dontStoreData {
			d.records[id] = &record{data: nil, ty: ty, object: val, created: getCallstack(4)}
		} else {
			d.records[id] = &record{data: data, ty: ty, object: val, created: getCallstack(4)}
		}
	}

	return id, nil
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
	build := rs == nil
	if build {
		// First request for this resolvable.

		// Grab the resolve chain from the caller's context.
		rc := &resolveChain{r, getResolveChain(ctx)}

		// Build a cancellable context for the resolve from database's resolve
		// context. We use this as we don't want to cancel the resolve if a
		// single caller cancel's their context.
		resolveCtx, cancel := task.WithCancel(d.resolveCtx)

		rs = &resolveState{
			ctx:      rc.bind(resolveCtx),
			finished: make(chan struct{}),
			cancel:   cancel,
		}
		r.resolveState = rs

		// Build the resolvable on a separate go-routine.
		ctx := ctx // Don't let changes to ctx leak into this go-routine.
		crash.Go(func() {
			// Propagate the status, so that resolve tasks appear under the
			// context that first triggered the resolve.
			ctx := status.PutTask(rs.ctx, status.GetTask(ctx))

			defer d.resolvePanicHandler(ctx)
			err := r.resolve(ctx)

			// Signal that the resolvable has finished.
			d.mutex.Lock()
			close(rs.finished)
			rs.err, rs.finished = err, nil
			d.mutex.Unlock()
		})
	}

	if finished := rs.finished; finished != nil {
		if !build {
			ctx = status.StartBackground(ctx, "Wait DB Resolve<%T> %p", r.object, rs)
			defer status.Finish(ctx)
			status.Block(ctx)
			defer status.Unblock(ctx)
		}

		// Buildable has not yet finished.
		// Increment the waiting go-routine counter.
		rs.waiting++
		rs.callstacks = append(rs.callstacks, getCallstack(4))
		// Wait for either the resolve to finish or ctx to be cancelled.
		d.mutex.Unlock()
		var cancelledContextErr error
		select {
		case <-finished:
		case <-task.ShouldStop(ctx):
			cancelledContextErr = task.StopReason(ctx)
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

		if cancelledContextErr != nil {
			return nil, cancelledContextErr
		}
	}

	if rs.err != nil {
		return nil, rs.err // Resolve errored.
	}

	if r.ty == blobFunc {
		if x, ok := r.object.(func(ctx context.Context) ([]byte, error)); ok {
			return func() ([]byte, error) {
				d.mutex.Unlock()
				defer d.mutex.Lock()
				return x(ctx)
			}()
		}
		return nil, fmt.Errorf("Resource '%v' of incorrect type", id)
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

// Implements Database
func (d *memory) IsResolved(ctx context.Context, id id.ID) (res bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	r, got := d.records[id]
	if !got {
		return false
	}
	rs := r.resolveState
	if rs.finished == nil {
		return true
	}
	return false
}
