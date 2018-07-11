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
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

// objectKey is a map and object identifier pair used for a remapping key.
// Ideally we'd just use the object or object pointer as the key, but we have
// commands that want to remap the identifier before the state object is
// created.
// TODO: It maybe possible to rework the state-mutator and/or APIs to achieve
// this.
type objectKey struct {
	mapPtr interface{}
	mapKey interface{}
}

func (i BufferId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Buffers(), i}, true
	}
	return
}

func (i FramebufferId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Framebuffers(), i}, true
	}
	return
}

func (i RenderbufferId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Renderbuffers(), i}, true
	}
	return
}

func (i ProgramId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Programs(), i}, true
	}
	return
}

func (i ShaderId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Shaders(), i}, true
	}
	return
}

func (i TextureId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		if tex := ctx.Objects().Textures().Get(i); !tex.IsNil() {
			_, isDeleteCmd := cmd.(*GlDeleteTextures)
			if eglImage := tex.EGLImage(); !eglImage.IsNil() && !isDeleteCmd {
				// Ignore this texture and use the data that EGLImage points to.
				// (unless it is a delete command - we do not want kill the shared data)
				ctx := GetState(s).EGLContexts().Get(eglImage.Context())
				isTex2D := eglImage.Target() == EGLenum_EGL_GL_TEXTURE_2D
				i := TextureId(eglImage.Buffer().Address())
				if !ctx.IsNil() && isTex2D && ctx.Objects().Textures().Contains(i) {
					return objectKey{ctx.Objects().Textures(), i}, true
				}
				panic(fmt.Errorf("Can not find EGL image target: %v", eglImage))
			}
		}
		key, remap = objectKey{ctx.Objects().Textures(), i}, true
	}
	return
}

func (i UniformBlockIndex) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	program := ctx.Bound().Program().GetID()
	switch cmd := cmd.(type) {
	case *GlGetActiveUniformBlockName:
		program = cmd.Program()
	case *GlGetActiveUniformBlockiv:
		program = cmd.Program()
	case *GlGetUniformBlockIndex:
		program = cmd.Program()
	case *GlUniformBlockBinding:
		program = cmd.Program()
	}
	return struct {
		p Programʳ
		i UniformBlockIndex
	}{ctx.Objects().Programs().Get(program), i}, true
}

func (i VertexArrayId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().VertexArrays(), i}, true
	}
	return
}

func (i QueryId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Queries(), i}, true
	}
	return
}

func (i GLsync) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && !i.IsNullptr() {
		key, remap = objectKey{ctx.Objects().SyncObjects(), i}, true
	}
	return
}

func (i GLsync) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	return value.AbsolutePointer(i)
}

func (i SamplerId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Samplers(), i}, true
	}
	return
}

func (i PipelineId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().Pipelines(), i}, true
	}
	return
}

func (i TransformFeedbackId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && i != 0 {
		key, remap = objectKey{ctx.Objects().TransformFeedbacks(), i}, true
	}
	return
}

func (i UniformLocation) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	program := ctx.Bound().Program().GetID()
	switch cmd := cmd.(type) {
	case *GlGetActiveUniform:
		program = cmd.Program()
	case *GlGetUniformLocation:
		program = cmd.Program()
	}
	return struct {
		p Programʳ
		l UniformLocation
	}{ctx.Objects().Programs().Get(program), i}, true
}

func (i SrcImageId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	c, ok := cmd.(interface {
		api.Cmd
		SrcName() SrcImageId
		SrcTarget() GLenum
	})
	if !ok {
		panic(fmt.Errorf("Remap of SrcImageId for unhandled command: %v", cmd))
	}
	return remapImageID(c, s, GLuint(c.SrcName()), c.SrcTarget())
}

func (i DstImageId) remap(cmd api.Cmd, s *api.GlobalState) (key interface{}, remap bool) {
	c, ok := cmd.(interface {
		api.Cmd
		DstName() DstImageId
		DstTarget() GLenum
	})
	if !ok {
		panic(fmt.Errorf("Remap of DstImageId for unhandled command: %v", cmd))
	}
	return remapImageID(c, s, GLuint(c.DstName()), c.DstTarget())
}

func remapImageID(cmd api.Cmd, s *api.GlobalState, name GLuint, target GLenum) (key interface{}, remap bool) {
	ctx := GetContext(s, cmd.Thread())
	if !ctx.IsNil() && name != 0 {
		if target == GLenum_GL_RENDERBUFFER {
			return RenderbufferId(name).remap(cmd, s)
		}
		return TextureId(name).remap(cmd, s)
	}
	return
}

func (i IndicesPointer) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	c := GetContext(s, cmd.Thread())
	if !c.Bound().VertexArray().ElementArrayBuffer().IsNil() {
		return value.AbsolutePointer(i)
	}
	return value.ObservedPointer(i)
}

func (i VertexPointer) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	c := GetContext(s, cmd.Thread())
	if !c.Bound().ArrayBuffer().IsNil() {
		return value.AbsolutePointer(i)
	}
	return value.ObservedPointer(i)
}

func (i TexturePointer) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	if i == 0 || !GetContext(s, cmd.Thread()).Bound().PixelUnpackBuffer().IsNil() {
		return value.AbsolutePointer(i)
	}
	return value.ObservedPointer(i)
}

func (i BufferDataPointer) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	if i == 0 {
		return value.AbsolutePointer(i)
	}
	return value.ObservedPointer(i)
}

func (i GLeglImageOES) value(b *builder.Builder, cmd api.Cmd, s *api.GlobalState) value.Value {
	return value.AbsolutePointer(i)
}

