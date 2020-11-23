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
	"github.com/google/gapid/gapis/api"
)

func TestReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	complex := BuildComplex()

	// complex -> protoA -> decoded -> protoB

	protoA, err := protoconv.ToProto(ctx, complex)
	if !assert.For("ToProtoA").ThatError(err).Succeeded() {
		return
	}

	decodedObj, err := protoconv.ToObject(ctx, protoA)
	if !assert.For("ToObject").ThatError(err).Succeeded() {
		return
	}

	decoded := decodedObj.(Complex)

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
	decoded.RefObject().SetValue(55)            // was 42
	decoded.Entries().Add(4, NewTestObject(33)) // was 50
	assert.For("Object ref").That(decoded.RefObjectAlias()).Equals(decoded.RefObject())
	assert.For("Map ref").That(decoded.EntriesAlias()).Equals(decoded.Entries())
	assert.For("RefEntries").That(decoded.RefEntries().Get(0)).Equals(decoded.RefObject())
	assert.For("NestedRefs[6]").That(decoded.NestedRefs().Get(6).Ref()).Equals(decoded.RefObject())
	assert.For("NestedRefs[7]").That(decoded.NestedRefs().Get(7).Ref()).Equals(decoded.RefObject())
}

func TestEquals(t *testing.T) {
	ctx := log.Testing(t)
	complex := BuildComplex()

	check(ctx, complex, complex, "equals")
}

func TestCloneReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	complex := BuildComplex()

	cloned := complex.Clone(api.CloneContext{})
	check(ctx, cloned.Data(), complex.Data(), "Data")
	check(ctx, cloned.Object(), complex.Object(), "Object")
	check(ctx, cloned.ObjectArray(), complex.ObjectArray(), "ObjectArray")
	check(ctx, cloned.RefObject().Value(), complex.RefObject().Value(), "RefObject")
	check(ctx, cloned.RefObject().Value(), complex.RefObjectAlias().Value(), "RefObjectAlias")
	check(ctx, cloned.NilRefObject().IsNil(), true, "NilRefObject")
	check(ctx, cloned.Entries(), complex.Entries(), "Entries")
	check(ctx, cloned.EntriesAlias(), complex.EntriesAlias(), "EntriesAlias")
	check(ctx, cloned.NilMap(), complex.NilMap(), "NilMap")
	check(ctx, cloned.Strings(), complex.Strings(), "Strings")
	check(ctx, cloned.BoolMap(), complex.BoolMap(), "BoolMap")
	// RefEntries
	assert.For("RefEntries.Len").That(cloned.RefEntries().Len()).Equals(complex.RefEntries().Len())
	for _, k := range complex.RefEntries().Keys() {
		check(ctx, cloned.RefEntries().Contains(k), true, "RefEntries[%d]", k)
		e, a := complex.RefEntries().Get(k), cloned.RefEntries().Get(k)
		if e.IsNil() {
			check(ctx, a.IsNil(), true, "RefEntries[%d]", k)
		} else {
			check(ctx, a.Value(), e.Value(), "RefEntries[%d]", k)
		}
	}
	// LinkedList
	for i, e, a := 0, complex.LinkedList(), cloned.LinkedList(); !e.IsNil(); i++ {
		check(ctx, a.IsNil(), false, "LinkedList[%d]", i)
		check(ctx, a.Value(), e.Value(), "LinkedList[%d]", i)
		check(ctx, a.Next().IsNil(), e.Next().IsNil(), "LinkedList[%d]", i)
		e, a = e.Next(), a.Next()
	}
	// Cycle
	check(ctx, cloned.Cycle().IsNil(), false, "Cycle[0]")
	check(ctx, cloned.Cycle().Value(), uint32(1), "Cycle[0]")
	check(ctx, cloned.Cycle().Next().IsNil(), false, "Cycle[1]")
	check(ctx, cloned.Cycle().Next().Value(), uint32(2), "Cycle[1]")
	check(ctx, cloned.Cycle().Next().Next(), cloned.Cycle(), "Cycle")
	// NestedRefs
	check(ctx, cloned.NestedRefs().Len(), complex.NestedRefs().Len(), "NestedRefs.Len")
	for _, k := range complex.NestedRefs().Keys() {
		check(ctx, cloned.NestedRefs().Contains(k), true, "NestedRefs[%d]", k)
		e, a := complex.NestedRefs().Get(k), cloned.NestedRefs().Get(k)
		check(ctx, a.IsNil(), e.IsNil(), "NestedRefs[%d]", k)
		if !e.IsNil() {
			check(ctx, a.Ref().IsNil(), e.Ref().IsNil(), "NestedRefs[%d].ref", k)
			if !e.Ref().IsNil() {
				check(ctx, a.Ref().Value(), e.Ref().Value(), "NestedRefs[%d].ref", k)
			}
		}
	}

	// Test that all cloned references see changes to their referenced objects.
	cloned.RefObject().SetValue(55)            // was 42
	cloned.Entries().Add(4, NewTestObject(33)) // was 50
	check(ctx, cloned.RefObjectAlias(), cloned.RefObject(), "Object ref")
	check(ctx, cloned.EntriesAlias(), cloned.Entries(), "Map ref")
	check(ctx, cloned.RefEntries().Get(0), cloned.RefObject(), "RefEntries")
	check(ctx, cloned.NestedRefs().Get(6).Ref(), cloned.RefObject(), "NestedRefs[6]")
	check(ctx, cloned.NestedRefs().Get(7).Ref(), cloned.RefObject(), "NestedRefs[7]")
}

func check(ctx context.Context, got, expected interface{}, name string, fmt ...interface{}) bool {
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
			if !check(ctx, g, e, "%v got[%v] == expected[%v]", name, k, k) {
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

	type ieq interface {
		Equals(generic.TO) bool
	}
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
