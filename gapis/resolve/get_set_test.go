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
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/resolve/testatom_pb"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"

	// The following are the imports that generated source files pull in when present
	// Having these here helps out tools that can't cope with missing dependancies

	_ "github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/constset"
)

type (
	stringːstring map[string]string

	intːtestStructPtr map[int]*testStruct

	testStruct struct {
		Str string
		Ref *testStruct
	}

	testStructSlice []testStruct

	testAPI struct{}

	testAtom struct {
		Str  string            `param:"Str"`
		Sli  []bool            `param:"Sli"`
		Ref  *testStruct       `param:"Ref"`
		Map  stringːstring     `param:"Map"`
		PMap intːtestStructPtr `param:"PMap"`
	}
)

var (
	testAPIID = gfxapi.ID{1, 2, 3}

	atomA = &testAtom{
		Str: "aaa",
		Sli: []bool{true, false, true},
		Ref: &testStruct{Str: "ccc", Ref: &testStruct{Str: "ddd"}},
		Map: stringːstring{"cat": "meow", "dog": "woof"},
	}

	atomB = &testAtom{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Map: stringːstring{"bird": "tweet", "fox": "?"},
		PMap: intːtestStructPtr{
			100: &testStruct{Str: "baldrick"},
		},
	}

	cmdA = &service.Command{
		Name: "testAtom",
		Api:  &path.API{Id: path.NewID(id.ID(testAPIID))},
		Parameters: []*service.Parameter{
			{Name: "Str", Value: box.NewValue(atomA.Str)},
			{Name: "Sli", Value: box.NewValue(atomA.Sli)},
			{Name: "Ref", Value: box.NewValue(atomA.Ref)},
			{Name: "Map", Value: box.NewValue(atomA.Map)},
			{Name: "PMap", Value: box.NewValue(atomA.PMap)},
		},
	}

	cmdB = &service.Command{
		Name: "testAtom",
		Api:  &path.API{Id: path.NewID(id.ID(testAPIID))},
		Parameters: []*service.Parameter{
			{Name: "Str", Value: box.NewValue(atomB.Str)},
			{Name: "Sli", Value: box.NewValue(atomB.Sli)},
			{Name: "Ref", Value: box.NewValue(atomB.Ref)},
			{Name: "Map", Value: box.NewValue(atomB.Map)},
			{Name: "PMap", Value: box.NewValue(atomB.PMap)},
		},
	}
)

func (testAPI) Name() string                 { return "foo" }
func (testAPI) ID() gfxapi.ID                { return testAPIID }
func (testAPI) Index() uint8                 { return 15 }
func (testAPI) ConstantSets() *constset.Pack { return nil }
func (testAPI) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (uint32, uint32, *image.Format, error) {
	return 0, 0, nil, nil
}
func (testAPI) Context(*gfxapi.State) gfxapi.Context { return nil }
func (testAtom) AtomName() string                    { return "testAtom" }
func (testAtom) API() gfxapi.API                     { return gfxapi.Find(testAPIID) }
func (testAtom) AtomFlags() atom.Flags               { return 0 }
func (testAtom) Extras() *atom.Extras                { return nil }
func (testAtom) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

func newPathTest(ctx context.Context, a *atom.List) *path.Capture {
	h := &capture.Header{Abi: device.WindowsX86_64}
	p, err := capture.New(ctx, "test", h, a.Atoms)
	if err != nil {
		log.F(ctx, "Couldn't create capture: %v", err)
	}
	return p
}

type boxedTestAtom struct{ box.Value }

func init() {
	gfxapi.Register(testAPI{})
	atom.Register(testAPI{}, &testAtom{})
	protoconv.Register(func(ctx context.Context, a *testAtom) (*testatom_pb.TestAtom, error) {
		return &testatom_pb.TestAtom{Data: box.NewValue(a)}, nil
	}, func(ctx context.Context, b *testatom_pb.TestAtom) (*testAtom, error) {
		var a testAtom
		if err := b.Data.AssignTo(&a); err != nil {
			return nil, err
		}
		return &a, nil
	})
}

