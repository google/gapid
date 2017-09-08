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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct{}

func GetContext(s *api.GlobalState, thread uint64) *Context {
	return GetState(s).GetContext(thread)
}

func (s *State) GetContext(thread uint64) *Context {
	return s.Contexts[thread]
}

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (s *State) Root(ctx context.Context, p *path.State) (path.Node, error) {
	if p.Context == nil || !p.Context.IsValid() {
		return p, nil
	}
	c, err := resolve.Context(ctx, p.After.Capture.Context(p.Context))
	if err != nil {
		return nil, err
	}
	for thread, context := range s.Contexts {
		if c.ID() == context.ID() {
			return s.contextRoot(p.After, thread), nil
		}
	}
	return nil, nil
}

func (s *State) contextRoot(p *path.Command, thread uint64) *path.MapIndex {
	return path.NewField("Contexts", resolve.APIStateAfter(p, ID)).MapIndex(thread)
}

func (s *State) objectsRoot(p *path.Command, thread uint64) *path.Field {
	return s.contextRoot(p, thread).Field("Objects")
}

func (c *State) preMutate(ctx context.Context, s *api.GlobalState, cmd api.Cmd) error {
	c.CurrentContext = c.GetContext(cmd.Thread())
	// TODO: Find better way to separate GL and EGL commands.
	if c.CurrentContext == nil && strings.HasPrefix(cmd.CmdName(), "gl") {
		if f := s.NewMessage; f != nil {
			f(log.Error, messages.ErrNoContextBound(cmd.Thread()))
		}
		return &api.ErrCmdAborted{Reason: "No context bound"}
	}
	return nil
}

func (b *Buffer) GetID() BufferId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

func (b *Framebuffer) GetID() FramebufferId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

func (b *Renderbuffer) GetID() RenderbufferId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

func (b *Program) GetID() ProgramId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

func (b *VertexArray) GetID() VertexArrayId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

func (b *Texture) GetID() TextureId {
	if b != nil {
		return b.ID
	} else {
		return 0
	}
}

// GetFramebufferAttachmentInfo returns the width, height and format of the specified framebuffer attachment.
func (API) GetFramebufferAttachmentInfo(state *api.GlobalState, thread uint64, attachment api.FramebufferAttachment) (width, height uint32, index uint32, format *image.Format, err error) {
	s := GetState(state)
	c := s.GetContext(thread)
	if c == nil {
		return 0, 0, 0, nil, fmt.Errorf("No context bound")
	}
	if !c.Info.Initialized {
		return 0, 0, 0, nil, fmt.Errorf("Context not initialized")
	}

	fb := c.Bound.DrawFramebuffer.GetID()

	w, h, sizedFormat, err := s.getFramebufferAttachmentInfo(thread, fb, attachment)
	if sizedFormat == 0 {
		return 0, 0, 0, nil, fmt.Errorf("No format set")
	}
	if err != nil {
		return 0, 0, 0, nil, err
	}
	fmt, ty := getUnsizedFormatAndType(sizedFormat)
	f, err := getImageFormat(fmt, ty)
	return w, h, 0, f, err
}

// Context returns the active context for the given state and thread.
func (API) Context(s *api.GlobalState, thread uint64) api.Context {
	if c := GetContext(s, thread); c != nil {
		return c
	}
	return nil
}

// Mesh implements the api.MeshProvider interface.
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh) (*api.Mesh, error) {
	if dc, ok := o.(drawCall); ok {
		return drawCallMesh(ctx, dc, p)
	}
	return nil, nil
}

// GetDependencyGraphBehaviourProvider implements dependencygraph.DependencyGraphBehaviourProvider interface
func (API) GetDependencyGraphBehaviourProvider(ctx context.Context) dependencygraph.BehaviourProvider {
	return newGlesDependencyGraphBehaviourProvider()
}
