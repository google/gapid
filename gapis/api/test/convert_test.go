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

package test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

func CreateTestExtra(a arena.Arena) TestExtra {
	o := NewTestObjectʳ(a, 42)
	m := NewU32ːTestObjectᵐ(a).
		Add(4, NewTestObject(a, 40)).
		Add(5, NewTestObject(a, 50))
	cycle := NewTestListʳ(a,
		1, // value
		NewTestListʳ(a, // next
			2,            // value
			NilTestListʳ, // next
		),
	)
	cycle.Next().SetNext(cycle)
	return NewTestExtra(a,
		NewU8ˢ(a, // Data
			0x1000,           // root
			0x1000,           // base
			42,               // size
			42,               // count
			memory.PoolID(1), // pool
		),
		NewTestObject(a, 10), // Object
		NewTestObjectː2ᵃ(a, // ObjectArray
			NewTestObject(a, 20),
			NewTestObject(a, 30),
		),
		o,              // RefObject
		o,              // RefObjectAlias
		NilTestObjectʳ, // NilRefObject
		m,              // Entries
		m,              // EntriesAlias
		NewU32ːTestObjectᵐ(a), // NilMap
		NewU32ːTestObjectʳᵐ(a). // RefEntries
					Add(0, o).
					Add(6, NewTestObjectʳ(a, 60)).
					Add(7, NewTestObjectʳ(a, 70)).
					Add(9, NilTestObjectʳ),
		NewStringːu32ᵐ(a). // Strings
					Add("one", 1).
					Add("two", 2).
					Add("three", 3),
		NewU32ːboolᵐ(a). // BoolMap
					Add(0, false).
					Add(1, true),
		NewTestListʳ(a, // LinkedList
			1, // value
			NewTestListʳ(a, // next
				2,            // value
				NilTestListʳ, // next
			),
		),
		cycle, // Cycle
		NewU32ːNestedRefʳᵐ(a). // NestedRefs
					Add(6, NewNestedRefʳ(a, o)).
					Add(7, NewNestedRefʳ(a, o)).
					Add(8, NewNestedRefʳ(a, NilTestObjectʳ)).
					Add(9, NilNestedRefʳ),
	)
}

func TestReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)

	a := arena.New()
	defer a.Dispose()

	ctx = arena.Put(ctx, a)
	extra := CreateTestExtra(a)

	// extra -> protoA -> decoded -> protoB

	protoA, err := protoconv.ToProto(ctx, extra)
	if !assert.For("ToProtoA").ThatError(err).Succeeded() {
		return
	}

	decodedObj, err := protoconv.ToObject(ctx, protoA)
	if !assert.For("ToObject").ThatError(err).Succeeded() {
		return
	}

	decoded := decodedObj.(TestExtra)

	assert.For("Object ref").That(decoded.RefObjectAlias()).Equals(decoded.RefObject())
	assert.For("Map ref").That(decoded.EntriesAlias()).Equals(decoded.Entries())
	assert.For("NestedRefs[6]").That(decoded.NestedRefs().Get(6).Ref()).Equals(decoded.RefObject())
	assert.For("NestedRefs[7]").That(decoded.NestedRefs().Get(7).Ref()).Equals(decoded.RefObject())

	protoB, err := protoconv.ToProto(ctx, decoded)
	if !assert.For("ToProtoB").ThatError(err).Succeeded() {
		return
	}

	assert.For("Protos").TestDeepEqual(protoA, protoB)

	// Test that all decoded references see changes to their referenced objects.
	decoded.RefObject().SetValue(55)               // was 42
	decoded.Entries().Add(4, NewTestObject(a, 33)) // was 50
	assert.For("Object ref").That(decoded.RefObjectAlias()).Equals(decoded.RefObject())
	assert.For("Map ref").That(decoded.EntriesAlias()).Equals(decoded.Entries())
	assert.For("RefEntries").That(decoded.RefEntries().Get(0)).Equals(decoded.RefObject())
	assert.For("NestedRefs[6]").That(decoded.NestedRefs().Get(6).Ref()).Equals(decoded.RefObject())
	assert.For("NestedRefs[7]").That(decoded.NestedRefs().Get(7).Ref()).Equals(decoded.RefObject())
}

func TestEquals(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)

	a := arena.New()
	defer a.Dispose()

	ctx = arena.Put(ctx, a)
	extra := CreateTestExtra(a)

	assert.For("equals").That(extra.Equals(extra)).Equals(true)
}

func TestCloneReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)

	a := arena.New()
	defer a.Dispose()

	ctx = arena.Put(ctx, a)
	extra := CreateTestExtra(a)

	cloned := extra.Clone(a, api.CloneContext{})
	assert.For("Data").That(cloned.Data()).Equals(extra.Data())
	assert.For("Object").That(cloned.Object().Value()).Equals(extra.Object().Value())
	assert.For("ObjectArray").That(cloned.ObjectArray().Equals(extra.ObjectArray())).Equals(true)
	assert.For("RefObject").That(cloned.RefObject().Value()).Equals(extra.RefObject().Value())
	assert.For("RefObjectAlias").That(cloned.RefObject().Value()).Equals(extra.RefObjectAlias().Value())
	assert.For("NilRefObject").That(cloned.NilRefObject().IsNil()).Equals(true)
	cloned.Entries().compare(assert.For("Entries"), extra.Entries())
	cloned.EntriesAlias().compare(assert.For("EntriesAlias"), extra.EntriesAlias())
	cloned.NilMap().compare(assert.For("NilMap"), extra.NilMap())
	cloned.Strings().compare(assert.For("Strings"), extra.Strings())
	cloned.BoolMap().compare(assert.For("BoolMap"), extra.BoolMap())
	// RefEntries
	assert.For("RefEntries.Len").That(cloned.RefEntries().Len()).Equals(extra.RefEntries().Len())
	for _, k := range extra.RefEntries().Keys() {
		assert.For("RefEntries[%d]", k).That(cloned.RefEntries().Contains(k)).Equals(true)
		e, a := extra.RefEntries().Get(k), cloned.RefEntries().Get(k)
		if e.IsNil() {
			assert.For("RefEntries[%d]", k).That(a.IsNil()).Equals(true)
		} else {
			assert.For("RefEntries[%d]", k).That(a.Value()).Equals(e.Value())
		}
	}
	// LinkedList
	for i, e, a := 0, extra.LinkedList(), cloned.LinkedList(); !e.IsNil(); i++ {
		assert.For("LinkedList[%d]", i).That(a.IsNil()).Equals(false)
		assert.For("LinkedList[%d]", i).That(a.Value()).Equals(e.Value())
		assert.For("LinkedList[%d]", i).That(a.Next().IsNil()).Equals(e.Next().IsNil())
		e, a = e.Next(), a.Next()
	}
	// Cycle
	assert.For("Cycle[0]").That(cloned.Cycle().IsNil()).Equals(false)
	assert.For("Cycle[0]").That(cloned.Cycle().Value()).Equals(uint32(1))
	assert.For("Cycle[1]").That(cloned.Cycle().Next().IsNil()).Equals(false)
	assert.For("Cycle[1]").That(cloned.Cycle().Next().Value()).Equals(uint32(2))
	assert.For("Cycle").That(cloned.Cycle().Next().Next()).Equals(cloned.Cycle())
	// NestedRefs
	assert.For("NestedRefs.Len").That(cloned.NestedRefs().Len()).Equals(extra.NestedRefs().Len())
	for _, k := range extra.NestedRefs().Keys() {
		assert.For("NestedRefs[%d]", k).That(cloned.NestedRefs().Contains(k)).Equals(true)
		e, a := extra.NestedRefs().Get(k), cloned.NestedRefs().Get(k)
		assert.For("NestedRefs[%d]", k).That(a.IsNil()).Equals(e.IsNil())
		if !e.IsNil() {
			assert.For("NestedRefs[%d].ref", k).That(a.Ref().IsNil()).Equals(e.Ref().IsNil())
			if !e.Ref().IsNil() {
				assert.For("NestedRefs[%d].ref", k).That(a.Ref().Value()).Equals(e.Ref().Value())
			}
		}
	}

	// Test that all cloned references see changes to their referenced objects.
	cloned.RefObject().SetValue(55)               // was 42
	cloned.Entries().Add(4, NewTestObject(a, 33)) // was 50
	assert.For("Object ref").That(cloned.RefObjectAlias()).Equals(cloned.RefObject())
	assert.For("Map ref").That(cloned.EntriesAlias()).Equals(cloned.Entries())
	assert.For("RefEntries").That(cloned.RefEntries().Get(0)).Equals(cloned.RefObject())
	assert.For("NestedRefs[6]").That(cloned.NestedRefs().Get(6).Ref()).Equals(cloned.RefObject())
	assert.For("NestedRefs[7]").That(cloned.NestedRefs().Get(7).Ref()).Equals(cloned.RefObject())
}

func (actual U32ːTestObjectᵐ) compare(a *assert.Assertion, expected U32ːTestObjectᵐ) bool {
	a.Compare(*actual.Map, "==", *expected.Map)
	if !a.Test(actual.Len() == expected.Len()) {
		return false
	}

	for _, k := range expected.Keys() {
		if !a.Test(actual.Contains(k) && actual.Get(k).Equals(expected.Get(k))) {
			return false
		}
	}

	return true
}

func (actual Stringːu32ᵐ) compare(a *assert.Assertion, expected Stringːu32ᵐ) bool {
	a.Compare(*actual.Map, "==", *expected.Map)
	if !a.Test(actual.Len() == expected.Len()) {
		return false
	}

	for _, k := range expected.Keys() {
		if !a.Test(actual.Contains(k) && actual.Get(k) == expected.Get(k)) {
			return false
		}
	}

	return true
}

func (actual U32ːboolᵐ) compare(a *assert.Assertion, expected U32ːboolᵐ) bool {
	a.Compare(*actual.Map, "==", *expected.Map)
	if !a.Test(actual.Len() == expected.Len()) {
		return false
	}

	for _, k := range expected.Keys() {
		if !a.Test(actual.Contains(k) && actual.Get(k) == expected.Get(k)) {
			return false
		}
	}

	return true
}
