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
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

type (
	// Converter is the type for a function that converts from a storage atom
	// to a live one.
	Converter func(atom_pb.Atom) interface{}

	// Convertible is the interface to something that can be converted to
	// a proto.Atom stream.
	Convertible interface {
		// Convert emits the stream of serialized atoms for this object.
		Convert(log.Context, atom_pb.Handler) error
	}

	invokeMarker struct{}
)

const (
	// ErrNotConvertible is the error to indicate that an object that does not
	// support the atom.Convertible interface was found in an atom list that
	// was being converted.
	ErrNotConvertible = fault.Const("Object is not atom.Convertible")
)

var (
	converters = []Converter{internalConverter}
)

// RegisterConverter adds a new conversion function to the set.
func RegisterConverter(c Converter) {
	converters = append(converters, c)
}

// ConvertAllTo converts from the set of live atoms to a stream of storage atoms.
func ConvertAllTo(ctx log.Context, atoms *List, handler func(log.Context, atom_pb.Atom) error) error {
	for _, a := range atoms.Atoms {
		c, ok := a.(Convertible)
		if !ok {
			return ErrNotConvertible
		}
		if err := c.Convert(ctx, handler); err != nil {
			return err
		}
	}
	return nil
}

// ConvertFrom converts from a storage form to a live atom for any atom from this package.
// It returns nil for any type it does not understand.
func ConvertFrom(from atom_pb.Atom) interface{} {
	// TODO: Memoise the type to converter mapping so we don't need to scan the list in future
	for _, converter := range converters {
		if a := converter(from); a != nil {
			return a
		}
	}
	return nil
}

// ConvertAllFrom goes through the list of storage atoms converting them all, to
// the live form and handing the converted to the handler.
func ConvertAllFrom(ctx log.Context, atoms []atom_pb.Atom, handler func(a Atom)) error {
	converter := FromConverter(handler)
	for _, a := range atoms {
		if err := converter(ctx, a); err != nil {
			return err
		}
	}
	return converter(ctx, nil)
}

// FromConverter returns a function that converts all the storage atoms it is handed,
// passing the generated live atoms to the handler.
// You must call this with a nil to flush the final atom.
func FromConverter(handler func(a Atom)) func(log.Context, atom_pb.Atom) error {
	var (
		last         Atom
		observations *Observations
		invoked      bool
		count        int
	)
	return func(ctx log.Context, in atom_pb.Atom) error {
		count++
		if in == nil {
			if last != nil {
				handler(last)
			}
			last = nil
			return nil
		}
		out := ConvertFrom(in)
		switch out := out.(type) {
		case Atom:
			if last != nil {
				handler(last)
			}
			last = out
			invoked = false
			observations = nil
		case Observation:
			if observations == nil {
				observations = &Observations{}
				e := last.Extras()
				if e == nil {
					return cause.Explainf(ctx, nil, "Not allowed extras %T:%v", last, last)
				}
				*e = append(*e, observations)
			}
			if !invoked {
				observations.Reads = append(observations.Reads, out)
			} else {
				observations.Writes = append(observations.Writes, out)
			}
		case Extra:
			e := last.Extras()
			if e == nil {
				return cause.Explainf(ctx, nil, "Not allowed extras %T:%v", last, last)
			}
			*e = append(*e, out)
		case invokeMarker:
			invoked = true
		default:
			return cause.Explainf(ctx, nil, "Unhandled type during conversion %T:%v", out, out)
		}
		return nil
	}
}

func internalConverter(from atom_pb.Atom) interface{} {
	switch from := from.(type) {
	case *atom_pb.Invoke:
		return invokeMarker{}
	case *atom_pb.Aborted:
		to := AbortedFrom(from)
		return &to
	case *atom_pb.Resource:
		to := ResourceFrom(from)
		return &to
	case *atom_pb.FramebufferObservation:
		to := FramebufferObservationFrom(from)
		return &to
	case *atom_pb.FieldAlignments:
		to := FieldAlignmentsFrom(from)
		return &to
	case *memory_pb.Observation:
		return ObservationFrom(from)
	case *memory_pb.Pointer:
		to := memory.PointerFrom(from)
		return &to
	case *memory_pb.Slice:
		to := memory.SliceInfoFrom(from)
		return &to
	default:
		return nil
	}
}
