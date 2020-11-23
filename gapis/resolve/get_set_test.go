// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License")
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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/test"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func newPathTest(ctx context.Context) *path.Capture {
	h := &capture.Header{ABI: device.WindowsX86_64}
	cb := test.CommandBuilder{}
	cmds := []api.Cmd{
		cb.CmdTypeMix(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, true, test.Voidᵖ(0x12345678), 2),
		cb.CmdTypeMix(1, 15, 25, 35, 45, 55, 65, 75, 85, 95, 105, false, test.Voidᵖ(0x87654321), 3),
		cb.PrimeState(test.U8ᵖ(0x89abcdef)),
	}
	p, err := capture.NewGraphicsCapture(ctx, "test", h, nil, cmds)
	if err != nil {
		log.F(ctx, true, "Couldn't create capture: %v", err)
	}
	path, err := p.Path(ctx)
	if err != nil {
		log.F(ctx, true, "Couldn't get capture path: %v", err)
	}
	return path
}

func TestGet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := newPathTest(ctx)
	ctx = capture.Put(ctx, p)
	cA, cB := p.Command(0), p.Command(1)
	sA, sB := p.Command(0).StateAfter(), p.Command(2).StateAfter()

	// Get tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{cA.Parameter("ID"), uint64(0), nil},
		{cA.Parameter("U8"), uint8(10), nil},
		{cA.Parameter("S8"), int8(20), nil},
		{cA.Parameter("U16"), uint16(30), nil},
		{cA.Parameter("S16"), int16(40), nil},
		{cA.Parameter("U32"), uint32(50), nil},
		{cA.Parameter("S32"), int32(60), nil},
		{cA.Parameter("U64"), uint64(70), nil},
		{cA.Parameter("S64"), int64(80), nil},
		{cA.Parameter("F32"), float32(90), nil},
		{cA.Parameter("F64"), float64(100), nil},
		{cA.Parameter("Bool"), true, nil},
		{cA.Parameter("Ptr"), test.Voidᵖ(0x12345678), nil},
		{cA.Result(), uint32(2), nil},

		{cB.Parameter("ID"), uint64(1), nil},
		{cB.Parameter("U8"), uint8(15), nil},
		{cB.Parameter("S8"), int8(25), nil},
		{cB.Parameter("U16"), uint16(35), nil},
		{cB.Parameter("S16"), int16(45), nil},
		{cB.Parameter("U32"), uint32(55), nil},
		{cB.Parameter("S32"), int32(65), nil},
		{cB.Parameter("U64"), uint64(75), nil},
		{cB.Parameter("S64"), int64(85), nil},
		{cB.Parameter("F32"), float32(95), nil},
		{cB.Parameter("F64"), float64(105), nil},
		{cB.Parameter("Bool"), false, nil},
		{cB.Parameter("Ptr"), test.Voidᵖ(0x87654321), nil},
		{cB.Result(), uint32(3), nil},

		{sA.Field("Str"), "", nil},
		{sA.Field("Sli"), test.NewBoolˢ(0, 0, 0, 0, 0), nil},
		{sA.Field("Ref"), test.NilComplexʳ, nil},
		{sA.Field("Ptr"), test.U8ᵖ(0), nil},

		{sB.Field("Str"), "aaa", nil},
		{sB.Field("Sli"), test.NewBoolˢ(0, 0, 3, 3, 1), nil},
		{sB.Field("Sli").ArrayIndex(0), test.NewBoolˢ(0, 0, 1, 1, 1), nil},
		{sB.Field("Sli").ArrayIndex(1), test.NewBoolˢ(0, 1, 1, 1, 1), nil},
		{sB.Field("Sli").ArrayIndex(2), test.NewBoolˢ(0, 2, 1, 1, 1), nil},
		{sB.Field("Ref").Field("Strings").MapIndex("123"), uint32(123), nil},
		{sB.Field("Ref").Field("RefObject").Field("value"), uint32(555), nil},
		{sB.Field("Ptr"), test.U8ᵖ(0x89abcdef), nil},
		{sB.Field("Map").MapIndex("cat").Field("Object").Field("value"), uint32(100), nil},
		{sB.Field("Map").MapIndex("dog").Field("Object").Field("value"), uint32(200), nil},

		// Test invalid paths
		{p.Command(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(2)),
			Path:   p.Command(5).Path(),
		}},
		{cA.Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist("cmdTypeMix", "doesnotexist"),
			Path:   cA.Parameter("doesnotexist").Path(),
		}},
		{sB.Field("Ref").Field("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("Complexʳ", "doesnotexist"),
			Path:   sB.Field("Ref").Field("doesnotexist").Path(),
		}},
		{sA.Field("Ref").Field("Strings"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   sA.Field("Ref").Field("Strings").Path(),
		}},
		{sB.Field("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   sB.Field("Sli").ArrayIndex(4).Path(),
		}},
		{sB.Field("Sli").Slice(2, 4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrSliceOutOfBounds(uint64(2), uint64(4), "Start", "End", uint64(0), uint64(2)),
			Path:   sB.Field("Sli").Slice(2, 4).Path(),
		}},
		{sB.Field("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   sB.Field("Str").ArrayIndex(4).Path(),
		}},
		{sB.Field("Ref").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("Complexʳ"),
			Path:   sB.Field("Ref").ArrayIndex(4).Path(),
		}},
		{sB.Field("Ref").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("Complexʳ"),
			Path:   sB.Field("Ref").MapIndex("foo").Path(),
		}},
		{sB.Field("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   sB.Field("Map").MapIndex(10.0).Path(),
		}},
		{sB.Field("Map").MapIndex("rabbit"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrMapKeyDoesNotExist("rabbit"),
			Path:   sB.Field("Map").MapIndex("rabbit").Path(),
		}},
	} {
		got, err := Get(ctx, test.path.Path(), nil)
		assert.For(ctx, "Get(%v) value", test.path).That(got).DeepEquals(test.val)
		assert.For(ctx, "Get(%v) error", test.path).ThatError(err).DeepEquals(test.err)
	}
}

