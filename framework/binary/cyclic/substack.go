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

package cyclic

import (
	"fmt"
	"strings"

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/framework/binary/schema"
)

// substack is a stack of type objects. Only types which need decoder
// support for nested sub-structures are added to the stack. The
// substack is an implementation detail of the cyclic decoder.
type substack struct {
	stack []binary.SubspaceType
}

// repeat is a special stack element object. It is used so that we can
// do counted loops without prefilling the stack with the subtypes for
// all the iterations in advance. The 'repeat' element replaces the
// first subtype of the loop (except in the first iteration). Stack elements
// of type 'repeat' are treated specially by the stack. Calling popType()
// when a 'repeat' element is on the top of the stack returns the subtype
// r.replaces() and pushes the subtypes returned by r.repetition().
type repeat struct {
	count   uint32              // The number of times to repeat the element.
	repeats binary.SubspaceType // The type to repeat
}

// Verify binary.SubspaceType implemented
var _ binary.SubspaceType = &repeat{}

func (r *repeat) subtypes() binary.TypeList {
	subspace := r.repeats.Subspace()
	subtypes := subspace.ExpandSubTypes()
	if !subspace.Counted || subspace.Inline || len(subspace.SubTypes) == 0 {
		panic(fmt.Errorf("Bad repeating element: %v", r))
	}
	return subtypes
}

// replaces, returns the type of the element replaced by the repeating element.
// This is the first sub-type in the loop.
func (r *repeat) replaces() binary.SubspaceType {
	return r.subtypes()[0]
}

// repetition, returns the type list needed to complete the loop, including
// further repetitions, if needed.
func (r *repeat) repetition() binary.TypeList {
	// The remaining elements of the loop
	subs := r.subtypes()[1:]
	if r.count != 1 {
		// further iterations are required add a new repeat element with a lower
		// count.
		subs = append(binary.TypeList{}, subs...)
		subs = append(subs, &repeat{count: r.count - 1, repeats: r.repeats})
	}
	return subs
}

// Subspace is provided to satisfy the binary.SubspaceType interface
func (r *repeat) Subspace() *binary.Subspace {
	return r.replaces().Subspace()
}

// HasSubspace is provided to satisfy the binary.SubspaceType interface
func (r *repeat) HasSubspace() bool {
	return r.repeats.HasSubspace()
}

// Format implements the fmt.Formatter interface
func (r *repeat) Format(f fmt.State, c rune) {
	repeatSub := r.repeats.Subspace().SubTypes
	if r.count == 1 {
		fmt.Fprintf(f, "(final repetition of %"+string(c)+")", repeatSub)
	} else if r.count > 1 {
		fmt.Fprintf(f, "(%d repetitions of %"+string(c)+")", r.count, repeatSub)
	} else {
		fmt.Fprintf(f, "(%d repeat element shouldn't exist for %"+string(c)+")", r.count, repeatSub)
	}
}

// String returns a description of the substack.
func (s substack) String() string {
	parts := make([]string, len(s.stack))
	for i, t := range s.stack {
		parts[i] = fmt.Sprintf("(%d): %v", i, t)
	}
	return strings.Join(parts, "\n")
}

// pushStruct pushes the sub-types needed to decode a struct described
// the the schema object 'ent'.
func (s *substack) pushStruct(ent *binary.Entity) {
	s.pushSubTypes(ent.Subspace().ExpandSubTypes())
}

// entityForStruct if 't' is a schema object for a struct type return
// the schema entity for that struct.
func entityForStruct(t binary.SubspaceType) *binary.Entity {
	if s, ok := t.(*schema.Struct); ok {
		return s.Entity
	} else if r, ok := t.(*repeat); ok {
		return entityForStruct(r.replaces())
	}
	return nil
}

func (s *substack) pushRepeatIfNeeded(t binary.SubspaceType) binary.SubspaceType {
	if r, ok := t.(*repeat); ok {
		s.pushSubTypes(r.repetition())
		return r.replaces()
	}
	return t
}

func (s *substack) pushExpectStruct(t binary.SubspaceType) *binary.Entity {
	entity := entityForStruct(t)
	if entity == nil {
		return nil
	}
	s.pushSubTypes(t.Subspace().ExpandSubTypes())
	return entity
}

// pushSubTypes pushes the sub-types needed to decode a value of type 't'.
func (s *substack) pushSubTypes(t binary.TypeList) {
	for i := len(t) - 1; i >= 0; i-- {
		s.stack = append(s.stack, t[i])
	}
}

func (s *substack) pushRepeat(count uint32, repeats binary.SubspaceType) {
	if count > 1 {
		repeater := &repeat{count: count - 1, repeats: repeats}
		s.stack = append(s.stack, repeater)
	}
}

// pushCount, pops the top type from the stack and pushes any sub-types
// of that type count times. This is used to decode a collection of size
// count. If the top item on the stack is not a collection (slice, array, map)
// then an error is returned.
func (s *substack) pushCount(count uint32) error {
	t, err := s.popType()
	if err != nil {
		return err
	}
	if !t.HasSubspace() {
		return fmt.Errorf(
			"Decoding counted collection, found non-counted type %s", t)
		return nil
	}
	sub := t.Subspace()
	if !sub.Counted {
		return fmt.Errorf(
			"Decoding counted collection, found non-counted type %s", t)
	}
	if count == 0 {
		// empty collection
		return nil
	}
	subTypes := sub.ExpandSubTypes()
	if len(subTypes) == 0 {
		// non-empty collection, but with no interesting sub-types.
		return nil
	}
	s.pushRepeat(count, t)
	s.pushSubTypes(subTypes)
	return nil
}

// popType pops the type which is on the top of the stack. An error
// is returned if the stack is empty.
func (s *substack) popType() (binary.SubspaceType, error) {
	if len(s.stack) == 0 {
		return nil, fmt.Errorf("Pop on empty subtype Entity stack")
	}
	head := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return s.pushRepeatIfNeeded(head), nil
}
