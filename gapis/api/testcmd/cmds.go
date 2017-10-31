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

// Package testcmd provides fake commands used for testing.
package testcmd

import (
	"context"
	"reflect"

	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/testcmd/test_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/box"
)

type A struct {
	ID    api.CmdID
	Flags api.CmdFlags
}

func (a *A) Caller() api.CmdID                                                  { return api.CmdNoID }
func (a *A) SetCaller(api.CmdID)                                                {}
func (a *A) Thread() uint64                                                     { return 1 }
func (a *A) SetThread(uint64)                                                   {}
func (a *A) CmdName() string                                                    { return "A" }
func (a *A) API() api.API                                                       { return nil }
func (a *A) CmdFlags(context.Context, api.CmdID, *api.GlobalState) api.CmdFlags { return a.Flags }
func (a *A) Extras() *api.CmdExtras                                             { return nil }
func (a *A) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder) error {
	return nil
}

type B struct {
	ID   api.CmdID
	Bool bool
}

func (*B) Caller() api.CmdID                                                  { return api.CmdNoID }
func (*B) SetCaller(api.CmdID)                                                {}
func (*B) Thread() uint64                                                     { return 1 }
func (*B) SetThread(uint64)                                                   {}
func (*B) CmdName() string                                                    { return "B" }
func (*B) API() api.API                                                       { return nil }
func (*B) CmdFlags(context.Context, api.CmdID, *api.GlobalState) api.CmdFlags { return 0 }
func (*B) Extras() *api.CmdExtras                                             { return nil }
func (*B) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder) error {
	return nil
}

type (
	Pointer struct {
		addr uint64
		pool memory.PoolID
	}

	StringːString struct{ M map[string]string }

	IntːStructPtr struct{ M map[int]*Struct }

	RawIntːStructPtr map[string]string

	Struct struct {
		Str string
		Ref *Struct
	}
)

var _ data.Assignable = &StringːString{}

func (m StringːString) Dictionary() dictionary.I { return dictionary.From(m.M) }

func (m *StringːString) Assign(v interface{}) bool {
	m.M = map[string]string{}
	return deep.Copy(&m.M, v) == nil
}

var _ data.Assignable = &IntːStructPtr{}

func (m IntːStructPtr) Dictionary() dictionary.I { return dictionary.From(m.M) }

func (m *IntːStructPtr) Assign(v interface{}) bool {
	m.M = map[int]*Struct{}
	return deep.Copy(&m.M, v) == nil
}

// Interface compliance checks
var _ memory.Pointer = &Pointer{}

func (p Pointer) String() string                            { return memory.PointerToString(p) }
func (p Pointer) IsNullptr() bool                           { return p.addr == 0 && p.pool == memory.ApplicationPool }
func (p Pointer) Address() uint64                           { return p.addr }
func (p Pointer) Pool() memory.PoolID                       { return p.pool }
func (p Pointer) Offset(n uint64) memory.Pointer            { panic("not implemented") }
func (p Pointer) ElementSize(m *device.MemoryLayout) uint64 { return 1 }
func (p Pointer) ElementType() reflect.Type                 { return reflect.TypeOf(byte(0)) }
func (p Pointer) ISlice(start, end uint64, m *device.MemoryLayout) memory.Slice {
	panic("not implemented")
}

var _ data.Assignable = &Pointer{}

func (p *Pointer) Assign(o interface{}) bool {
	if o, ok := o.(memory.Pointer); ok {
		*p = Pointer{o.Address(), o.Pool()}
		return true
	}
	return false
}

type X struct {
	Str  string           `param:"Str"`
	Sli  []bool           `param:"Sli"`
	Ref  *Struct          `param:"Ref"`
	Ptr  Pointer          `param:"Ptr"`
	Map  StringːString    `param:"Map"`
	PMap IntːStructPtr    `param:"PMap"`
	RMap RawIntːStructPtr `param:"RMap"`
}

func (X) Caller() api.CmdID                                                  { return api.CmdNoID }
func (X) SetCaller(api.CmdID)                                                {}
func (X) Thread() uint64                                                     { return 1 }
func (X) SetThread(uint64)                                                   {}
func (X) CmdName() string                                                    { return "X" }
func (X) API() api.API                                                       { return api.Find(APIID) }
func (X) CmdFlags(context.Context, api.CmdID, *api.GlobalState) api.CmdFlags { return 0 }
func (X) Extras() *api.CmdExtras                                             { return nil }
func (X) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder) error {
	return nil
}

type API struct{}

func (API) Name() string                 { return "foo" }
func (API) ID() api.ID                   { return APIID }
func (API) Index() uint8                 { return 15 }
func (API) ConstantSets() *constset.Pack { return nil }
func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (width, height, index uint32, format *image.Format, err error) {
	return 0, 0, 0, nil, nil
}
func (API) Context(*api.GlobalState, uint64) api.Context { return nil }
func (API) CreateCmd(name string) api.Cmd {
	switch name {
	case "X":
		return &X{}
	default:
		return nil
	}
}

var (
	APIID = api.ID{1, 2, 3}

	P = &X{
		Str:  "aaa",
		Sli:  []bool{true, false, true},
		Ref:  &Struct{Str: "ccc", Ref: &Struct{Str: "ddd"}},
		Ptr:  Pointer{0x123, 0x456},
		Map:  StringːString{map[string]string{"cat": "meow", "dog": "woof"}},
		PMap: IntːStructPtr{map[int]*Struct{}},
		RMap: map[string]string{"eyes": "see", "nose": "smells"},
	}

	Q = &X{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Ptr: Pointer{0x321, 0x654},
		Map: StringːString{map[string]string{"bird": "tweet", "fox": "?"}},
		PMap: IntːStructPtr{map[int]*Struct{
			100: &Struct{Str: "baldrick"},
		}},
		RMap: map[string]string{"ears": "hear", "tongue": "taste"},
	}
)

func init() {
	api.Register(API{})
	protoconv.Register(func(ctx context.Context, a *X) (*test_pb.X, error) {
		return &test_pb.X{Data: box.NewValue(*a)}, nil
	}, func(ctx context.Context, b *test_pb.X) (*X, error) {
		var a X
		if err := b.Data.AssignTo(&a); err != nil {
			return nil, err
		}
		return &a, nil
	})
}
