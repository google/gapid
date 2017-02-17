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
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
	_ "github.com/google/gapid/framework/binary/any"
	"github.com/google/gapid/framework/binary/registry"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

var Namespace = registry.NewNamespace()

type testStruct struct {
	binary.Generate

	Str string
	Ptr *testStruct
}

type stringːstring map[string]string
type intːtestStructPtr map[int]*testStruct

type testAtom struct {
	binary.Generate

	api gfxapi.ID

	Str  string
	Sli  []bool
	Any  interface{}
	Ptr  *testStruct
	Map  stringːstring
	PMap intːtestStructPtr
}

var _ atom.Atom = &testAtom{}

func (m stringːstring) KeysSorted() []string {
	s := make(sort.StringSlice, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Sort(s)
	return s
}

type testStructSlice []testStruct

func (a testStructSlice) Len() int      { return len(a) }
func (a testStructSlice) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a testStructSlice) Less(i, j int) bool {
	return reflect.ValueOf(a[i].Ptr).Pointer() < reflect.ValueOf(a[j].Ptr).Pointer() && a[i].Str < a[j].Str
}

func (m intːtestStructPtr) KeysSorted() []int {
	s := make([]int, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	sort.Ints(s)
	return s
}

type testAPI struct{}

func (testAPI) Name() string  { return "foo" }
func (testAPI) ID() gfxapi.ID { return gfxapi.ID{1, 2, 3} }
func (testAPI) Index() uint8  { return 15 }
func (testAPI) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (uint32, uint32, *image.Format, error) {
	return 0, 0, nil, nil
}
func (testAPI) Context(*gfxapi.State) gfxapi.Context { return nil }

func (a testAtom) API() gfxapi.API     { return gfxapi.Find(a.api) }
func (testAtom) AtomFlags() atom.Flags { return 0 }
func (testAtom) Extras() *atom.Extras  { return nil }
func (testAtom) Mutate(log.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

func newPathTest(ctx log.Context, a *atom.List) *path.Capture {
	p, err := capture.ImportAtomList(ctx, "test", a)
	if err != nil {
		jot.Fatal(ctx, err, "Creating capture")
	}
	return p
}

func init() {
	gfxapi.Register(testAPI{})
	registry.Global.AddFallbacks(Namespace)
}

func TestGet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	atomA := &testAtom{
		api: testAPI{}.ID(),
		Str: "aaa",
		Sli: []bool{true, false, true},
		Any: &testStruct{Str: "bbb"},
		Ptr: &testStruct{Str: "ccc", Ptr: &testStruct{Str: "ddd"}},
		Map: stringːstring{"cat": "meow", "dog": "woof"},
	}
	atomB := &testAtom{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Any: &testStruct{Str: "www"},
		Map: stringːstring{"bird": "tweet", "fox": "?"},
		PMap: intːtestStructPtr{
			100: &testStruct{Str: "baldrick"},
		},
	}
	a := atom.NewList(atomA, atomB)
	p := newPathTest(ctx, a)
	ctx = capture.Put(ctx, p)

	// Get tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{p.Commands().Index(1), a.Atoms[1], nil},
		{p.Commands().Index(1).Parameter("Str"), "xyz", nil},
		{p.Commands().Index(1).Parameter("Sli"), []bool{false, true, false}, nil},
		{p.Commands().Index(1).Parameter("Any"), &testStruct{Str: "www"}, nil},
		{p.Commands().Index(0).Parameter("Ptr"), &testStruct{Str: "ccc", Ptr: &testStruct{Str: "ddd"}}, nil},
		{p.Commands().Index(1).Parameter("Sli").ArrayIndex(1), true, nil},
		{p.Commands().Index(1).Parameter("Sli").Slice(1, 3), []bool{true, false}, nil},
		{p.Commands().Index(1).Parameter("Str").ArrayIndex(1), byte('y'), nil},
		{p.Commands().Index(1).Parameter("Str").Slice(1, 3), "yz", nil},
		{p.Commands().Index(1).Parameter("Map").MapIndex("bird"), "tweet", nil},
		{p.Commands().Index(1).Parameter("Map").MapIndex([]rune("bird")), "tweet", nil},
		{p.Commands().Index(1).Parameter("PMap").MapIndex(100), &testStruct{Str: "baldrick"}, nil},

		// Test invalid paths
		{p.Commands().Index(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(1)),
			Path:   p.Commands().Index(5).Path(),
		}},
		{p.Commands().Index(1).StateAfter(), nil, &service.ErrDataUnavailable{
			Reason: messages.ErrStateUnavailable(),
		}},
		{p.Commands().Index(0).StateAfter(), nil, &service.ErrDataUnavailable{
			Reason: messages.ErrStateUnavailable(),
		}},
		{p.Commands().Index(1).Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("testAtom", "doesnotexist"),
			Path:   p.Commands().Index(1).Parameter("doesnotexist").Path(),
		}},
		{p.Commands().Index(1).Parameter("Ptr").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   p.Commands().Index(1).Parameter("Ptr").Field("ccc").Path(),
		}},
		{p.Commands().Index(1).Parameter("Sli").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("[]bool", "ccc"),
			Path:   p.Commands().Index(1).Parameter("Sli").Field("ccc").Path(),
		}},
		{p.Commands().Index(1).Parameter("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Commands().Index(1).Parameter("Sli").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Sli").Slice(2, 4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrSliceOutOfBounds(uint64(2), uint64(4), "Start", "End", uint64(0), uint64(2)),
			Path:   p.Commands().Index(1).Parameter("Sli").Slice(2, 4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Commands().Index(1).Parameter("Str").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(0).Parameter("Ptr").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("ptr<testStruct>"),
			Path:   p.Commands().Index(0).Parameter("Ptr").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   p.Commands().Index(1).Parameter("Map").MapIndex(10.0).Path(),
		}},
		{p.Commands().Index(1).Parameter("Map").MapIndex("rabbit"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrMapKeyDoesNotExist("rabbit"),
			Path:   p.Commands().Index(1).Parameter("Map").MapIndex("rabbit").Path(),
		}},
		{p.Commands().Index(1).Parameter("Ptr").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("ptr<testStruct>"),
			Path:   p.Commands().Index(1).Parameter("Ptr").MapIndex("foo").Path(),
		}},
	} {
		ctx := ctx.V("path", test.path.Text())
		got, err := Get(ctx, test.path.Path())
		assert.With(ctx).That(got).DeepEquals(test.val)
		assert.With(ctx).ThatError(err).DeepEquals(test.err)
	}
}

