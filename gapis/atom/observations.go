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

package atom

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// Observations is a collection of reads and write observations performed by an
// atom.
type Observations struct {
	binary.Generate `javaDefineEmptyArrays:"true"`
	Reads           []Observation
	Writes          []Observation
}

func (o *Observations) String() string {
	return fmt.Sprintf("Reads: %v, Writes: %v", o.Reads, o.Writes)
}

// AddRead appends the read to the list of observations.
func (o *Observations) AddRead(rng memory.Range, id id.ID) {
	o.Reads = append(o.Reads, Observation{Range: rng, ID: id})
}

// AddWrite appends the write to the list of observations.
func (o *Observations) AddWrite(rng memory.Range, id id.ID) {
	o.Writes = append(o.Writes, Observation{Range: rng, ID: id})
}

// ApplyReads applies all the observed reads to memory pool p.
// This is a no-op when called when o is nil.
func (o *Observations) ApplyReads(p *memory.Pool) {
	if o != nil {
		for _, r := range o.Reads {
			p.Write(r.Range.Base, memory.Resource(r.ID, r.Range.Size))
		}
	}
}

// ApplyReads applies all the observed writes to the memory pool p.
// This is a no-op when called when o is nil.
func (o *Observations) ApplyWrites(p *memory.Pool) {
	if o != nil {
		for _, w := range o.Writes {
			p.Write(w.Range.Base, memory.Resource(w.ID, w.Range.Size))
		}
	}
}

// DataString returns a string describing all reads/writes and their raw data.
func (o *Observations) DataString(ctx context.Context) string {
	var buf bytes.Buffer
	db := database.Get(ctx)
	for _, read := range o.Reads {
		buf.WriteString(fmt.Sprintf("[read] %v\n", read))
		if data, err := db.Resolve(ctx, read.ID); err == nil {
			buf.WriteString(fmt.Sprintf("[data] %v\n", data))
		} else {
			buf.WriteString(fmt.Sprintf("[data] failed: %v\n", err))
		}
	}
	for _, write := range o.Writes {
		buf.WriteString(fmt.Sprintf("[write] %v\n", write))
		if data, err := db.Resolve(ctx, write.ID); err == nil {
			buf.WriteString(fmt.Sprintf("[data] %v\n", data))
		} else {
			buf.WriteString(fmt.Sprintf("[data] failed: %v\n", err))
		}
	}
	return buf.String()
}

func (o *Observations) Convert(ctx context.Context, out atom_pb.Handler) error {
	for _, entry := range o.Reads {
		if err := entry.Convert(ctx, out); err != nil {
			return err
		}
	}
	if len(o.Writes) > 0 {
		if err := out(ctx, atom_pb.InvokeMarker); err != nil {
			return err
		}
		for _, entry := range o.Writes {
			if err := entry.Convert(ctx, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// Observation represents a single read or write observation made by an atom.
type Observation struct {
	binary.Generate
	Range memory.Range // Memory range that was observed.
	ID    id.ID        // The resource identifier of the observed data.
}

func (o Observation) String() string {
	return fmt.Sprintf("{Range: %v, ID: %v}", o.Range, o.ID)
}

func (o *Observation) Convert(ctx context.Context, out atom_pb.Handler) error {
	return out(ctx, &memory_pb.Observation{
		Base: o.Range.Base,
		Size: o.Range.Size,
		Id:   o.ID.String(),
	})
}

func ObservationFrom(from *memory_pb.Observation) Observation {
	o := Observation{}
	o.Range.Base = from.Base
	o.Range.Size = from.Size
	o.ID.Parse(from.Id)
	return o
}
