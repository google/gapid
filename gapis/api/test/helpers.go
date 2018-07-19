// Copyright (C) 2018 Google Inc.
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
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/memory"
)

// Cmds holds a number of prebuilt example commands that can be used for tests.
var Cmds struct {
	A *CmdTypeMix
	B *CmdTypeMix
}

func init() {
	arena := arena.New()
	cb := CommandBuilder{Arena: arena}
	Cmds.A = cb.CmdTypeMix(10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, Voidᵖ(0x12345678), 100)
	Cmds.B = cb.CmdTypeMix(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, false, Voidᵖ(0xabcdef9), 200)
}

// BuildComplex returns a Complex populated with data.
func BuildComplex(a arena.Arena) Complex {
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
	return NewComplex(a,
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
