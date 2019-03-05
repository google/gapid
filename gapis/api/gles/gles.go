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
	"github.com/google/gapid/core/stream"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service/path"
)

type customState struct{}

func (customState) init(*State) {}

func GetContext(s *api.GlobalState, thread uint64) Contextʳ {
	return GetState(s).GetContext(thread)
}

func (s *State) GetContext(thread uint64) Contextʳ {
	return s.Contexts().Get(thread)
}

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (s *State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	if p.Context == nil || !p.Context.IsValid() {
		return p, nil
	}
	c, err := resolve.Context(ctx, p.After.Capture.Context(p.Context.ID()), r)
	if err != nil {
		return nil, err
	}
	for thread, context := range s.Contexts().All() {
		if c.ID == context.ID() {
			return s.contextRoot(p.After, thread), nil
		}
	}
	return nil, nil
}

// SetupInitialState sanitizes deserialized state to make it valid.
// It can fill in any derived data which we choose not to serialize, or it can
// apply backward-compatibility fixes for older traces.
func (s *State) SetupInitialState(ctx context.Context) {
	a := s.Arena()
	contexts := NewU64ːContextʳᵐ(a)
	contexts.Add(0, NilContextʳ)
	s.SetContexts(contexts)
	s.SetGLXContexts(NewGLXContextːContextʳᵐ(a))
	s.SetWGLContexts(NewHGLRCːContextʳᵐ(a))
	s.SetCGLContexts(NewCGLContextObjːContextʳᵐ(a))
	for _, c := range s.EGLContexts().All() {
		if t := c.Other().BoundOnThread(); t != 0 {
			s.Contexts().Add(t, c) // Current thread bindings.
		}
		if id := c.Identifier(); id >= s.NextContextID() {
			s.SetNextContextID(id + 1)
		}
	}
}

func (s *State) contextRoot(p *path.Command, thread uint64) *path.MapIndex {
	return path.NewField("Contexts", resolve.APIStateAfter(p, ID)).MapIndex(thread)
}

func (s *State) objectsRoot(p *path.Command, thread uint64) *path.Field {
	return s.contextRoot(p, thread).Field("Objects")
}

func (c *State) preMutate(ctx context.Context, s *api.GlobalState, cmd api.Cmd) error {
	c.SetCurrentContext(c.GetContext(cmd.Thread()))
	// TODO: Find better way to separate GL and EGL commands.
	if c.CurrentContext().IsNil() && strings.HasPrefix(cmd.CmdName(), "gl") {
		if f := s.NewMessage; f != nil {
			f(log.Error, messages.ErrNoContextBound(cmd.Thread()))
		}
		return &api.ErrCmdAborted{Reason: "No context bound"}
	}
	if !c.CurrentContext().IsNil() {
		c.SetVersion(c.CurrentContext().Other().SupportedVersions())
		c.SetExtension(c.CurrentContext().Other().SupportedExtensions())
	} else {
		c.SetVersion(NilSupportedVersionsʳ)
		c.SetExtension(NilSupportedExtensionsʳ)
	}
	return nil
}

func (b Bufferʳ) GetID() BufferId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (b Framebufferʳ) GetID() FramebufferId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (b Renderbufferʳ) GetID() RenderbufferId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (b Programʳ) GetID() ProgramId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (o Shaderʳ) GetID() ShaderId {
	if !o.IsNil() {
		return o.ID()
	}
	return 0
}

func (b VertexArrayʳ) GetID() VertexArrayId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (b Textureʳ) GetID() TextureId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (b ImageUnitʳ) GetID() ImageUnitId {
	if !b.IsNil() {
		return b.ID()
	}
	return 0
}

func (o Samplerʳ) GetID() SamplerId {
	if !o.IsNil() {
		return o.ID()
	}
	return 0
}

func (o Queryʳ) GetID() QueryId {
	if !o.IsNil() {
		return o.ID()
	}
	return 0
}

func (o Pipelineʳ) GetID() PipelineId {
	if !o.IsNil() {
		return o.ID()
	}
	return 0
}

func (o TransformFeedbackʳ) GetID() TransformFeedbackId {
	if !o.IsNil() {
		return o.ID()
	}
	return 0
}

// GetFramebufferAttachmentInfo returns the width, height and format of the
// specified attachment of the currently bound framebuffer.
func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (inf api.FramebufferAttachmentInfo, err error) {

	return GetFramebufferAttachmentInfoByID(state, thread, attachment, 0)
}

// GetFramebufferAttachmentInfoByID returns the width, height and format of the
// specified attachment of the framebuffer with the given id.
// If fb is 0 then the currently bound framebuffer is used.
func GetFramebufferAttachmentInfoByID(
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment,
	fb FramebufferId) (inf api.FramebufferAttachmentInfo, err error) {

	s := GetState(state)

	if fb == 0 {
		c := s.GetContext(thread)
		if c.IsNil() {
			return api.FramebufferAttachmentInfo{}, fmt.Errorf("No context bound")
		}
		if !c.Other().Initialized() {
			return api.FramebufferAttachmentInfo{}, fmt.Errorf("Context not initialized")
		}
		fb = c.Bound().DrawFramebuffer().GetID()
	}

	glAtt, err := attachmentToEnum(attachment)
	if err != nil {
		return api.FramebufferAttachmentInfo{}, err
	}

	fbai, err := s.getFramebufferAttachmentInfo(thread, fb, glAtt)
	if fbai.sizedFormat == 0 {
		return api.FramebufferAttachmentInfo{}, fmt.Errorf("No format set")
	}
	if err != nil {
		return api.FramebufferAttachmentInfo{}, err
	}
	fmt, ty := getUnsizedFormatAndType(fbai.sizedFormat)
	f, err := getImageFormat(fmt, ty)
	if err != nil {
		return api.FramebufferAttachmentInfo{}, err
	}
	switch {
	case attachment.IsDepth():
		f = filterUncompressedImageFormat(f, stream.Channel.IsDepth)
	case attachment.IsStencil():
		f = filterUncompressedImageFormat(f, stream.Channel.IsStencil)
	}
	return api.FramebufferAttachmentInfo{fbai.width, fbai.height, 0, f, true}, nil
}

// Context returns the active context for the given state and thread.
func (API) Context(ctx context.Context, s *api.GlobalState, thread uint64) api.Context {
	if c := GetContext(s, thread); !c.IsNil() {
		return c
	}
	return nil
}

// Mesh implements the api.MeshProvider interface.
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	if dc, ok := o.(drawCall); ok {
		return drawCallMesh(ctx, dc, p, r)
	}
	return nil, nil
}

// GetDependencyGraphBehaviourProvider implements dependencygraph.DependencyGraphBehaviourProvider interface
func (API) GetDependencyGraphBehaviourProvider(ctx context.Context) dependencygraph.BehaviourProvider {
	return newGlesDependencyGraphBehaviourProvider()
}
