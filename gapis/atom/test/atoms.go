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

// Package test provides testing helpers for the atom package.
package test

import (
	"context"
	"reflect"

	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test/test_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/box"
)

type AtomA struct {
	ID    atom.ID
	Flags api.CmdFlags
}

func (a *AtomA) Thread() uint64         { return 1 }
func (a *AtomA) SetThread(uint64)       {}
func (a *AtomA) CmdName() string        { return "AtomA" }
func (a *AtomA) API() api.API           { return nil }
func (a *AtomA) CmdFlags() api.CmdFlags { return a.Flags }
func (a *AtomA) Extras() *api.CmdExtras { return nil }
func (a *AtomA) Mutate(context.Context, *api.State, *builder.Builder) error {
	return nil
}

type AtomB struct {
	ID   atom.ID
	Bool bool
}

func (a *AtomB) Thread() uint64         { return 1 }
func (a *AtomB) SetThread(uint64)       {}
func (a *AtomB) CmdName() string        { return "AtomB" }
func (a *AtomB) API() api.API           { return nil }
func (a *AtomB) CmdFlags() api.CmdFlags { return 0 }
func (a *AtomB) Extras() *api.CmdExtras { return nil }
func (a *AtomB) Mutate(context.Context, *api.State, *builder.Builder) error {
	return nil
}

type AtomC struct {
	String string
}

func (a *AtomC) Thread() uint64         { return 1 }
func (a *AtomC) SetThread(uint64)       {}
func (a *AtomC) CmdName() string        { return "AtomC" }
func (a *AtomC) API() api.API           { return nil }
func (a *AtomC) CmdFlags() api.CmdFlags { return 0 }
func (a *AtomC) Extras() *api.CmdExtras { return nil }
func (a *AtomC) Mutate(context.Context, *api.State, *builder.Builder) error {
	return nil
}

type (
	Pointer struct {
		addr uint64
		pool memory.PoolID
	}

	StringːString map[string]string

	IntːStructPtr map[int]*Struct

	Struct struct {
		Str string
		Ref *Struct
	}
)

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

type AtomX struct {
	Str  string        `param:"Str"`
	Sli  []bool        `param:"Sli"`
	Ref  *Struct       `param:"Ref"`
	Ptr  Pointer       `param:"Ptr"`
	Map  StringːString `param:"Map"`
	PMap IntːStructPtr `param:"PMap"`
}

func (AtomX) Thread() uint64         { return 1 }
func (AtomX) SetThread(uint64)       {}
func (AtomX) CmdName() string        { return "AtomX" }
func (AtomX) API() api.API           { return api.Find(APIID) }
func (AtomX) CmdFlags() api.CmdFlags { return 0 }
func (AtomX) Extras() *api.CmdExtras { return nil }
func (AtomX) Mutate(context.Context, *api.State, *builder.Builder) error {
	return nil
}

type API struct{}

func (API) Name() string                 { return "foo" }
func (API) ID() api.ID                   { return APIID }
func (API) Index() uint8                 { return 15 }
func (API) ConstantSets() *constset.Pack { return nil }
func (API) GetFramebufferAttachmentInfo(*api.State, uint64, api.FramebufferAttachment) (uint32, uint32, uint32, *image.Format, error) {
	return 0, 0, 0, nil, nil
}
func (API) Context(*api.State, uint64) api.Context { return nil }

var (
	APIID = api.ID{1, 2, 3}

	P = &AtomX{
		Str:  "aaa",
		Sli:  []bool{true, false, true},
		Ref:  &Struct{Str: "ccc", Ref: &Struct{Str: "ddd"}},
		Ptr:  Pointer{0x123, 0x456},
		Map:  StringːString{"cat": "meow", "dog": "woof"},
		PMap: IntːStructPtr{},
	}

	Q = &AtomX{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Ptr: Pointer{0x321, 0x654},
		Map: StringːString{"bird": "tweet", "fox": "?"},
		PMap: IntːStructPtr{
			100: &Struct{Str: "baldrick"},
		},
	}
)

func init() {
	api.Register(API{})
	atom.Register(API{}, &AtomX{})
	protoconv.Register(func(ctx context.Context, a *AtomX) (*test_pb.AtomX, error) {
		return &test_pb.AtomX{Data: box.NewValue(*a)}, nil
	}, func(ctx context.Context, b *test_pb.AtomX) (*AtomX, error) {
		var a AtomX
		if err := b.Data.AssignTo(&a); err != nil {
			return nil, err
		}
		return &a, nil
	})
}