func TestSet(t *testing.T) {
	ctx := log.Testing(t)
	ctx = bind.PutRegistry(ctx, bind.NewRegistry())
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	p := newPathTest(ctx)
	ctx = capture.Put(ctx, p)
	cA, cB := p.Command(0), p.Command(1)
	sA, sB := p.Command(0).StateAfter(), p.Command(2).StateAfter()

	_, _, _, _ = cA, cB, sA, sB
	// Set tests
	for _, test := range []struct {
		path path.Node
		val  interface{}
		err  error
	}{
		{cA.Parameter("ID"), uint64(2), nil},
		{cA.Parameter("U8"), uint8(12), nil},
		{cA.Parameter("S8"), int8(22), nil},
		{cA.Parameter("U16"), uint16(32), nil},
		{cA.Parameter("S16"), int16(42), nil},
		{cA.Parameter("U32"), uint32(52), nil},
		{cA.Parameter("S32"), int32(62), nil},
		{cA.Parameter("U64"), uint64(72), nil},
		{cA.Parameter("S64"), int64(82), nil},
		{cA.Parameter("F32"), float32(92), nil},
		{cA.Parameter("F64"), float64(102), nil},
		{cA.Parameter("Bool"), false, nil},
		{cA.Parameter("Ptr"), test.Voidᵖ(0x111111), nil},
		// {cA.Result(), uint32(5), nil}, // TODO: 'Unknown path type *path.Result'

		{cB.Parameter("ID"), uint64(8), nil},
		{cB.Parameter("U8"), uint8(18), nil},
		{cB.Parameter("S8"), int8(28), nil},
		{cB.Parameter("U16"), uint16(38), nil},
		{cB.Parameter("S16"), int16(48), nil},
		{cB.Parameter("U32"), uint32(58), nil},
		{cB.Parameter("S32"), int32(68), nil},
		{cB.Parameter("U64"), uint64(78), nil},
		{cB.Parameter("S64"), int64(88), nil},
		{cB.Parameter("F32"), float32(98), nil},
		{cB.Parameter("F64"), float64(108), nil},
		{cB.Parameter("Bool"), true, nil},
		{cB.Parameter("Ptr"), test.Voidᵖ(0x2222222), nil},
		// {cB.Result(), uint32(7), nil}, // TODO: 'Unknown path type *path.Result'

		// Test the state cannot be mutated (current restriction)
		{cA.StateAfter(), nil, fmt.Errorf("State can not currently be mutated")},

		// Test invalid paths
		{p.Command(5), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(5), "Index", uint64(0), uint64(2)),
			Path:   p.Command(5).Path(),
		}},
		{cA.Parameter("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist("cmdTypeMix", "doesnotexist"),
			Path:   cA.Parameter("doesnotexist").Path(),
		}},
		{sB.Field("Ref").Field("doesnotexist"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrFieldDoesNotExist("Complexʳ", "doesnotexist"),
			Path:   sB.Field("Ref").Field("doesnotexist").Path(),
		}},
		{sA.Field("Ref").Field("Strings"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrNilPointerDereference(),
			Path:   sA.Field("Ref").Field("Strings").Path(),
		}},
		/* TODO: `<ERR_TYPE_NOT_ARRAY_INDEXABLE [ty: Boolˢ]>`
		{sB.Field("Sli").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   sB.Field("Sli").ArrayIndex(4).Path(),
		}},
		*/
		/* TODO: `Unknown path type *path.Slice`
		{sB.Field("Sli").Slice(2, 4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrSliceOutOfBounds(uint64(2), uint64(4), "Start", "End", uint64(0), uint64(2)),
			Path:   sB.Field("Sli").Slice(2, 4).Path(),
		}},
		*/
		{sB.Field("Str").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(uint64(4), "Index", uint64(0), uint64(2)),
			Path:   sB.Field("Str").ArrayIndex(4).Path(),
		}},
		{sB.Field("Ref").ArrayIndex(4), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotArrayIndexable("Complexʳ"),
			Path:   sB.Field("Ref").ArrayIndex(4).Path(),
		}},
		{sB.Field("Ref").MapIndex("foo"), nil, &service.ErrInvalidPath{
			Reason: messages.ErrTypeNotMapIndexable("Complexʳ"),
			Path:   sB.Field("Ref").MapIndex("foo").Path(),
		}},
		{sB.Field("Map").MapIndex(10.0), nil, &service.ErrInvalidPath{
			Reason: messages.ErrIncorrectMapKeyType("float64", "string"),
			Path:   sB.Field("Map").MapIndex(10.0).Path(),
		}},

		// Test invalid sets
		{sB.Field("Map").MapIndex("bird"), 10.0, fmt.Errorf(
			"Map at capture<%v>.commands[2].state.Map has value of type test.Complexʳ, got type float64", p.ID.ID())},
	} {
		ctx := log.V{"path": test.path, "value": test.val}.Bind(ctx)

		path, err := Set(ctx, test.path.Path(), test.val, nil)
		assert.For(ctx, "Set").ThatError(err).DeepEquals(test.err)

		if (path == nil) == (err == nil) {
			log.E(ctx, "Set returned %T %v and %v.", path, path, err)
		}

		if err == nil {
			// Check the paths have changed
			assert.For(ctx, "Set returned path").That(path).DeepNotEquals(test.path)

			ctx := log.V{"changed_path": path}.Bind(ctx)

			// Get the changed value
			got, err := Get(ctx, path, nil)
			assert.For(ctx, "Get(changed_path) error").ThatError(err).Succeeded()
			ctx = log.V{"got": got}.Bind(ctx)

			// Check it matches what we set it too.
			assert.For(ctx, "Get(changed_path)").That(got).DeepEquals(test.val)
		}
	}
}
