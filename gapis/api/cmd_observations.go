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
	Pool  memory.PoolID // The pool in which the memory was observed.
	Range memory.Range  // Memory range that was observed.
	ID    id.ID         // The resource identifier of the observed data.
}

func (o CmdObservation) String() string {
	return fmt.Sprintf("{Range: %v, ID: %v}", o.Range, o.ID)
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a CmdObservation) (*memory_pb.Observation, error) {
			resIndex, err := id.GetRemapper(ctx).RemapID(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			return &memory_pb.Observation{
				Pool:     uint32(a.Pool),
				Base:     a.Range.Base,
				Size:     a.Range.Size,
				ResIndex: resIndex,
			}, nil
		},
		func(ctx context.Context, a *memory_pb.Observation) (CmdObservation, error) {
			id, err := id.GetRemapper(ctx).RemapIndex(ctx, a.ResIndex)
			if err != nil {
				return CmdObservation{}, err
			}
			o := CmdObservation{}
			o.Pool = memory.PoolID(a.Pool)
			o.Range.Base = a.Base
			o.Range.Size = a.Size
			o.ID = id
			return o, nil
		},
	)
}