func TestSet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	atomA := &testAtom{
		api: testAPI{}.ID(),
		Str: "aaa",
		Sli: []bool{true, false, true},
		Any: &testStruct{Str: "bbb"},
		Ptr: &testStruct{Str: "ccc", Ptr: &testStruct{Str: "ddd"}},
		Map: stringːstring{"cat": "meow", "dog": "woof"},
	}
	atomB := &testAtom{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Any: &testStruct{Str: "www"},
		Map: stringːstring{"bird": "tweet", "fox": "?"},
		PMap: intːtestStructPtr{
			100: &testStruct{Str: "baldrick"},
		},
	}
	a := atom.NewList(atomA, atomB)
	p := newPathTest(ctx, a)
	ctx = capture.Put(ctx, p)

	// Set tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{path: p.Commands(), val: atom.NewList(atomB)},
		{path: p.Commands().Index(0), val: atomB},
		{path: p.Commands().Index(0).Parameter("Str"), val: "bbb"},
		{path: p.Commands().Index(0).Parameter("Sli"), val: []bool{false, true, false}},
		{path: p.Commands().Index(0).Parameter("Any"), val: 0.123},
		{path: p.Commands().Index(0).Parameter("Ptr"), val: &testStruct{Str: "ddd"}},
		{path: p.Commands().Index(0).Parameter("Ptr").Field("Str"), val: "purr"},
		{path: p.Commands().Index(1).Parameter("Sli").ArrayIndex(1), val: false},
		{path: p.Commands().Index(1).Parameter("Map").MapIndex("bird"), val: "churp"},
		{path: p.Commands().Index(1).Parameter("Map").MapIndex([]rune("bird")), val: "churp"},

		// Test invalid paths
		{p.Commands().Index(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(1)),
			Path:   p.Commands().Index(5).Path(),
		}},
		{p.Commands().Index(1).StateAfter(), nil, fmt.Errorf("State can not currently be mutated")},
		{p.Commands().Index(1).Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("testAtom", "doesnotexist"),
			Path:   p.Commands().Index(1).Parameter("doesnotexist").Path(),
		}},
		{p.Commands().Index(1).Parameter("Ptr").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   p.Commands().Index(1).Parameter("Ptr").Field("ccc").Path(),
		}},
		{p.Commands().Index(1).Parameter("Sli").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("[]bool", "ccc"),
			Path:   p.Commands().Index(1).Parameter("Sli").Field("ccc").Path(),
		}},
		{p.Commands().Index(1).Parameter("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Commands().Index(1).Parameter("Sli").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Commands().Index(1).Parameter("Str").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Ptr").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("ptr<testStruct>"),
			Path:   p.Commands().Index(1).Parameter("Ptr").ArrayIndex(4).Path(),
		}},
		{p.Commands().Index(1).Parameter("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   p.Commands().Index(1).Parameter("Map").MapIndex(10.0).Path(),
		}},
		{p.Commands().Index(1).Parameter("Ptr").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("ptr<testStruct>"),
			Path:   p.Commands().Index(1).Parameter("Ptr").MapIndex("foo").Path(),
		}},

		// Test invalid sets
		{p.Commands().Index(1).Parameter("Sli").ArrayIndex(2), "blah", fmt.Errorf(
			"Slice or array at capture<%v>.commands[1].Sli has element of type bool, got type string", p.Id.ID())},
		{p.Commands().Index(1).Parameter("Map").MapIndex("bird"), 10.0, fmt.Errorf(
			"Map at capture<%v>.commands[1].Map has value of type string, got type float64", p.Id.ID())},
	} {
		ctx := ctx.V("path", test.path.Text()).V("value", test.val)

		path, err := Set(ctx, test.path.Path(), test.val)
		assert.With(ctx).ThatError(err).DeepEquals(test.err)

		if (path == nil) == (err == nil) {
			ctx.Error().Logf("Set returned %T %v and %v.", path, path, err)
		}

		if err == nil {
			// Check the paths have changed
			assert.With(ctx).That(p).DeepNotEquals(test.path)

			ctx := ctx.V("changed_path", path.Text())

			// Get the changed value
			got, err := Get(ctx, path)
			assert.For(ctx, "Get(changed_path) error").ThatError(err).Succeeded()

			// Check it matches what we set it too.
			assert.For(ctx, "Get(changed_path)").That(got).DeepEquals(test.val)
		}
	}
}
