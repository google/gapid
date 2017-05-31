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

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapil/constset"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test/test_pb"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/box"
)

type AtomA struct {
	ID    atom.ID
	Flags atom.Flags
}

func (a *AtomA) AtomName() string      { return "AtomA" }
func (a *AtomA) API() gfxapi.API       { return nil }
func (a *AtomA) AtomFlags() atom.Flags { return a.Flags }
func (a *AtomA) Extras() *atom.Extras  { return nil }
func (a *AtomA) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type AtomB struct {
	ID   atom.ID
	Bool bool
}

func (a *AtomB) AtomName() string      { return "AtomB" }
func (a *AtomB) API() gfxapi.API       { return nil }
func (a *AtomB) AtomFlags() atom.Flags { return 0 }
func (a *AtomB) Extras() *atom.Extras  { return nil }
func (a *AtomB) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type AtomC struct {
	String string
}

func (a *AtomC) AtomName() string      { return "AtomC" }
func (a *AtomC) API() gfxapi.API       { return nil }
func (a *AtomC) AtomFlags() atom.Flags { return 0 }
func (a *AtomC) Extras() *atom.Extras  { return nil }
func (a *AtomC) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type (
	stringːstring map[string]string

	intːStructPtr map[int]*Struct

	Struct struct {
		Str string
		Ref *Struct
	}
)

type AtomX struct {
	Str  string         `param:"Str"`
	Sli  []bool         `param:"Sli"`
	Ref  *Struct        `param:"Ref"`
	Ptr  memory.Pointer `param:"Ptr"`
	Map  stringːstring  `param:"Map"`
	PMap intːStructPtr  `param:"PMap"`
}

func (AtomX) AtomName() string      { return "AtomX" }
func (AtomX) API() gfxapi.API       { return gfxapi.Find(APIID) }
func (AtomX) AtomFlags() atom.Flags { return 0 }
func (AtomX) Extras() *atom.Extras  { return nil }
func (AtomX) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type API struct{}

func (API) Name() string                 { return "foo" }
func (API) ID() gfxapi.ID                { return APIID }
func (API) Index() uint8                 { return 15 }
func (API) ConstantSets() *constset.Pack { return nil }
func (API) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (uint32, uint32, *image.Format, error) {
	return 0, 0, nil, nil
}
func (API) Context(*gfxapi.State) gfxapi.Context { return nil }

var (
	APIID = gfxapi.ID{1, 2, 3}

	P = &AtomX{
		Str:  "aaa",
		Sli:  []bool{true, false, true},
		Ref:  &Struct{Str: "ccc", Ref: &Struct{Str: "ddd"}},
		Ptr:  memory.BytePtr(0x123, 0x456),
		Map:  stringːstring{"cat": "meow", "dog": "woof"},
		PMap: intːStructPtr{},
	}

	Q = &AtomX{
		Str: "xyz",
		Sli: []bool{false, true, false},
		Ptr: memory.BytePtr(0x321, 0x654),
		Map: stringːstring{"bird": "tweet", "fox": "?"},
		PMap: intːStructPtr{
			100: &Struct{Str: "baldrick"},
		},
	}
)

func init() {
	gfxapi.Register(API{})
	atom.Register(API{}, &AtomX{})
	protoconv.Register(func(ctx context.Context, a *AtomX) (*test_pb.AtomX, error) {
		return &test_pb.AtomX{Data: box.NewValue(a)}, nil
	}, func(ctx context.Context, b *test_pb.AtomX) (*AtomX, error) {
		var a AtomX
		if err := b.Data.AssignTo(&a); err != nil {
			return nil, err
		}
		return &a, nil
	})
}
