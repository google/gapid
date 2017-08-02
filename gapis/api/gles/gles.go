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

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct{}

func GetContext(s *api.State, thread uint64) *Context {
	return GetState(s).GetContext(thread)
}

func (s *State) GetContext(thread uint64) *Context {
	return s.Contexts[thread]
}

func (s *State) SetThread(thread uint64) {
	s.CurrentContext = s.GetContext(thread)
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
func (API) GetFramebufferAttachmentInfo(state *api.State, thread uint64, attachment api.FramebufferAttachment) (width, height uint32, index uint32, format *image.Format, err error) {
	w, h, sizedFormat, err := GetState(state).getFramebufferAttachmentInfo(thread, attachment)
	if sizedFormat == 0 {
		return 0, 0, 0, nil, fmt.Errorf("No format set")
	}
	if err != nil {
		return 0, 0, 0, nil, err
	}
	fmt, ty := getUnsizedFormatAndType(sizedFormat)
	f, err := getImageFormat(fmt, ty)
	return w, h, uint32(attachment), f, err
}

// Context returns the active context for the given state and thread.
func (API) Context(s *api.State, thread uint64) api.Context {
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
