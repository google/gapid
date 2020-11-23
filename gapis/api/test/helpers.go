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
	"github.com/google/gapid/core/data/compare"
	"github.com/google/gapid/gapis/memory"
)

// Cmds holds a number of prebuilt example commands that can be used for tests.
var Cmds struct {
	A *CmdTypeMix
	B *CmdTypeMix

	// IgnoreArena is a custom compare rule for excluding the arena in command
	// tests.
	IgnoreArena compare.Custom
}

func init() {
	cb := CommandBuilder{}
	Cmds.A = cb.CmdTypeMix(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, Voidᵖ(0x12345678), 100)
	Cmds.B = cb.CmdTypeMix(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, false, Voidᵖ(0xabcdef9), 200)

	Cmds.IgnoreArena.Register(func(c compare.Comparator, reference, value *CmdTypeMix) {
		c.With(c.Path.Member("Thread", reference, value)).Compare(reference.Thread(), value.Thread())
		c.With(c.Path.Member("CmdName", reference, value)).Compare(reference.CmdName(), value.CmdName())
		c.With(c.Path.Member("Extras", reference, value)).Compare(reference.Extras(), value.Extras())

		c.With(c.Path.Member("U8", reference, value)).Compare(reference.U8(), value.U8())
		c.With(c.Path.Member("S8", reference, value)).Compare(reference.S8(), value.S8())
		c.With(c.Path.Member("U16", reference, value)).Compare(reference.U16(), value.U16())
		c.With(c.Path.Member("S16", reference, value)).Compare(reference.S16(), value.S16())
		c.With(c.Path.Member("U32", reference, value)).Compare(reference.U32(), value.U32())
		c.With(c.Path.Member("S32", reference, value)).Compare(reference.S32(), value.S32())
		c.With(c.Path.Member("U64", reference, value)).Compare(reference.U64(), value.U64())
		c.With(c.Path.Member("S64", reference, value)).Compare(reference.S64(), value.S64())
		c.With(c.Path.Member("F32", reference, value)).Compare(reference.F32(), value.F32())
		c.With(c.Path.Member("F64", reference, value)).Compare(reference.F64(), value.F64())
		c.With(c.Path.Member("Bool", reference, value)).Compare(reference.Bool(), value.Bool())
		c.With(c.Path.Member("Ptr", reference, value)).Compare(reference.Ptr(), value.Ptr())
	})
}

// BuildComplex returns a Complex populated with data.
func BuildComplex() Complex {
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
	return NewComplex(
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
		o,                    // RefObject
		o,                    // RefObjectAlias
		NilTestObjectʳ,       // NilRefObject
		m,                    // Entries
		m,                    // EntriesAlias
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
		NewU32ːNestedRefʳᵐ(). // NestedRefs
					Add(6, NewNestedRefʳ(o)).
					Add(7, NewNestedRefʳ(o)).
					Add(8, NewNestedRefʳ(NilTestObjectʳ)).
					Add(9, NilNestedRefʳ),
	)
}
