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
	"fmt"

	"github.com/google/gapid/gapis/api"
)

// fbai is the result getFramebufferAttachmentInfo.
type fbai struct {
	width        uint32
	height       uint32
	format       GLenum // sized
	multisampled bool
}

func attachmentToEnum(a api.FramebufferAttachment) (GLenum, error) {
	switch a {
	case api.FramebufferAttachment_Color0:
		return GLenum_GL_COLOR_ATTACHMENT0, nil
	case api.FramebufferAttachment_Color1:
		return GLenum_GL_COLOR_ATTACHMENT1, nil
	case api.FramebufferAttachment_Color2:
		return GLenum_GL_COLOR_ATTACHMENT2, nil
	case api.FramebufferAttachment_Color3:
		return GLenum_GL_COLOR_ATTACHMENT3, nil
	case api.FramebufferAttachment_Depth:
		return GLenum_GL_DEPTH_ATTACHMENT, nil
	case api.FramebufferAttachment_Stencil:
		return GLenum_GL_STENCIL_ATTACHMENT, nil
	default:
		return 0, fmt.Errorf("Framebuffer attachment %v unsupported by gles", a)
	}
}

func (f *Framebuffer) getAttachment(a GLenum) (FramebufferAttachment, error) {
	switch a {
	case GLenum_GL_DEPTH_ATTACHMENT, GLenum_GL_DEPTH_STENCIL_ATTACHMENT:
		return f.DepthAttachment, nil
	case GLenum_GL_STENCIL_ATTACHMENT:
		return f.StencilAttachment, nil
	default:
		if a >= GLenum_GL_COLOR_ATTACHMENT0 && a < GLenum_GL_COLOR_ATTACHMENT0+64 {
			return f.ColorAttachments.Get(GLint(a - GLenum_GL_COLOR_ATTACHMENT0)), nil
		}
		return FramebufferAttachment{}, fmt.Errorf("Unhandled attachment: %v", a)
	}
}

// TODO: When gfx api macros produce functions instead of inlining, move this logic
// to the gles.api file.
func (s *State) getFramebufferAttachmentInfo(thread uint64, fb FramebufferId, att GLenum) (fbai, error) {
	c := s.GetContext(thread)
	if c == nil {
		return fbai{}, fmt.Errorf("No context bound")
	}
	if !c.Info.Initialized {
		return fbai{}, fmt.Errorf("Context not initialized")
	}

	framebuffer, ok := c.Objects.Framebuffers.Lookup(fb)
	if !ok {
		return fbai{}, fmt.Errorf("Invalid framebuffer %v", fb)
	}

	a, err := framebuffer.getAttachment(att)
	if err != nil {
		return fbai{}, err
	}

	switch a.Type {
	case GLenum_GL_NONE:
		return fbai{}, fmt.Errorf("%s is not bound", att)
	case GLenum_GL_TEXTURE:
		t := a.Texture
		l := t.Levels.Get(a.TextureLevel).Layers.Get(a.TextureLayer)
		if l == nil {
			return fbai{}, fmt.Errorf("Texture %v does not have Level[%v].Layer[%v]",
				t.ID, a.TextureLevel, a.TextureLayer)
		}
		multisampled := l.Samples > 0
		return fbai{uint32(l.Width), uint32(l.Height), l.SizedFormat, multisampled}, nil
	case GLenum_GL_RENDERBUFFER:
		r := a.Renderbuffer
		multisampled := r.Samples > 0
		return fbai{uint32(r.Width), uint32(r.Height), r.InternalFormat, multisampled}, nil
	default:
		return fbai{}, fmt.Errorf("Unknown framebuffer attachment type %T", a.Type)
	}
}
