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

package gles

import (
	"context"

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

// ctxIDKey is an object identifier and context pair used for a remapping key.
// Ideally we'd just use the object or object pointer as the key, but we have
// atoms that want to remap the identifier before the state object is created.
// TODO: It maybe possible to rework the state-mutator and/or APIs to achieve
// this.
type ctxIDKey struct {
	id  interface{}
	ctx *Context
}

func (i BufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i FramebufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i RenderbufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i ProgramId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i ShaderId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i TextureId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i UniformBlockId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	program := ctx.BoundProgram
	switch a := a.(type) {
	case *GlGetActiveUniformBlockName:
		program = a.Program
	case *GlGetActiveUniformBlockiv:
		program = a.Program
	case *GlGetUniformBlockIndex:
		program = a.Program
	case *GlUniformBlockBinding:
		program = a.Program
	}
	return struct {
		p *Program
		i UniformBlockId
	}{ctx.Instances.Programs[program], i}, true
}

func (i VertexArrayId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i QueryId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i != 0 {
		key, remap = ctxIDKey{i, GetContext(s)}, true
	}
	return
}

func (i GLsync) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	if i.Address != 0 {
		key, remap = ctxIDKey{i.Address, GetContext(s)}, true
	}
	return
}

func (i GLsync) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	return value.AbsolutePointer(i.Address)
}

func (i UniformLocation) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	program := ctx.BoundProgram
	switch a := a.(type) {
	case *GlGetActiveUniform:
		program = a.Program
	case *GlGetUniformLocation:
		program = a.Program
	}
	return struct {
		p *Program
		l UniformLocation
	}{ctx.Instances.Programs[program], i}, true
}

func (i IndicesPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	c := GetContext(s)
	if c.Instances.VertexArrays[c.BoundVertexArray].ElementArrayBuffer != 0 {
		return value.AbsolutePointer(i.Address)
	} else {
		return value.ObservedPointer(i.Address)
	}
}

func (i VertexPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	if GetContext(s).BoundBuffers.ArrayBuffer != 0 {
		return value.AbsolutePointer(i.Address)
	} else {
		return value.ObservedPointer(i.Address)
	}
}

func (i TexturePointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	if i.Address == 0 || GetContext(s).BoundBuffers.PixelUnpackBuffer != 0 {
		return value.AbsolutePointer(i.Address)
	} else {
		return value.ObservedPointer(i.Address)
	}
}

func (i BufferDataPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	if i.Address == 0 {
		return value.AbsolutePointer(i.Address)
	} else {
		return value.ObservedPointer(i.Address)
	}
}

func (i GLeglImageOES) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	return value.AbsolutePointer(i.Address)
}

func OnSwitchThread(ctx context.Context, gs *gfxapi.State, b *builder.Builder) error {
	s := GetState(gs)
	context := s.Contexts[s.CurrentThread]
	if context == nil {
		return nil
	}
	ctxID := uint32(context.Identifier)
	return NewReplayBindRenderer(ctxID).Mutate(ctx, gs, b)
}

func (ω *EglCreateContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).EGLContexts[ω.Result].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *EglMakeCurrent) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	_, wasCreated := GetState(s).EGLContexts[ω.Context]
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Context.Address == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).EGLContexts[ω.Context].Identifier)
	if !wasCreated {
		// The eglCreateContext call was missing, so fake it (can happen on Samsung).
		if err := NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b); err != nil {
			return err
		}
	}
	if err := NewReplayBindRenderer(ctxID).Mutate(ctx, s, b); err != nil {
		return err
	}
	if cs := FindDynamicContextState(ω.Extras()); cs != nil {
		cmd := NewReplayChangeBackbuffer(
			cs.BackbufferWidth,
			cs.BackbufferHeight,
			cs.BackbufferColorFmt,
			cs.BackbufferDepthFmt,
			cs.BackbufferStencilFmt,
			cs.ResetViewportScissor,
		)
		if err := cmd.Mutate(ctx, s, b); err != nil {
			return err
		}
	}
	return nil
}

func (ω *WglCreateContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).WGLContexts[ω.Result].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *WglCreateContextAttribsARB) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).WGLContexts[ω.Result].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *WglMakeCurrent) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Hglrc.Address == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).WGLContexts[ω.Hglrc].Identifier)
	return NewReplayBindRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *CGLCreateContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).CGLContexts[ω.Ctx.Read(ctx, ω, s, b)].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *CGLSetCurrentContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Ctx.Address == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).CGLContexts[ω.Ctx].Identifier)
	return NewReplayBindRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *GlXCreateContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).GLXContexts[ω.Result].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *GlXCreateNewContext) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).GLXContexts[ω.Result].Identifier)
	return NewReplayCreateRenderer(ctxID).Mutate(ctx, s, b)
}

func (ω *GlXMakeContextCurrent) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	err := ω.mutate(ctx, s, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Ctx.Address == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).GLXContexts[ω.Ctx].Identifier)
	return NewReplayBindRenderer(ctxID).Mutate(ctx, s, b)
}

// Force all attributes to use the capture-observed locations during replay.
func bindAttribLocations(ctx context.Context, a atom.Atom, s *gfxapi.State, b *builder.Builder, pid ProgramId) error {
	pi := FindProgramInfo(a.Extras())
	if pi != nil && b != nil {
		for _, attr := range pi.ActiveAttributes {
			a := NewGlBindAttribLocation(pid, AttributeLocation(attr.Location), attr.Name)
			if err := a.Mutate(ctx, s, b); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *GlProgramBinaryOES) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	if err := bindAttribLocations(ctx, a, s, b, a.Program); err != nil {
		return err
	}
	return a.mutate(ctx, s, b)
}

func (a *GlLinkProgram) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	if err := bindAttribLocations(ctx, a, s, b, a.Program); err != nil {
		return err
	}
	return a.mutate(ctx, s, b)
}

func (a *GlProgramBinary) Mutate(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
	if err := bindAttribLocations(ctx, a, s, b, a.Program); err != nil {
		return err
	}
	return a.mutate(ctx, s, b)
}