func TestGet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := newPathTest(ctx, atom.NewList(atomA, atomB))
	ctx = capture.Put(ctx, p)

	// Get tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{p.Command(1), cmdB, nil},
		{p.Command(1).Parameter("Str"), "xyz", nil},
		{p.Command(1).Parameter("Sli"), []bool{false, true, false}, nil},
		{p.Command(0).Parameter("Ref"), &testStruct{Str: "ccc", Ref: &testStruct{Str: "ddd"}}, nil},
		{p.Command(1).Parameter("Sli").ArrayIndex(1), true, nil},
		{p.Command(1).Parameter("Sli").Slice(1, 3), []bool{true, false}, nil},
		{p.Command(1).Parameter("Str").ArrayIndex(1), byte('y'), nil},
		{p.Command(1).Parameter("Str").Slice(1, 3), "yz", nil},
		{p.Command(1).Parameter("Map").MapIndex("bird"), "tweet", nil},
		{p.Command(1).Parameter("Map").MapIndex([]rune("bird")), "tweet", nil},
		{p.Command(1).Parameter("PMap").MapIndex(100), &testStruct{Str: "baldrick"}, nil},

		// Test invalid paths
		{p.Command(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(1)),
			Path:   p.Command(5).Path(),
		}},
		{p.Command(1).StateAfter(), nil, &service.ErrDataUnavailable{
			Reason: messages.ErrStateUnavailable(),
		}},
		{p.Command(0).StateAfter(), nil, &service.ErrDataUnavailable{
			Reason: messages.ErrStateUnavailable(),
		}},
		{p.Command(1).Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist("testAtom", "doesnotexist"),
			Path:   p.Command(1).Parameter("doesnotexist").Path(),
		}},
		{p.Command(1).Parameter("Ref").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   p.Command(1).Parameter("Ref").Field("ccc").Path(),
		}},
		{p.Command(1).Parameter("Sli").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("[]bool", "ccc"),
			Path:   p.Command(1).Parameter("Sli").Field("ccc").Path(),
		}},
		{p.Command(1).Parameter("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Command(1).Parameter("Sli").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Sli").Slice(2, 4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrSliceOutOfBounds(uint64(2), uint64(4), "Start", "End", uint64(0), uint64(2)),
			Path:   p.Command(1).Parameter("Sli").Slice(2, 4).Path(),
		}},
		{p.Command(1).Parameter("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Command(1).Parameter("Str").ArrayIndex(4).Path(),
		}},
		{p.Command(0).Parameter("Ref").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("ptr<testStruct>"),
			Path:   p.Command(0).Parameter("Ref").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   p.Command(1).Parameter("Map").MapIndex(10.0).Path(),
		}},
		{p.Command(1).Parameter("Map").MapIndex("rabbit"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrMapKeyDoesNotExist("rabbit"),
			Path:   p.Command(1).Parameter("Map").MapIndex("rabbit").Path(),
		}},
		{p.Command(1).Parameter("Ref").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("ptr<testStruct>"),
			Path:   p.Command(1).Parameter("Ref").MapIndex("foo").Path(),
		}},
	} {
		ctx := log.V{"path": test.path.Text()}.Bind(ctx)
		got, err := Get(ctx, test.path.Path())
		assert.With(ctx).That(got).DeepEquals(test.val)
		assert.With(ctx).ThatError(err).DeepEquals(test.err)
	}
}

func TestSet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	a := atom.NewList(atomA, atomB)
	p := newPathTest(ctx, a)
	ctx = capture.Put(ctx, p)

	// Set tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{path: p.Command(0), val: cmdB},
		{path: p.Command(0).Parameter("Str"), val: "bbb"},
		{path: p.Command(0).Parameter("Sli"), val: []bool{false, true, false}},
		{path: p.Command(0).Parameter("Ref"), val: &testStruct{Str: "ddd"}},
		{path: p.Command(0).Parameter("Ref").Field("Str"), val: "purr"},
		{path: p.Command(1).Parameter("Sli").ArrayIndex(1), val: false},
		{path: p.Command(1).Parameter("Map").MapIndex("bird"), val: "churp"},
		{path: p.Command(1).Parameter("Map").MapIndex([]rune("bird")), val: "churp"},

		// Test invalid paths
		{p.Command(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(1)),
			Path:   p.Command(5).Path(),
		}},
		{p.Command(1).StateAfter(), nil, fmt.Errorf("State can not currently be mutated")},
		{p.Command(1).Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist("testAtom", "doesnotexist"),
			Path:   p.Command(1).Parameter("doesnotexist").Path(),
		}},
		{p.Command(1).Parameter("Ref").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   p.Command(1).Parameter("Ref").Field("ccc").Path(),
		}},
		{p.Command(1).Parameter("Sli").Field("ccc"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("[]bool", "ccc"),
			Path:   p.Command(1).Parameter("Sli").Field("ccc").Path(),
		}},
		{p.Command(1).Parameter("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Command(1).Parameter("Sli").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   p.Command(1).Parameter("Str").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Ref").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("ptr<testStruct>"),
			Path:   p.Command(1).Parameter("Ref").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   p.Command(1).Parameter("Map").MapIndex(10.0).Path(),
		}},
		{p.Command(1).Parameter("Ref").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("ptr<testStruct>"),
			Path:   p.Command(1).Parameter("Ref").MapIndex("foo").Path(),
		}},

		// Test invalid sets
		{p.Command(1).Parameter("Sli").ArrayIndex(2), "blah", fmt.Errorf(
			"Slice or array at capture<%v>.commands[1].Sli has element of type bool, got type string", p.Id.ID())},
		{p.Command(1).Parameter("Map").MapIndex("bird"), 10.0, fmt.Errorf(
			"Map at capture<%v>.commands[1].Map has value of type string, got type float64", p.Id.ID())},
	} {
		ctx := log.V{"path": test.path.Text(), "value": test.val}.Bind(ctx)

		path, err := Set(ctx, test.path.Path(), test.val)
		assert.For(ctx, "Set").ThatError(err).DeepEquals(test.err)

		if (path == nil) == (err == nil) {
			log.E(ctx, "Set returned %T %v and %v.", path, path, err)
		}

		if err == nil {
			// Check the paths have changed
			assert.For(ctx, "Set returned path").That(path).DeepNotEquals(test.path)

			ctx := log.V{"changed_path": path.Text()}.Bind(ctx)

			// Get the changed value
			got, err := Get(ctx, path)
			assert.For(ctx, "Get(changed_path) error").ThatError(err).Succeeded()
			ctx = log.V{"got": got}.Bind(ctx)

			// Check it matches what we set it too.
			assert.For(ctx, "Get(changed_path)").That(got).DeepEquals(test.val)
		}
	}
}
