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
	"context"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/data/generic"
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

	a := arena.New()
	defer a.Dispose()

	ctx = arena.Put(ctx, a)
	extra := CreateTestExtra(a)

	compare(ctx, extra, extra, "equals")
}

func TestCloneReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)

	a := arena.New()
	defer a.Dispose()

	ctx = arena.Put(ctx, a)
	extra := CreateTestExtra(a)

	cloned := extra.Clone(a, api.CloneContext{})
	compare(ctx, cloned.Data(), extra.Data(), "Data")
	compare(ctx, cloned.Object(), extra.Object(), "Object")
	compare(ctx, cloned.ObjectArray(), extra.ObjectArray(), "ObjectArray")
	compare(ctx, cloned.RefObject().Value(), extra.RefObject().Value(), "RefObject")
	compare(ctx, cloned.RefObject().Value(), extra.RefObjectAlias().Value(), "RefObjectAlias")
	compare(ctx, cloned.NilRefObject().IsNil(), true, "NilRefObject")
	compare(ctx, cloned.Entries(), extra.Entries(), "Entries")
	compare(ctx, cloned.EntriesAlias(), extra.EntriesAlias(), "EntriesAlias")
	compare(ctx, cloned.NilMap(), extra.NilMap(), "NilMap")
	compare(ctx, cloned.Strings(), extra.Strings(), "Strings")
	compare(ctx, cloned.BoolMap(), extra.BoolMap(), "BoolMap")
	// RefEntries
	assert.For("RefEntries.Len").That(cloned.RefEntries().Len()).Equals(extra.RefEntries().Len())
	for _, k := range extra.RefEntries().Keys() {
		compare(ctx, cloned.RefEntries().Contains(k), true, "RefEntries[%d]", k)
		e, a := extra.RefEntries().Get(k), cloned.RefEntries().Get(k)
		if e.IsNil() {
			compare(ctx, a.IsNil(), true, "RefEntries[%d]", k)
		} else {
			compare(ctx, a.Value(), e.Value(), "RefEntries[%d]", k)
		}
	}
	// LinkedList
	for i, e, a := 0, extra.LinkedList(), cloned.LinkedList(); !e.IsNil(); i++ {
		compare(ctx, a.IsNil(), false, "LinkedList[%d]", i)
		compare(ctx, a.Value(), e.Value(), "LinkedList[%d]", i)
		compare(ctx, a.Next().IsNil(), e.Next().IsNil(), "LinkedList[%d]", i)
		e, a = e.Next(), a.Next()
	}
	// Cycle
	compare(ctx, cloned.Cycle().IsNil(), false, "Cycle[0]")
	compare(ctx, cloned.Cycle().Value(), uint32(1), "Cycle[0]")
	compare(ctx, cloned.Cycle().Next().IsNil(), false, "Cycle[1]")
	compare(ctx, cloned.Cycle().Next().Value(), uint32(2), "Cycle[1]")
	compare(ctx, cloned.Cycle().Next().Next(), cloned.Cycle(), "Cycle")
	// NestedRefs
	compare(ctx, cloned.NestedRefs().Len(), extra.NestedRefs().Len(), "NestedRefs.Len")
	for _, k := range extra.NestedRefs().Keys() {
		compare(ctx, cloned.NestedRefs().Contains(k), true, "NestedRefs[%d]", k)
		e, a := extra.NestedRefs().Get(k), cloned.NestedRefs().Get(k)
		compare(ctx, a.IsNil(), e.IsNil(), "NestedRefs[%d]", k)
		if !e.IsNil() {
			compare(ctx, a.Ref().IsNil(), e.Ref().IsNil(), "NestedRefs[%d].ref", k)
			if !e.Ref().IsNil() {
				compare(ctx, a.Ref().Value(), e.Ref().Value(), "NestedRefs[%d].ref", k)
			}
		}
	}

	// Test that all cloned references see changes to their referenced objects.
	cloned.RefObject().SetValue(55)               // was 42
	cloned.Entries().Add(4, NewTestObject(a, 33)) // was 50
	compare(ctx, cloned.RefObjectAlias(), cloned.RefObject(), "Object ref")
	compare(ctx, cloned.EntriesAlias(), cloned.Entries(), "Map ref")
	compare(ctx, cloned.RefEntries().Get(0), cloned.RefObject(), "RefEntries")
	compare(ctx, cloned.NestedRefs().Get(6).Ref(), cloned.RefObject(), "NestedRefs[6]")
	compare(ctx, cloned.NestedRefs().Get(7).Ref(), cloned.RefObject(), "NestedRefs[7]")
}

func compare(ctx context.Context, got, expected interface{}, name string, fmt ...interface{}) bool {
	g, e := got, expected

	if g, e := dictionary.From(g), dictionary.From(e); g != nil && e != nil {
		// Comparing dictionaries
		if !assert.For(ctx, "%v.Len", name).That(g.Len()).Equals(e.Len()) {
			return false
		}

		for _, k := range e.Keys() {
			e := e.Get(k)
			g, ok := g.Lookup(k)
			if !assert.For(ctx, "%v.Contains(%v)", name, k).That(ok).Equals(true) {
				return false
			}
			if !compare(ctx, g, e, "%v got[%v] == expected[%v]", name, k, k) {
				return false
			}
		}

		for _, k := range g.Keys() {
			_, ok := e.Lookup(k)
			if !assert.For(ctx, "%v.Missing(%v)", name, k).That(ok).Equals(true) {
				return false
			}
		}

		return true
	}

	type ieq interface{ Equals(generic.TO) bool }
	ieqTy := reflect.TypeOf((*ieq)(nil)).Elem()
	gTy, eTy := reflect.TypeOf(g), reflect.TypeOf(e)
	if m := generic.Implements(reflect.TypeOf(g), ieqTy); m.Ok() && gTy == eTy {
		// Comparing using Equals() method
		f := reflect.ValueOf(g).MethodByName("Equals")
		ok := f.Call([]reflect.Value{reflect.ValueOf(e)})[0].Interface().(bool)
		return assert.For(ctx, name, fmt...).Compare(g, "==", e).Test(ok)
	}

	// Comparing using regular assert comparison
	return assert.For(ctx, name, fmt...).That(g).Equals(e)
}
