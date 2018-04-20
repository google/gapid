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
	"github.com/google/gapid/gapis/memory"
)

func TestReferences(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	o := NewTestObjectʳ(42)
	m := NewU32ːTestObjectᵐ().
		Add(4, NewTestObject(40)).
		Add(5, NewTestObject(50))
	cycle := NewTestListʳ(
		1, // value
		NewTestListʳ( // next
			2,            // value
			NilTestListʳ, // next
		),
	)
	cycle.Next().SetNext(cycle)
	extra := NewTestExtra(
		NewU8ˢ( // Data
			0x1000,           // root
			0x1000,           // base
			42,               // size
			42,               // count
			memory.PoolID(1), // pool
		),
		NewTestObject(10), // Object
		NewTestObjectː2ᵃ( // ObjectArray
			NewTestObject(20),
			NewTestObject(30),
		),
		o,              // RefObject
		o,              // RefObjectAlias
		NilTestObjectʳ, // NilRefObject
		m,              // Entries
		m,              // EntriesAlias
		NewU32ːTestObjectᵐ(), // NilMap
		NewU32ːTestObjectʳᵐ(). // RefEntries
					Add(0, o).
					Add(6, NewTestObjectʳ(60)).
					Add(7, NewTestObjectʳ(70)).
					Add(9, NilTestObjectʳ),
		NewStringːu32ᵐ(). // Strings
					Add("one", 1).
					Add("two", 2).
					Add("three", 3),
		NewU32ːboolᵐ(). // BoolMap
				Add(0, false).
				Add(1, true),
		NewTestListʳ( // LinkedList
			1, // value
			NewTestListʳ( // next
				2,            // value
				NilTestListʳ, // next
			),
		),
		cycle, // Cycle
	)

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

	assert.For("Object ref").That(decoded.RefObject()).Equals(decoded.RefObjectAlias())
	assert.For("Map ref").That(decoded.Entries()).Equals(decoded.EntriesAlias())

	protoB, err := protoconv.ToProto(ctx, decoded)
	if !assert.For("ToProtoB").ThatError(err).Succeeded() {
		return
	}

	assert.For("Protos").TestDeepEqual(protoA, protoB)
}
