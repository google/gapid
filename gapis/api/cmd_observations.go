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

package api

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// ResourceReference is the interface implemented by types that hold references
// to Resource identifiers which require remapping. Resources are stored in
// the capture files without any identifier and are referenced by index.
// When resources are stored into the server database, the database returns
// an identifier for this data. We need to transform the capture resource
// index to the database resource identifier when the capture is loaded.
//
// Similarly, we also need to iterate over all referenced resources when
// we store capture, save them, and map identifiers back to indices.
type ResourceReference interface {
	// RemapResourceIDs calls the given callback for each resource ID field.
	// The callback may modify (remap) the ID.
	// The function returns the, now potentially modified, copy of the object.
	RemapResourceIDs(cb func(id *id.ID, idx *int64) error) (ResourceReference, error)
}

// CmdObservations is a collection of reads and write observations performed by an
// command.
type CmdObservations struct {
	Reads  []CmdObservation
	Writes []CmdObservation
}

func (o *CmdObservations) String() string {
	return fmt.Sprintf("Reads: %v, Writes: %v", o.Reads, o.Writes)
}

// AddRead appends the read to the list of observations.
func (o *CmdObservations) AddRead(rng memory.Range, id id.ID) {
	o.Reads = append(o.Reads, CmdObservation{Range: rng, ID: id})
}

// AddWrite appends the write to the list of observations.
func (o *CmdObservations) AddWrite(rng memory.Range, id id.ID) {
	o.Writes = append(o.Writes, CmdObservation{Range: rng, ID: id})
}

// ApplyReads applies all the observed reads to memory pool p.
// This is a no-op when called when o is nil.
func (o *CmdObservations) ApplyReads(p *memory.Pool) {
	if o != nil {
		for _, r := range o.Reads {
			p.Write(r.Range.Base, memory.Resource(r.ID, r.Range.Size))
		}
	}
}

// ApplyWrites applies all the observed writes to the memory pool p.
// This is a no-op when called when o is nil.
func (o *CmdObservations) ApplyWrites(p *memory.Pool) {
	if o != nil {
		for _, w := range o.Writes {
			p.Write(w.Range.Base, memory.Resource(w.ID, w.Range.Size))
		}
	}
}

// DataString returns a string describing all reads/writes and their raw data.
func (o *CmdObservations) DataString(ctx context.Context) string {
	var buf bytes.Buffer
	for _, read := range o.Reads {
		buf.WriteString(fmt.Sprintf("[read] %v\n", read))
		if data, err := database.Resolve(ctx, read.ID); err == nil {
			buf.WriteString(fmt.Sprintf("[data] %v\n", data))
		} else {
			buf.WriteString(fmt.Sprintf("[data] failed: %v\n", err))
		}
	}
	for _, write := range o.Writes {
		buf.WriteString(fmt.Sprintf("[write] %v\n", write))
		if data, err := database.Resolve(ctx, write.ID); err == nil {
			buf.WriteString(fmt.Sprintf("[data] %v\n", data))
		} else {
			buf.WriteString(fmt.Sprintf("[data] failed: %v\n", err))
		}
	}
	return buf.String()
}

// CmdObservation represents a single read or write observation made by an
// command.
type CmdObservation struct {
	Range    memory.Range // Memory range that was observed.
	ID       id.ID        // The resource identifier of the observed data.
	ResIndex int64        // The resource index of the observed data.
}

func (o CmdObservation) String() string {
	return fmt.Sprintf("{Range: %v, ID: %v}", o.Range, o.ID)
}

var _ ResourceReference = CmdObservation{}

// RemapResourceIDs calls the given callback for each resource ID field.
func (o CmdObservation) RemapResourceIDs(cb func(id *id.ID, idx *int64) error) (ResourceReference, error) {
	err := cb(&o.ID, &o.ResIndex)
	return o, err
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a CmdObservation) (*memory_pb.Observation, error) {
			return &memory_pb.Observation{
				Base:     a.Range.Base,
				Size:     a.Range.Size,
				Id:       a.ID[:],
				ResIndex: a.ResIndex,
			}, nil
		},
		func(ctx context.Context, a *memory_pb.Observation) (CmdObservation, error) {
			o := CmdObservation{}
			o.Range.Base = a.Base
			o.Range.Size = a.Size
			copy(o.ID[:], a.Id)
			o.ResIndex = a.ResIndex
			return o, nil
		},
	)
}
