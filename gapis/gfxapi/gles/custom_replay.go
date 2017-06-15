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
	"fmt"

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

// objectKey is a map and object identifier pair used for a remapping key.
// Ideally we'd just use the object or object pointer as the key, but we have
// atoms that want to remap the identifier before the state object is created.
// TODO: It maybe possible to rework the state-mutator and/or APIs to achieve
// this.
type objectKey struct {
	mapPtr interface{}
	mapKey interface{}
}

func (i BufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Buffers, i}, true
	}
	return
}

func (i FramebufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Framebuffers, i}, true
	}
	return
}

func (i RenderbufferId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Renderbuffers, i}, true
	}
	return
}

func (i ProgramId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Programs, i}, true
	}
	return
}

func (i ShaderId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Shaders, i}, true
	}
	return
}

func (i TextureId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Textures, i}, true
	}
	return
}

func (i UniformBlockIndex) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	program := ctx.Bound.Program.GetID()
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
		i UniformBlockIndex
	}{ctx.Objects.Shared.Programs[program], i}, true
}

func (i VertexArrayId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.VertexArrays, i}, true
	}
	return
}

func (i QueryId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Queries, i}, true
	}
	return
}

func (i GLsync) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && !i.IsNullptr() {
		key, remap = objectKey{&ctx.Objects.Shared.SyncObjects, i}, true
	}
	return
}

func (i GLsync) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	return value.AbsolutePointer(i.addr)
}

func (i SamplerId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Shared.Samplers, i}, true
	}
	return
}

func (i PipelineId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.Pipelines, i}, true
	}
	return
}

func (i TransformFeedbackId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && i != 0 {
		key, remap = objectKey{&ctx.Objects.TransformFeedbacks, i}, true
	}
	return
}

func (i UniformLocation) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	program := ctx.Bound.Program.GetID()
	switch a := a.(type) {
	case *GlGetActiveUniform:
		program = a.Program
	case *GlGetUniformLocation:
		program = a.Program
	}
	return struct {
		p *Program
		l UniformLocation
	}{ctx.Objects.Shared.Programs[program], i}, true
}

func (i SrcImageId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	switch a := a.(type) {
	case *GlCopyImageSubData:
		return remapImageId(GLuint(a.SrcName), a.SrcTarget, s)
	case *GlCopyImageSubDataEXT:
		return remapImageId(GLuint(a.SrcName), a.SrcTarget, s)
	case *GlCopyImageSubDataOES:
		return remapImageId(GLuint(a.SrcName), a.SrcTarget, s)
	default:
		panic(fmt.Errorf("Remap of SrcImageId for unhandeled atom: %v", a))
	}
	return
}

func (i DstImageId) remap(a atom.Atom, s *gfxapi.State) (key interface{}, remap bool) {
	switch a := a.(type) {
	case *GlCopyImageSubData:
		return remapImageId(GLuint(a.DstName), a.DstTarget, s)
	case *GlCopyImageSubDataEXT:
		return remapImageId(GLuint(a.DstName), a.DstTarget, s)
	case *GlCopyImageSubDataOES:
		return remapImageId(GLuint(a.DstName), a.DstTarget, s)
	default:
		panic(fmt.Errorf("Remap of DstImageId for unhandeled atom: %v", a))
	}
	return
}

func remapImageId(name GLuint, target GLenum, s *gfxapi.State) (key interface{}, remap bool) {
	ctx := GetContext(s)
	if ctx != nil && name != 0 {
		if target == GLenum_GL_RENDERBUFFER {
			key, remap = objectKey{&ctx.Objects.Shared.Renderbuffers, RenderbufferId(name)}, true
		} else {
			key, remap = objectKey{&ctx.Objects.Shared.Textures, TextureId(name)}, true
		}
	}
	return
}

func (i IndicesPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	c := GetContext(s)
	if c.Bound.VertexArray.ElementArrayBuffer != nil {
		return value.AbsolutePointer(i.addr)
	} else {
		return value.ObservedPointer(i.addr)
	}
}

func (i VertexPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	c := GetContext(s)
	if c.Bound.ArrayBuffer != nil {
		return value.AbsolutePointer(i.addr)
	} else {
		return value.ObservedPointer(i.addr)
	}
}

func (i TexturePointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	if i.addr == 0 || GetContext(s).Bound.PixelUnpackBuffer != nil {
		return value.AbsolutePointer(i.addr)
	} else {
		return value.ObservedPointer(i.addr)
	}
}

func (i BufferDataPointer) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	if i.addr == 0 {
		return value.AbsolutePointer(i.addr)
	} else {
		return value.ObservedPointer(i.addr)
	}
}

func (i GLeglImageOES) value(b *builder.Builder, a atom.Atom, s *gfxapi.State) value.Value {
	return value.AbsolutePointer(i.addr)
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
	if ω.Context.addr == 0 {
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
	if ω.Hglrc.addr == 0 {
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
	if ω.Ctx.addr == 0 {
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
	if ω.Ctx.addr == 0 {
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
