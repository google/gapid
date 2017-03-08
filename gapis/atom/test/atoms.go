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

	"github.com/google/gapid/framework/binary"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
)

type AtomA struct {
	binary.Generate `id:"AtomAID"`
	ID              atom.ID
	Flags           atom.Flags
}

func (a *AtomA) API() gfxapi.API       { return nil }
func (a *AtomA) AtomFlags() atom.Flags { return a.Flags }
func (a *AtomA) Extras() *atom.Extras  { return nil }
func (a *AtomA) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type AtomB struct {
	binary.Generate `id:"AtomBID"`
	ID              atom.ID
	Bool            bool
}

func (a *AtomB) API() gfxapi.API       { return nil }
func (a *AtomB) AtomFlags() atom.Flags { return 0 }
func (a *AtomB) Extras() *atom.Extras  { return nil }
func (a *AtomB) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}

type AtomC struct {
	binary.Generate `id:"AtomCID"`
	String          string
}

func (a *AtomC) API() gfxapi.API       { return nil }
func (a *AtomC) AtomFlags() atom.Flags { return 0 }
func (a *AtomC) Extras() *atom.Extras  { return nil }
func (a *AtomC) Mutate(context.Context, *gfxapi.State, *builder.Builder) error {
	return nil
}
