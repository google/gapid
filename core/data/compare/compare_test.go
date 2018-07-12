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

package compare_test

import (
	"reflect"
	"testing"

	"fmt"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/compare"
)

var (
	funcNil1   func()
	funcNil2   func()
	funcNotNil = func() { funcNil1() } // Not nil.

	leafBase      = &Object{0, nil}
	leafEqual     = &Object{0, nil}
	leafDifferent = &Object{2, nil}

	objectBase       = &Object{0, []interface{}{leafBase, leafBase}}
	objectEqual1     = &Object{0, []interface{}{leafBase, leafBase}}
	objectEqual2     = &Object{0, []interface{}{leafEqual, leafBase}}
	objectDifferent  = &Object{0, []interface{}{leafDifferent, leafBase}}
	objectInverse    = &Object{0, []interface{}{leafBase, leafEqual}}
	objectShort      = &Object{0, []interface{}{leafBase}}
	objectInvalid1   = &Object{0, []interface{}{nil}}
	objectInvalid2   = &Object{0, []interface{}{nil}}
	objectWithInt    = &Object{0, []interface{}{0}}
	objectWithString = &Object{0, []interface{}{""}}

	root   = compare.Path{}
	noDiff = []compare.Path{}

	valueBase      = Value(1)
	valueEqual     = Value(1)
	valueDifferent = Value(2)
	alternate      = Alternate(1)

	sliceBase      = []int{1, 2, 3}
	sliceEqual     = []int{1, 2, 3}
	sliceLonger    = []int{1, 2, 3, 4}
	sliceDifferent = []int{1, 2, 4}

	arrayBase      = [3]int{1, 2, 3}
	arrayEqual     = [3]int{1, 2, 3}
	arrayDifferent = [3]int{1, 2, 4}

	structBase           = Struct{1, 0.5}
	structEqual          = Struct{1, 0.5}
	structDifferentInt   = Struct{2, 0.5}
	structDifferentFloat = Struct{1, 0.6}
	aliasBase            = Alias{1, 0.5}
	otherBase            = OtherStruct{1, 0.5}

	mapBase            = map[int]string{1: "one", 2: "two"}
	mapEqual           = map[int]string{1: "one", 2: "two"}
	mapDifferentKeys   = map[int]string{1: "one", 3: "three"}
	mapDifferentValues = map[int]string{1: "one", 2: "too"}
	mapDifferentLength = map[int]string{1: "one"}
	mapUint            = map[uint]string{1: "one", 2: "two"}

	hidden1 = Hide{1}
	hidden2 = Hide{2}

	compareTests = []struct {
		name string
		a, b interface{}
		diff []compare.Path
	}{
		// Equalities
		{"nil ==", nil, nil, noDiff},
		{"int ==", 1, 1, noDiff},
		{"int32 ==", int32(1), int32(1), noDiff},
		{"float64 ==", 0.5, 0.5, noDiff},
		{"float32 ==", float32(0.5), float32(0.5), noDiff},
		{"string ==", "hello", "hello", noDiff},
		{"slice ==", sliceBase, sliceEqual, noDiff},
		{"empty slice ==", []int{}, []int{}, noDiff},
		{"nill slice ==", []int(nil), []int(nil), noDiff},
		{"array ==", arrayBase, arrayEqual, noDiff},
		{"struct ==", structBase, structEqual, noDiff},
		{"error ==", error(nil), error(nil), noDiff},
		{"map ==", mapBase, mapEqual, noDiff},
		{"empty map ==", map[int]int{}, map[int]int{}, noDiff},
		{"nil map ==", map[int]int(nil), map[int]int(nil), noDiff},
		{"func ==", funcNil1, funcNil2, noDiff},
		{"leaf ==", leafBase, leafEqual, noDiff},
		{"object ==", objectBase, objectBase, noDiff},
		{"object pointer ==", objectBase, objectEqual1, noDiff},
		{"object value ==", objectBase, objectEqual2, noDiff},
		{"object inverse ==", objectEqual2, objectInverse, noDiff},
		{"hidden==", Hide{1}, Hide{1}, noDiff},
		{"pointer ==", &valueBase, &valueEqual, noDiff},
		{"invalid", objectInvalid1, objectInvalid2, noDiff},
		{"custom 1", Special(1), Special(1), noDiff},
		{"custom 3", Special(1), Special(3), noDiff},
		{"special struct custom 1", SpecialStruct{false, 1}, SpecialStruct{false, 1}, noDiff},
		{"special struct custom 3", SpecialStruct{false, 1}, SpecialStruct{false, 3}, noDiff},

		// Inequalities
		{"int !=", 1, 2, []compare.Path{root.Diff(1, 2)}},
		{"int32 !=", int32(1), int32(2), []compare.Path{root.Diff(int32(1), int32(2))}},
		{"float64 !=", 0.5, 0.6, []compare.Path{root.Diff(0.5, 0.6)}},
		{"float32 !=", float32(0.5), float32(0.6), []compare.Path{root.Diff(float32(0.5), float32(0.6))}},
		{"string !=", "hello", "hey", []compare.Path{root.Diff("hello", "hey")}},
		{"slice.longer", sliceBase, sliceLonger, []compare.Path{
			root.Length(sliceBase, sliceLonger).Diff(3, 4),
			root.Index(3, sliceBase, sliceLonger).Missing(compare.Missing, 4),
		}},
		{"slice.shorter", sliceLonger, sliceBase, []compare.Path{
			root.Index(3, sliceLonger, sliceBase).Missing(4, compare.Missing),
			root.Length(sliceLonger, sliceBase).Diff(4, 3),
		}},
		{"slice.value", sliceBase, sliceDifferent, []compare.Path{
			root.Index(2, sliceBase, sliceDifferent).Diff(3, 4),
		}},
		{"array !=", arrayBase, arrayDifferent, []compare.Path{
			root.Index(2, arrayBase, arrayDifferent).Diff(3, 4),
		}},
		{"struct.float !=", structBase, structDifferentFloat, []compare.Path{
			root.Member("Float", structBase, structDifferentFloat).Diff(float32(0.5), float32(0.6)),
		}},
		{"struct.int !=", structBase, structDifferentInt, []compare.Path{
			root.Member("Int", structBase, structDifferentInt).Diff(1, 2),
		}},
		{"map.keys !=", mapBase, mapDifferentKeys, []compare.Path{
			root.Entry(2, mapBase, mapDifferentKeys).Missing("two", compare.Missing),
			root.Entry(3, mapBase, mapDifferentKeys).Missing(compare.Missing, "three"),
		}},
		{"map.values !=", mapBase, mapDifferentValues, []compare.Path{
			root.Entry(2, mapBase, mapDifferentValues).Diff("two", "too"),
		}},
		{"len(map) 1 != 2", mapDifferentLength, mapBase, []compare.Path{
			root.Length(mapDifferentLength, mapBase).Diff(1, 2),
			root.Entry(2, mapDifferentLength, mapBase).Missing(compare.Missing, "two"),
		}},
		{"len(map) 2 != 1", mapBase, mapDifferentLength, []compare.Path{
			root.Length(mapBase, mapDifferentLength).Diff(2, 1),
			root.Entry(2, mapBase, mapDifferentLength).Missing("two", compare.Missing),
		}},
		{"nil != int", nil, 1, []compare.Path{
			root.Nil(nil, 1),
		}},
		{"int != nil", 1, nil, []compare.Path{
			root.Nil(1, nil),
		}},
		{"nil != func", funcNil1, funcNotNil, []compare.Path{{}}},
		{"func != func", funcNotNil, funcNotNil, []compare.Path{{}}},
		{"leaf !=", leafBase, leafDifferent, []compare.Path{
			root.Member("Value", leafBase, leafDifferent).Diff(0, 2),
		}},
		{"object 1 !=", objectBase, objectDifferent, []compare.Path{
			root.Member("Children", objectBase, objectDifferent).
				Index(0, objectBase.Children, objectDifferent.Children).
				Member("Value", objectBase.Children[0], objectDifferent.Children[0]).
				Diff(0, 2),
		}},
		{"object 2 !=", objectBase, objectShort, []compare.Path{
			root.Member("Children", objectBase, objectShort).
				Index(1, objectBase.Children, objectShort.Children).
				Missing(objectBase.Children[1], compare.Missing),
			root.Member("Children", objectBase, objectShort).
				Length(objectBase.Children, objectShort.Children).
				Diff(2, 1),
		}},
		{"invalid reference", objectInvalid1, objectShort, []compare.Path{
			root.Member("Children", objectInvalid1, objectShort).
				Index(0, objectInvalid1.Children, objectShort.Children).
				Nil(nil, leafBase),
		}},
		{"invalid value", objectShort, objectInvalid1, []compare.Path{
			root.Member("Children", objectShort, objectInvalid1).
				Index(0, objectShort.Children, objectInvalid1.Children).
				Nil(leafBase, nil),
		}},
		{"int aliases", valueBase, alternate, []compare.Path{
			root.Type(valueBase, alternate).Diff(reflect.TypeOf(valueBase), reflect.TypeOf(alternate)),
		}},
		{"empty slice != nil slice", []int{}, []int(nil), []compare.Path{
			root.Nil([]int{}, nil),
		}},
		{"empty map != nil map", map[int]int{}, map[int]int(nil), []compare.Path{
			root.Nil(map[int]int{}, nil),
		}},
		{"hidden!=", Hide{1}, Hide{2}, []compare.Path{
			root.Member("Field<0>", Hide{1}, Hide{2}).Diff(1, 2),
		}},
		{"pointer!=", &valueBase, &valueDifferent, []compare.Path{
			root.Diff(valueBase, valueDifferent),
		}},
		{"custom", Special(1), Special(2), []compare.Path{
			root.Diff(Special(1), Special(2)),
		}},
		{"not so special struct", SpecialStruct{true, 1}, SpecialStruct{true, 3}, []compare.Path{
			root.Member("Value", SpecialStruct{true, 1}, SpecialStruct{true, 3}).
				Diff(Special(1), Special(3)),
		}},

		// Mismatched types
		{"int != float64", 1, 1.0, []compare.Path{
			root.Type(1, 1.0).Diff(reflect.TypeOf(0), reflect.TypeOf(0.0)),
		}},
		{"int32 != int64", int32(1), int64(1), []compare.Path{
			root.Type(int32(1), int64(1)).Diff(reflect.TypeOf(int32(0)), reflect.TypeOf(int64(0))),
		}},
		{"float64 != string", 0.5, "hello", []compare.Path{
			root.Type(0.5, "hello").Diff(reflect.TypeOf(0.0), reflect.TypeOf("")),
		}},
		{"slice != array", sliceBase, arrayBase, []compare.Path{
			root.Type(sliceBase, arrayBase).Diff(reflect.TypeOf(sliceBase), reflect.TypeOf(arrayBase)),
		}},
		{"int != string", objectWithInt, objectWithString, []compare.Path{
			root.Member("Children", objectWithInt, objectWithString).
				Index(0, objectWithInt.Children, objectWithString.Children).
				Type(objectWithInt.Children[0], objectWithString.Children[0]).
				Diff(reflect.TypeOf(0), reflect.TypeOf("")),
		}},
		{"struct != alias", structBase, aliasBase, []compare.Path{
			root.Type(structBase, aliasBase).Diff(reflect.TypeOf(Struct{}), reflect.TypeOf(Alias{})),
		}},
		{"struct != other", structBase, otherBase, []compare.Path{
			root.Type(structBase, otherBase).Diff(reflect.TypeOf(Struct{}), reflect.TypeOf(OtherStruct{})),
		}},
		{"map[uint]string != map[int]string", mapBase, mapUint, []compare.Path{
			root.Type(mapBase, mapUint).Diff(reflect.TypeOf(mapBase), reflect.TypeOf(mapUint)),
		}},
	}

	stringTests = []struct {
		diff   compare.Path
		expect string
	}{
		{root,
			""},
		{root.Diff(0, 0),
			"⟦0⟧ != ⟦0⟧"},
		{root.Member("Field", 0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v.Field"},
		{root.Length(0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v·length"},
		{root.Type(0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v·type"},
		{root.Nil(0, 0),
			"nil ⟦0⟧ != ⟦0⟧"},
		{root.Index(2, 0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v[2]"},
		{root.Missing(0, 0),
			"key ⟦0⟧ != ⟦0⟧"},
		{root.Entry("MapKey", 0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v[MapKey]"},
		{root.Member("Slice", 0, 0).Entry(9, 0, 0).Entry("Key", 0, 0).Type(0, 0).Diff(0, 0),
			"⟦0⟧ != ⟦0⟧ for v.Slice[9][Key]·type"},
		{root.Diff(hidden1, compare.Missing),
			"⟦{private:1}⟧ != ⟦⚠ Missing⟧"},
	}
)

type Struct struct {
	Int   int
	Float float32
}

type OtherStruct struct {
	Int   int
	Float float32
}

type Alias Struct

type Object struct {
	Value    int
	Children []interface{}
}

type Hide struct {
	private int
}

type Value int
type Alternate int

type Special int

func compareSpecial(t compare.Comparator, reference, value Special) {
	if reference&0x01 != value&0x01 {
		t.AddDiff(reference, value)
	}
}

type SpecialStruct struct {
	NotSoSpecial bool
	Value        Special
}

func compareSpecialStruct(t compare.Comparator, reference, value SpecialStruct) compare.Action {
	if reference.NotSoSpecial {
		if reference.Value != value.Value {
			t.With(t.Path.Member("Value", reference, value)).AddDiff(reference.Value, value.Value)
		}
		return compare.Done
	}
	return compare.Fallback
}

func init() {
	compare.Register(compareSpecial)
	compare.Register(compareSpecialStruct)
}

func TestDeepEqual(t *testing.T) {
	assert := assert.To(t)
	for _, test := range compareTests {
		got := compare.DeepEqual(test.a, test.b)
		assert.For(test.name).That(got).Equals(len(test.diff) == 0)
	}
}

func TestDiff(t *testing.T) {
	assert := assert.To(t)
	for _, test := range compareTests {
		if len(test.diff) > 0 && len(test.diff[0]) == 0 {
			continue
		}
		got := compare.Diff(test.a, test.b, len(test.diff)+2)
		assert.For(test.name).ThatSlice(got).DeepEquals(test.diff)
	}
}

func TestDiffLimit(t *testing.T) {
	assert := assert.To(t)
	for _, test := range compareTests {
		got := compare.Diff(test.a, test.b, 1)
		expect := test.diff
		if len(expect) > 0 {
			expect = expect[:1]
			if len(expect[0]) == 0 {
				continue
			}
		}
		assert.For(test.name).ThatSlice(got).DeepEquals(expect)
	}
}

func TestStrings(t *testing.T) {
	assert := assert.To(t)
	for _, test := range stringTests {
		got := fmt.Sprint(test.diff)
		assert.For("strings").That(got).Equals(test.expect)
	}
}

func TestPanic(t *testing.T) {
	assert := assert.To(t)
	p := struct{}{}
	defer func() {
		err := recover()
		assert.For("Should panic").That(err).Equals(p)
	}()
	compare.Compare(1, 0, func(compare.Path) {
		panic(p)
	})
	assert.For("panic").Error("Expected compare.DeepEqual to panic")
}

func TestRegisterNonFunc(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").That(err).Equals("Invalid function int")
	}()
	compare.Register(0)
	assert.For("non func").Error("Expected compare.Register to panic")
}

func TestRegisterWrongArgCount(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").That(err).Equals("Compare functions must have 3 args, got func(int, int)")
	}()
	compare.Register(func(reference, value int) {})
	assert.For("wrong arg count").Error("Expected compare.Register to panic")
}

func TestRegisterInvalidComparatorArg(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").That(err).Equals("First argument must be compare.Comparator, got int")
	}()
	compare.Register(func(c, reference, value int) {})
	assert.For("invalid comparator").Error("Expected compare.Register to panic")
}

func TestRegisterInvalidComparatorReturn(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").ThatString(err).Contains("Compare functions must")
	}()
	compare.Register(func(compare.Comparator, unique, unique) string { return "" })
	assert.For("invalid comparator").Error("Expected compare.Register to panic")
}

func TestRegisterNonSymetric(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").That(err).Equals("Comparison arguments must be of the same type, got int and uint8")
	}()
	compare.Register(func(c compare.Comparator, reference int, value byte) {})
	assert.For("non symetric").Error("Expected compare.Register to panic")
}

type registerTestType int

func TestRegisterAlreadyRegistered(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").That(err).Equals("compare_test.registerTestType to compare_test.registerTestType already registered")
	}()
	compare.Register(func(c compare.Comparator, reference, value registerTestType) {})
	compare.Register(func(c compare.Comparator, reference, value registerTestType) {})
	assert.For("already registered").Error("Expected compare.Register to panic")
}

func TestCustomDiffFallback(t *testing.T) {
	assert := assert.To(t)
	custom := compare.Custom{}
	for _, test := range compareTests {
		if len(test.diff) > 0 && len(test.diff[0]) == 0 {
			continue
		}
		got := custom.Diff(test.a, test.b, len(test.diff)+2)
		assert.For(test.name).That(got).DeepEquals(test.diff)
	}
}

type unique bool

func TestCustomDiff(t *testing.T) {
	assert := assert.To(t)
	custom := compare.Custom{}
	got := custom.Diff(unique(false), unique(true), 1)
	diff := []compare.Path{root.Diff(unique(false), unique(true))}
	assert.For("pre-register").ThatSlice(got).DeepEquals(diff)
	custom.Register(func(compare.Comparator, unique, unique) {})
	got = custom.Diff(unique(false), unique(true), 1)
	assert.For("pre-register").ThatSlice(got).DeepEquals(noDiff)
}

func TestInvalidCustomAction(t *testing.T) {
	assert := assert.To(t)
	defer func() {
		err := recover()
		assert.For("recovered").ThatString(err).Contains("Unknown action -1")
	}()
	custom := compare.Custom{}
	custom.Register(func(compare.Comparator, unique, unique) compare.Action { return compare.Action(-1) })
	custom.Diff(unique(false), unique(true), 1)
	assert.For("invalid custom action").Error("Expected custom.Diff to panic")
}
