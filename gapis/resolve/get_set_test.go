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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/testcmd"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"

	// The following are the imports that generated source files pull in when present
	// Having these here helps out tools that can't cope with missing dependancies

	_ "github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/os/device"
)

var (
	cmdA = &api.Command{
		Name: "X",
		Api:  &path.API{Id: path.NewID(id.ID(testcmd.APIID))},
		Parameters: []*api.Parameter{
			{Name: "Str", Value: box.NewValue(testcmd.P.Str)},
			{Name: "Sli", Value: box.NewValue(testcmd.P.Sli)},
			{Name: "Ref", Value: box.NewValue(testcmd.P.Ref)},
			{Name: "Ptr", Value: box.NewValue(testcmd.P.Ptr)},
			{Name: "Map", Value: box.NewValue(testcmd.P.Map)},
			{Name: "PMap", Value: box.NewValue(testcmd.P.PMap)},
			{Name: "RMap", Value: box.NewValue(testcmd.P.RMap)},
		},
		Thread: testcmd.P.Thread(),
	}

	cmdB = &api.Command{
		Name: "X",
		Api:  &path.API{Id: path.NewID(id.ID(testcmd.APIID))},
		Parameters: []*api.Parameter{
			{Name: "Str", Value: box.NewValue(testcmd.Q.Str)},
			{Name: "Sli", Value: box.NewValue(testcmd.Q.Sli)},
			{Name: "Ref", Value: box.NewValue(testcmd.Q.Ref)},
			{Name: "Ptr", Value: box.NewValue(testcmd.Q.Ptr)},
			{Name: "Map", Value: box.NewValue(testcmd.Q.Map)},
			{Name: "PMap", Value: box.NewValue(testcmd.Q.PMap)},
			{Name: "RMap", Value: box.NewValue(testcmd.Q.RMap)},
		},
		Thread: testcmd.Q.Thread(),
	}
)

func newPathTest(ctx context.Context, cmds ...api.Cmd) *path.Capture {
	h := &capture.Header{Abi: device.WindowsX86_64}
	p, err := capture.New(ctx, "test", h, cmds)
	if err != nil {
		log.F(ctx, "Couldn't create capture: %v", err)
	}
	return p
}

func TestGet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := newPathTest(ctx, testcmd.P, testcmd.Q)
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
		{p.Command(0).Parameter("Ref"), &testcmd.Struct{Str: "ccc", Ref: &testcmd.Struct{Str: "ddd"}}, nil},
		{p.Command(0).Parameter("Ptr"), testcmd.P.Ptr, nil},
		{p.Command(1).Parameter("Ptr"), testcmd.Q.Ptr, nil},
		{p.Command(1).Parameter("Sli").ArrayIndex(1), true, nil},
		{p.Command(1).Parameter("Sli").Slice(1, 3), []bool{true, false}, nil},
		{p.Command(1).Parameter("Str").ArrayIndex(1), byte('y'), nil},
		{p.Command(1).Parameter("Str").Slice(1, 3), "yz", nil},
		{p.Command(1).Parameter("Map").MapIndex("bird"), "tweet", nil},
		{p.Command(1).Parameter("Map").MapIndex([]rune("bird")), "tweet", nil},
		{p.Command(1).Parameter("PMap").MapIndex(100), &testcmd.Struct{Str: "baldrick"}, nil},
		{p.Command(0).Parameter("RMap").MapIndex("eyes"), "see", nil},
		{p.Command(1).Parameter("RMap").MapIndex("ears"), "hear", nil},

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
			Reason: messages.ErrParameterDoesNotExist("X", "doesnotexist"),
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
			Reason: messages.ErrTypeNotArrayIndexable("ptr<Struct>"),
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
			Reason: messages.ErrTypeNotMapIndexable("ptr<Struct>"),
			Path:   p.Command(1).Parameter("Ref").MapIndex("foo").Path(),
		}},
	} {
		got, err := Get(ctx, test.path.Path())
		assert.For(ctx, "Get(%v)", test.path).That(got).DeepEquals(test.val)
		assert.For(ctx, "Get(%v)", test.path).ThatError(err).DeepEquals(test.err)
	}
}

func TestSet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	p := newPathTest(ctx, testcmd.P, testcmd.Q)
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
		{path: p.Command(0).Parameter("Ref"), val: &testcmd.Struct{Str: "ddd"}},
		{path: p.Command(0).Parameter("Ref").Field("Str"), val: "purr"},
		{path: p.Command(0).Parameter("Ptr"), val: testcmd.Q.Ptr},
		{path: p.Command(1).Parameter("Ptr"), val: testcmd.P.Ptr},
		{path: p.Command(1).Parameter("Sli").ArrayIndex(1), val: false},
		{path: p.Command(1).Parameter("Map").MapIndex("bird"), val: "churp"},
		{path: p.Command(1).Parameter("Map").MapIndex([]rune("bird")), val: "churp"},
		{path: p.Command(0).Parameter("RMap").MapIndex("eyes"), val: "blind"},
		{path: p.Command(1).Parameter("RMap").MapIndex("ears"), val: "deaf"},

		// Test invalid paths
		{p.Command(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(1)),
			Path:   p.Command(5).Path(),
		}},
		{p.Command(1).StateAfter(), nil, fmt.Errorf("State can not currently be mutated")},
		{p.Command(1).Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist("X", "doesnotexist"),
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
			Reason: messages.ErrTypeNotArrayIndexable("ptr<Struct>"),
			Path:   p.Command(1).Parameter("Ref").ArrayIndex(4).Path(),
		}},
		{p.Command(1).Parameter("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   p.Command(1).Parameter("Map").MapIndex(10.0).Path(),
		}},
		{p.Command(1).Parameter("Ref").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("ptr<Struct>"),
			Path:   p.Command(1).Parameter("Ref").MapIndex("foo").Path(),
		}},

		// Test invalid sets
		{p.Command(1).Parameter("Sli").ArrayIndex(2), "blah", fmt.Errorf(
			"Slice or array at capture<%v>.commands[1].Sli has element of type bool, got type string", p.Id.ID())},
		{p.Command(1).Parameter("Map").MapIndex("bird"), 10.0, fmt.Errorf(
			"Map at capture<%v>.commands[1].Map has value of type string, got type float64", p.Id.ID())},
	} {
		ctx := log.V{"path": test.path, "value": test.val}.Bind(ctx)

		path, err := Set(ctx, test.path.Path(), test.val)
		assert.For(ctx, "Set").ThatError(err).DeepEquals(test.err)

		if (path == nil) == (err == nil) {
			log.E(ctx, "Set returned %T %v and %v.", path, path, err)
		}

		if err == nil {
			// Check the paths have changed
			assert.For(ctx, "Set returned path").That(path).DeepNotEquals(test.path)

			ctx := log.V{"changed_path": path}.Bind(ctx)

			// Get the changed value
			got, err := Get(ctx, path)
			assert.For(ctx, "Get(changed_path) error").ThatError(err).Succeeded()
			ctx = log.V{"got": got}.Bind(ctx)

			// Check it matches what we set it too.
			assert.For(ctx, "Get(changed_path)").That(got).DeepEquals(test.val)
		}
	}
}
