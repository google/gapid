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

package generic_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/generic"
	"github.com/google/gapid/core/log"
)

type (
	Any = generic.Any
	T1  = generic.T1
	T2  = generic.T2
	T3  = generic.T3
	T4  = generic.T4
	TO  = generic.TO
	TP  = generic.TP
)

type O struct{}

func (O) M1()                                                 {}
func (O) M2(int, float32)                                     {}
func (O) M3(int, string)                                      {}
func (O) M4() (string, int)                                   { return "", 0 }
func (O) M5() (float32, int)                                  { return 0, 0 }
func (O) M7(int, float32, []int, [3]float32, map[int]float32) {}
func (O) M8(O)                                                {}
func (O) M9(*O)                                               {}

func TestImplements(t *testing.T) {
	ctx := log.Testing(t)

	E := func(s ...string) []error {
		out := make([]error, len(s))
		for i, m := range s {
			out[i] = fmt.Errorf("%v", m)
		}
		return out
	}

	for _, test := range []struct {
		name     string
		iface    reflect.Type
		expected []error
	}{
		{"empty", reflect.TypeOf((*interface{})(nil)).Elem(), E()},
		{"M1", reflect.TypeOf((*interface{ M1() })(nil)).Elem(), E()},
		{
			"missing-single",
			reflect.TypeOf((*interface{ Missing() })(nil)).Elem(),
			E("'generic_test.O' does not implement method 'Missing'"),
		}, {
			"missing-many",
			reflect.TypeOf((*interface {
				M1()
				MissingA()
				M3(int, string)
				MissingB()
				M4() (string, int)
				MissingC()
				M5() (float32, int)
			})(nil)).Elem(),
			E("'generic_test.O' does not implement method 'MissingA'",
				"'generic_test.O' does not implement method 'MissingB'",
				"'generic_test.O' does not implement method 'MissingC'"),
		}, {
			"match-all",
			reflect.TypeOf((*interface {
				M1()
				M2(int, float32)
				M3(int, string)
				M4() (string, int)
				M5() (float32, int)
			})(nil)).Elem(),
			E(),
		}, {
			"missing-params",
			reflect.TypeOf((*interface{ M1(int) })(nil)).Elem(),
			E("method 'M1' has too few parameters\nInterface:   func(int)\nImplementor: func()"),
		}, {
			"extra-params",
			reflect.TypeOf((*interface{ M2() })(nil)).Elem(),
			E("method 'M2' has too many parameters\nInterface:   func()\nImplementor: func(int, float32)"),
		}, {
			"missing-return",
			reflect.TypeOf((*interface{ M1() int })(nil)).Elem(),
			E("method 'M1' has too few return values\nInterface:   func() int\nImplementor: func()"),
		}, {
			"extra-return",
			reflect.TypeOf((*interface{ M4() })(nil)).Elem(),
			E(),
		}, {
			"any",
			reflect.TypeOf((*interface{ M2(Any, Any) })(nil)).Elem(),
			E(),
		}, {
			"one-use-generics",
			reflect.TypeOf((*interface{ M2(T1, T2) })(nil)).Elem(),
			E(),
		}, {
			"reuse-generics",
			reflect.TypeOf((*interface { // T1 = int, T2 = float32
				M2(T1, T2)
				M3(T1, string)
				M4() (string, T1)
				M5() (T2, T1)
				M7(T1, T2, []T1, [3]T2, map[T1]T2)
			})(nil)).Elem(),
			E(),
		}, {
			"conflicting-generics",
			reflect.TypeOf((*interface {
				M2(T1, T1)
				M3(T1, T1)
				M4() (T2, T2)
				M5() (T2, T2)
			})(nil)).Elem(),
			E("mixed use of generic type 'generic.T1'. First used as 'int', now used as 'float32'\nInterface:   func(generic.T1, generic.T1)\nImplementor: func(int, float32)",
				"mixed use of generic type 'generic.T2'. First used as 'string', now used as 'int'\nInterface:   func() (generic.T2, generic.T2)\nImplementor: func() (string, int)"),
		}, {
			"TC-TP",
			reflect.TypeOf((*interface { // TO = O, TP = *O
				M8(TO)
				M9(TP)
			})(nil)).Elem(),
			E(),
		}, {
			"TC-TP",
			reflect.TypeOf((*interface { // TO = O, TP = *O
				M8(TP)
				M9(TO)
			})(nil)).Elem(),
			E("mixed use of generic type 'generic.TP'. First used as '*generic_test.O', now used as 'generic_test.O'\nInterface:   func(generic.TP)\nImplementor: func(generic_test.O)",
				"mixed use of generic type 'generic.TO'. First used as 'generic_test.O', now used as '*generic_test.O'\nInterface:   func(generic.TO)\nImplementor: func(*generic_test.O)"),
		},
	} {
		got := generic.Implements(reflect.TypeOf(O{}), test.iface)
		assert.For(ctx, test.name).ThatSlice(got.Errors).DeepEquals(test.expected)
	}
}

func TestCheckSigs(t *testing.T) {
	ctx := log.Testing(t)

	E := func(s ...string) []error {
		out := make([]error, len(s))
		for i, m := range s {
			out[i] = fmt.Errorf("%v", m)
		}
		return out
	}

	for _, test := range []struct {
		name     string
		sigs     []generic.Sig
		expected []error
	}{
		{"empty", []generic.Sig{}, E()},
		{"func()", []generic.Sig{{"func()", func() {}, func() {}}}, E()},
		{"func(int)int", []generic.Sig{{"func(int)int", func(int) int { return 0 }, func(int) int { return 0 }}}, E()},
		{"func(T1)T1", []generic.Sig{{"func(T1)T1", func(T1) T1 { return T1{} }, func(int) int { return 0 }}}, E()},
		{"func(T1)T2", []generic.Sig{{"func(T1)T2", func(T1) T2 { return T2{} }, func(int) int { return 0 }}}, E()},
		{"func(T1)T1 fail", []generic.Sig{{"func(T1)T1", func(T1) T1 { return T1{} }, func(int) bool { return false }}},
			E("mixed use of generic type 'generic.T1'. First used as 'int', now used as 'bool'\nInterface:   func(generic.T1) generic.T1\nImplementor: func() bool")},
		{"func(T1)T2, func(T2)T1", []generic.Sig{
			{"func(T1)T2", func(T1) T2 { return T2{} }, func(int) bool { return false }},
			{"func(T2)T1", func(T2) T1 { return T1{} }, func(bool) int { return 0 }},
		}, E()},
		{"func(T1)T2, func(T2)T1 fail", []generic.Sig{
			{"func(T1)T2", func(T1) T2 { return T2{} }, func(int) bool { return false }},
			{"func(T2)T1", func(T2) T1 { return T1{} }, func(bool) bool { return false }},
		}, E("mixed use of generic type 'generic.T1'. First used as 'int', now used as 'bool'\nInterface:   func(generic.T2) generic.T1\nImplementor: func() bool")},
	} {
		got := generic.CheckSigs(test.sigs...)
		assert.For(ctx, test.name).ThatSlice(got.Errors).DeepEquals(test.expected)
	}
}