func (ω *EglCreateContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).EGLContexts().Get(ω.Result()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *EglMakeCurrent) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	prevContext := GetState(s).Contexts().Get(ω.Thread())
	existed := GetState(s).EGLContexts().Contains(ω.Context())
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	if ω.Context() == 0 {
		if prevContext.IsNil() {
			return nil
		}
		ctxID := uint32(prevContext.Identifier())
		return cb.ReplayUnbindRenderer(ctxID).Mutate(ctx, id, s, b, w)
	}
	ctxID := uint32(GetState(s).EGLContexts().Get(ω.Context()).Identifier())
	if !existed {
		// The eglCreateContext call was missing, so fake it (can happen on Samsung).
		if err := cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w); err != nil {
			return err
		}
	}
	if err := cb.ReplayBindRenderer(ctxID).Mutate(ctx, id, s, b, w); err != nil {
		return err
	}
	if cs := FindDynamicContextState(s.Arena, ω.Extras()); !cs.IsNil() {
		cmd := cb.ReplayChangeBackbuffer(
			ctxID,
			cs.BackbufferWidth(),
			cs.BackbufferHeight(),
			cs.BackbufferColorFmt(),
			cs.BackbufferDepthFmt(),
			cs.BackbufferStencilFmt(),
			cs.ResetViewportScissor(),
		)
		if err := cmd.Mutate(ctx, id, s, b, w); err != nil {
			return err
		}
	}
	return nil
}

func (ω *WglCreateContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).WGLContexts().Get(ω.Result()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *WglCreateContextAttribsARB) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).WGLContexts().Get(ω.Result()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *WglMakeCurrent) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Hglrc() == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).WGLContexts().Get(ω.Hglrc()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayBindRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *CGLCreateContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).CGLContexts().Get(ω.Ctx().MustRead(ctx, ω, s, b, nil)).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *CGLSetCurrentContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Ctx() == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).CGLContexts().Get(ω.Ctx()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayBindRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *GlXCreateContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).GLXContexts().Get(ω.Result()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *GlXCreateNewContext) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	ctxID := uint32(GetState(s).GLXContexts().Get(ω.Result()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayCreateRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

func (ω *GlXMakeContextCurrent) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	err := ω.mutate(ctx, id, s, nil, nil)
	if b == nil || err != nil {
		return err
	}
	if ω.Ctx() == 0 {
		return nil
	}
	ctxID := uint32(GetState(s).GLXContexts().Get(ω.Ctx()).Identifier())
	cb := CommandBuilder{Thread: ω.Thread(), Arena: s.Arena}
	return cb.ReplayBindRenderer(ctxID).Mutate(ctx, id, s, b, w)
}

// Force all attributes to use the capture-observed locations during replay.
func bindAttribLocations(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher, pid ProgramId) error {
	pi := FindLinkProgramExtra(s.Arena, cmd.Extras())
	if !pi.IsNil() && b != nil && !pi.ActiveResources().IsNil() {
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		for _, attr := range pi.ActiveResources().ProgramInputs().All() {
			if int32(attr.Locations().Get(0)) != -1 {
				tmp := s.AllocDataOrPanic(ctx, attr.Name())
				defer tmp.Free()
				cmd := cb.GlBindAttribLocation(pid, AttributeLocation(attr.Locations().Get(0)), tmp.Ptr()).
					AddRead(tmp.Data())
				if strings.HasPrefix(attr.Name(), "gl_") {
					// Active built-in mush have location of -1
					log.E(ctx, "Can not set location for built-in attribute: %v", cmd)
					continue
				}
				if err := cmd.Mutate(ctx, id, s, b, w); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Remap uniform block indices
func bindUniformBlocks(ctx context.Context, cmd api.Cmd, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher, pid ProgramId) error {
	pi := FindLinkProgramExtra(s.Arena, cmd.Extras())
	if !pi.IsNil() && b != nil && !pi.ActiveResources().IsNil() {
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		for i, ub := range pi.ActiveResources().UniformBlocks().All() {
			// Query replay-time uniform block index so that the remapping is established
			tmp := s.AllocDataOrPanic(ctx, ub.Name())
			defer tmp.Free()
			cmd := cb.GlGetUniformBlockIndex(pid, tmp.Ptr(), UniformBlockIndex(i)).
				AddRead(tmp.Data())
			if err := cmd.Mutate(ctx, id, s, b, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (cmd *GlProgramBinaryOES) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	if err := bindAttribLocations(ctx, cmd, id, s, b, w, cmd.Program()); err != nil {
		return err
	}
	if err := cmd.mutate(ctx, id, s, b, w); err != nil {
		return err
	}
	return bindUniformBlocks(ctx, cmd, id, s, b, w, cmd.Program())
}

func (cmd *GlLinkProgram) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	if err := bindAttribLocations(ctx, cmd, id, s, b, w, cmd.Program()); err != nil {
		return err
	}
	if err := cmd.mutate(ctx, id, s, b, w); err != nil {
		return err
	}
	return bindUniformBlocks(ctx, cmd, id, s, b, w, cmd.Program())
}

func (cmd *GlProgramBinary) Mutate(ctx context.Context, id api.CmdID, s *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	if err := bindAttribLocations(ctx, cmd, id, s, b, w, cmd.Program()); err != nil {
		return err
	}
	if err := cmd.mutate(ctx, id, s, b, w); err != nil {
		return err
	}
	return bindUniformBlocks(ctx, cmd, id, s, b, w, cmd.Program())
}
