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

package resolve

import (
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

var intType = reflect.TypeOf(memory.Int(0))

const intSize = 8

var _ api.PropertyProvider = TestStruct{}

type TestStruct struct {
	Bool      bool
	Int       int
	Float     float32
	String    string
	Reference *TestStruct
	Map       map[int]string
	Array     []int
	Slice     memory.Slice
	Pointer   memory.Pointer
	Interface interface{}
}

// Properties returns the field properties for the struct.
func (s TestStruct) Properties() api.Properties {
	return api.Properties{
		/* 0 */ api.NewProperty("Bool", func() bool { return s.Bool }, nil),
		/* 1 */ api.NewProperty("Int", func() int { return s.Int }, nil),
		/* 2 */ api.NewProperty("Float", func() float32 { return s.Float }, nil),
		/* 3 */ api.NewProperty("String", func() string { return s.String }, nil),
		/* 4 */ api.NewProperty("Reference", func() *TestStruct { return s.Reference }, nil),
		/* 5 */ api.NewProperty("Map", func() map[int]string { return s.Map }, nil),
		/* 6 */ api.NewProperty("Array", func() []int { return s.Array }, nil),
		/* 7 */ api.NewProperty("Slice", func() memory.Slice { return s.Slice }, nil),
		/* 8 */ api.NewProperty("Pointer", func() memory.Pointer { return s.Pointer }, nil),
		/* 9 */ api.NewProperty("Interface", func() interface{} { return s.Interface }, nil),
	}
}

var _ api.PropertyProvider = TestState{}

type TestState struct {
	Bool       bool
	Int        int
	Float      float32
	String     string
	ReferenceA *TestStruct
	ReferenceB *TestStruct
	ReferenceC *TestStruct
}

// Properties returns the field properties for the state.
func (s TestState) Properties() api.Properties {
	return api.Properties{
		/* 0 */ api.NewProperty("Bool", func() bool { return s.Bool }, nil),
		/* 1 */ api.NewProperty("Int", func() int { return s.Int }, nil),
		/* 2 */ api.NewProperty("Float", func() float32 { return s.Float }, nil),
		/* 3 */ api.NewProperty("String", func() string { return s.String }, nil),
		/* 4 */ api.NewProperty("ReferenceA", func() *TestStruct { return s.ReferenceA }, nil),
		/* 5 */ api.NewProperty("ReferenceB", func() *TestStruct { return s.ReferenceB }, nil),
		/* 6 */ api.NewProperty("ReferenceC", func() *TestStruct { return s.ReferenceC }, nil),
	}
}

var testState = TestState{
	Bool:   true,
	Int:    42,
	Float:  123.456,
	String: "meow",
	ReferenceA: &TestStruct{
		Bool:      true,
		Int:       7,
		Float:     0.25,
		String:    "hello cat",
		Reference: nil,
		Map:       map[int]string{1: "one", 5: "five", 9: "nine"},
		Array:     []int{0, 10, 20, 30, 40},
		Slice:     memory.NewSlice(0x1000, 0x1000, 5*intSize, 5, memory.ApplicationPool, intType),
		Pointer:   memory.NewPtr(0x1010, reflect.TypeOf(memory.Size(0))),
		Interface: &TestStruct{},
	},
	ReferenceB: &TestStruct{
		String: "this is a really, really, really, really, really, really, really long string",
		Map: map[int]string{
			0: "0.0", 1: "0.1", 2: "0.2", 3: "0.3", 4: "0.4", 5: "0.5", 6: "0.6", 7: "0.7", 8: "0.8", 9: "0.9",
			10: "1.0", 11: "1.1", 12: "1.2", 13: "1.3", 14: "1.4", 15: "1.5", 16: "1.6", 17: "1.7", 18: "1.8", 19: "1.9",
			20: "2.0", 21: "2.1", 22: "2.2", 23: "2.3", 24: "2.4", 25: "2.5", 26: "2.6", 27: "2.7", 28: "2.8", 29: "2.9",
			30: "3.0", 31: "3.1", 32: "3.2", 33: "3.3", 34: "3.4", 35: "3.5",
		},
		Array: []int{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
			30, 31, 32, 33,
		},
		Slice: memory.NewSlice(0x1000, 0x1000, 1005*intSize, 1005, memory.ApplicationPool, intType),
	},
	ReferenceC: nil,
}

func TestSubgroupSize(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		groupLimit, childCount, expected uint64
	}{
		{10, 5, 1},
		{10, 9, 1},
		{10, 10, 1},
		{10, 11, 10},
		{10, 20, 10},
		{10, 99, 10},
		{10, 100, 10},
		{10, 101, 100},
	} {
		got := subgroupSize(test.groupLimit, test.childCount)
		assert.For(ctx, "subgroupSize(%v, %v)", test.groupLimit, test.childCount).
			That(got).Equals(test.expected)
	}
}

func TestSubgroupCount(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		groupLimit, childCount, expected uint64
	}{
		{10, 5, 5},
		{10, 9, 9},
		{10, 10, 10},
		{10, 11, 2},
		{10, 20, 2},
		{10, 99, 10},
		{10, 100, 10},
		{10, 101, 2},
	} {
		got := subgroupCount(test.groupLimit, test.childCount)
		assert.For(ctx, "subgroupCount(%v, %v)", test.groupLimit, test.childCount).
			That(got).Equals(test.expected)
	}
}

func TestSubgroupRange(t *testing.T) {
	ctx := log.Testing(t)
	for _, test := range []struct {
		groupLimit, childCount, idx uint64
		s, e                        uint64
	}{
		{groupLimit: 10, childCount: 5, idx: 3, s: 3, e: 4},
		{groupLimit: 10, childCount: 9, idx: 3, s: 3, e: 4},
		{groupLimit: 10, childCount: 10, idx: 3, s: 3, e: 4},
		{groupLimit: 10, childCount: 11, idx: 0, s: 0, e: 10},
		{groupLimit: 10, childCount: 11, idx: 1, s: 10, e: 11},
		{groupLimit: 10, childCount: 20, idx: 0, s: 0, e: 10},
		{groupLimit: 10, childCount: 20, idx: 1, s: 10, e: 20},
		{groupLimit: 10, childCount: 99, idx: 5, s: 50, e: 60},
		{groupLimit: 10, childCount: 100, idx: 5, s: 50, e: 60},
		{groupLimit: 10, childCount: 101, idx: 0, s: 0, e: 100},
		{groupLimit: 10, childCount: 101, idx: 1, s: 100, e: 101},
	} {
		type R struct{ S, E uint64 }
		s, e := subgroupRange(test.groupLimit, test.childCount, test.idx)
		assert.For(ctx, "subgroupRange(groupLimit: %v, childCount: %v, idx: %v)", test.groupLimit, test.childCount, test.idx).
			That(R{s, e}).Equals(R{test.s, test.e})
	}
}
func TestStateTreeNode(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	header := capture.Header{ABI: device.AndroidARM64v8a}
	cap, err := capture.NewGraphicsCapture(ctx, "test-capture", &header, nil, []api.Cmd{})
	if err != nil {
		panic(err)
	}
	c, err := cap.Path(ctx)
	if err != nil {
		panic(err)
	}
	ctx = capture.Put(ctx, c)
	rootPath := c.Command(0).StateAfter()
	gs, err := capture.NewState(ctx)
	if err != nil {
		panic(err)
	}
	tree := &stateTree{
		globalState: gs,
		root: &stn{
			name:  "root",
			value: reflect.ValueOf(testState),
			path:  rootPath,
		},
		api:        &path.API{ID: path.NewID(id.ID(test.API{}.ID()))},
		groupLimit: 10,
	}
	root := &path.StateTreeNode{Indices: []uint64{}}

	// Write some data to 0x1000.
	e := gs.MemoryEncoder(memory.ApplicationPool, memory.Range{Base: 0x1000, Size: 0x8000})
	for i := 0; i < 0x1000; i++ {
		e.I64(int64(i * 10))
	}

	for _, test := range []struct {
		path     *path.StateTreeNode
		expected *service.StateTreeNode
	}{
		{
			root,
			&service.StateTreeNode{
				NumChildren: 7,
				Name:        "root",
				ValuePath:   rootPath.Path(),
			},
		}, {
			root.Index(0), // 0
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Bool",
				ValuePath:      rootPath.Field("Bool").Path(),
				Preview:        box.NewValue(true),
				PreviewIsValue: true,
			},
		}, {
			root.Index(1), // 1
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Int",
				ValuePath:      rootPath.Field("Int").Path(),
				Preview:        box.NewValue(42),
				PreviewIsValue: true,
			},
		}, {
			root.Index(2), // 2
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Float",
				ValuePath:      rootPath.Field("Float").Path(),
				Preview:        box.NewValue(float32(123.456)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(3), // 3
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "String",
				ValuePath:      rootPath.Field("String").Path(),
				Preview:        box.NewValue("meow"),
				PreviewIsValue: true,
			},
		},
		// testState.ReferenceA
		{
			root.Index(4), // [4]
			&service.StateTreeNode{
				NumChildren: 10,
				Name:        "ReferenceA",
				ValuePath:   rootPath.Field("ReferenceA").Path(),
			},
		}, {
			root.Index(4, 0), // [4.0]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Bool",
				ValuePath:      rootPath.Field("ReferenceA").Field("Bool").Path(),
				Preview:        box.NewValue(true),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 1), // [4.1]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Int",
				ValuePath:      rootPath.Field("ReferenceA").Field("Int").Path(),
				Preview:        box.NewValue(7),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 2), // [4.2]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Float",
				ValuePath:      rootPath.Field("ReferenceA").Field("Float").Path(),
				Preview:        box.NewValue(float32(0.25)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 3), // [4.3]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "String",
				ValuePath:      rootPath.Field("ReferenceA").Field("String").Path(),
				Preview:        box.NewValue("hello cat"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 4), // [4.4]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Reference",
				ValuePath:      rootPath.Field("ReferenceA").Field("Reference").Path(),
				Preview:        box.NewValue((*TestStruct)(nil)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 5), // [4.5]
			&service.StateTreeNode{
				NumChildren:    3,
				Name:           "Map",
				ValuePath:      rootPath.Field("ReferenceA").Field("Map").Path(),
				PreviewIsValue: false,
			},
		}, {
			root.Index(4, 5, 0), // [4.5.0]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "1",
				ValuePath:      rootPath.Field("ReferenceA").Field("Map").MapIndex(1).Path(),
				Preview:        box.NewValue("one"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 5, 1), // [4.5.1]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "5",
				ValuePath:      rootPath.Field("ReferenceA").Field("Map").MapIndex(5).Path(),
				Preview:        box.NewValue("five"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 5, 2), // [4.5.2]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "9",
				ValuePath:      rootPath.Field("ReferenceA").Field("Map").MapIndex(9).Path(),
				Preview:        box.NewValue("nine"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 6), // [4.6]
			&service.StateTreeNode{
				NumChildren:    5,
				Name:           "Array",
				ValuePath:      rootPath.Field("ReferenceA").Field("Array").Path(),
				Preview:        box.NewValue([]int{0, 10, 20, 30}),
				PreviewIsValue: false,
			},
		}, {
			root.Index(4, 6, 3), // [4.6.3]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "3",
				ValuePath:      rootPath.Field("ReferenceA").Field("Array").ArrayIndex(3).Path(),
				Preview:        box.NewValue(30),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 7), // [4.7]
			&service.StateTreeNode{
				NumChildren:    5,
				Name:           "Slice",
				ValuePath:      rootPath.Field("ReferenceA").Field("Slice").Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x1000, 5*intSize, 5, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 7, 0), // [4.7.0]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "0",
				ValuePath:      rootPath.Field("ReferenceA").Field("Slice").ArrayIndex(0).Path(),
				Preview:        box.NewValue(memory.Int(0)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 7, 2), // [4.7.2]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "2",
				ValuePath:      rootPath.Field("ReferenceA").Field("Slice").ArrayIndex(2).Path(),
				Preview:        box.NewValue(memory.Int(20)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 7, 4), // [4.7.4]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "4",
				ValuePath:      rootPath.Field("ReferenceA").Field("Slice").ArrayIndex(4).Path(),
				Preview:        box.NewValue(memory.Int(40)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 8), // [4.8]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "Pointer",
				ValuePath:      rootPath.Field("ReferenceA").Field("Pointer").Path(),
				Preview:        box.NewValue(memory.NewPtr(0x1010, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(4, 9), // [4.9]
			&service.StateTreeNode{
				NumChildren: 10,
				Name:        "Interface",
				ValuePath:   rootPath.Field("ReferenceA").Field("Interface").Path(),
			},
		},
		// testState.ReferenceB
		{
			root.Index(5), // [5]
			&service.StateTreeNode{
				NumChildren: 10,
				Name:        "ReferenceB",
				ValuePath:   rootPath.Field("ReferenceB").Path(),
			},
		}, {
			root.Index(5, 3), // [5.3]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "String",
				ValuePath:      rootPath.Field("ReferenceB").Field("String").Path(),
				Preview:        box.NewValue("this is a really, really, really, really, really, really, reall…"),
				PreviewIsValue: false,
			},
		}, {
			root.Index(5, 5), // [5.5]
			&service.StateTreeNode{
				NumChildren:    36,
				Name:           "Map",
				ValuePath:      rootPath.Field("ReferenceB").Field("Map").Path(),
				PreviewIsValue: false,
			},
		}, {
			root.Index(5, 5, 0), // [5.5.0]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "0",
				ValuePath:      rootPath.Field("ReferenceB").Field("Map").MapIndex(0).Path(),
				Preview:        box.NewValue("0.0"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 5, 5), // [5.5.5]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "5",
				ValuePath:      rootPath.Field("ReferenceB").Field("Map").MapIndex(5).Path(),
				Preview:        box.NewValue("0.5"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 5, 15), // [5.5.15]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "15",
				ValuePath:      rootPath.Field("ReferenceB").Field("Map").MapIndex(15).Path(),
				Preview:        box.NewValue("1.5"),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 6), // [5.6]
			&service.StateTreeNode{
				NumChildren:    4,
				Name:           "Array",
				ValuePath:      rootPath.Field("ReferenceB").Field("Array").Path(),
				Preview:        box.NewValue([]int{0, 1, 2, 3}),
				PreviewIsValue: false,
			},
		}, {
			root.Index(5, 6, 1), // [5.6.1]
			&service.StateTreeNode{
				NumChildren:    10,
				Name:           "[10 - 19]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Array").Slice(10, 19).Path(),
				Preview:        box.NewValue([]int{10, 11, 12, 13}),
				PreviewIsValue: false,
			},
		}, {
			root.Index(5, 6, 1, 2), // [5.6.1.2]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "12",
				ValuePath:      rootPath.Field("ReferenceB").Field("Array").Slice(10, 19).ArrayIndex(2).Path(),
				Preview:        box.NewValue(12),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 6, 3), // [5.6.3]
			&service.StateTreeNode{
				NumChildren:    4,
				Name:           "[30 - 33]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Array").Slice(30, 33).Path(),
				Preview:        box.NewValue([]int{30, 31, 32, 33}),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 6, 3, 2), // [5.6.3.2]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "32",
				ValuePath:      rootPath.Field("ReferenceB").Field("Array").Slice(30, 33).ArrayIndex(2).Path(),
				Preview:        box.NewValue(32),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7), // [5.7]
			&service.StateTreeNode{
				NumChildren:    2,
				Name:           "Slice",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x1000, 1005*intSize, 1005, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 0), // [5.7.0]
			&service.StateTreeNode{
				NumChildren:    10,
				Name:           "[0 - 999]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(0, 999).Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x1000, 1000*intSize, 1000, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 0, 4), // [5.7.0.4]
			&service.StateTreeNode{
				NumChildren:    10,
				Name:           "[400 - 499]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(0, 999).Slice(400, 499).Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x1C80, 100*intSize, 100, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 0, 4, 3), // [5.7.0.4.3]
			&service.StateTreeNode{
				NumChildren:    10,
				Name:           "[430 - 439]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(0, 999).Slice(400, 499).Slice(30, 39).Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x1D70, 10*intSize, 10, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 0, 4, 3, 5), // [5.7.0.4.3.5]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "435",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(0, 999).Slice(400, 499).Slice(30, 39).ArrayIndex(5).Path(),
				Preview:        box.NewValue(memory.Int(4350)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 1), // [5.7.1]
			&service.StateTreeNode{
				NumChildren:    5,
				Name:           "[1000 - 1004]",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(1000, 1004).Path(),
				Preview:        box.NewValue(memory.NewSlice(0x1000, 0x2F40, 5*intSize, 5, memory.ApplicationPool, intType)),
				PreviewIsValue: true,
			},
		}, {
			root.Index(5, 7, 1, 3), // [5.7.1.3]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "1003",
				ValuePath:      rootPath.Field("ReferenceB").Field("Slice").Slice(1000, 1004).ArrayIndex(3).Path(),
				Preview:        box.NewValue(memory.Int(10030)),
				PreviewIsValue: true,
			},
		},
		// testState.ReferenceC
		{
			root.Index(6), // [6]
			&service.StateTreeNode{
				NumChildren:    0,
				Name:           "ReferenceC",
				ValuePath:      rootPath.Field("ReferenceC").Path(),
				Preview:        box.NewValue((*TestStruct)(nil)),
				PreviewIsValue: true,
			},
		},
	} {

		node, err := stateTreeNode(ctx, tree, test.path)
		if assert.For(ctx, "stateTreeNode(%v)", test.path).
			ThatError(err).Succeeded() {
			assert.For(ctx, "stateTreeNode(%v)", test.path).
				That(node).DeepEquals(test.expected)
		}

		ctx := log.V{"path": test.path}.Bind(ctx)
		p := test.expected.ValuePath.Node()
		indices, err := stateTreeNodePath(ctx, tree, p)
		if assert.For(ctx, "stateTreeNodePath(%v)", p).
			ThatError(err).Succeeded() {
			assert.For(ctx, "stateTreeNodePath(%v)", p).
				ThatSlice(indices).Equals(test.path.Indices)
		}
	}
}
