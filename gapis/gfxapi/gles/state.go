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

	"github.com/google/gapid/gapis/gfxapi"
)

func (s *State) getContext() *Context {
	return s.Contexts[s.CurrentThread]
}

// TODO: When gfx api macros produce functions instead of inlining, move this logic
// to the gles.api file.
func (s *State) getFramebufferAttachmentInfo(att gfxapi.FramebufferAttachment) (width, height uint32, sizedFormat GLenum, err error) {
	c := s.getContext()
	if c == nil {
		return 0, 0, 0, fmt.Errorf("No context bound")
	}
	if !c.Info.Initialized {
		return 0, 0, 0, fmt.Errorf("Context not initialized")
	}

	framebuffer, ok := c.Objects.Framebuffers[c.BoundReadFramebuffer]
	if !ok {
		return 0, 0, 0, fmt.Errorf("No GL_FRAMEBUFFER bound")
	}

	var a FramebufferAttachment
	switch att {
	case gfxapi.FramebufferAttachment_Color0:
		a = framebuffer.ColorAttachments[0]
	case gfxapi.FramebufferAttachment_Color1:
		a = framebuffer.ColorAttachments[1]
	case gfxapi.FramebufferAttachment_Color2:
		a = framebuffer.ColorAttachments[2]
	case gfxapi.FramebufferAttachment_Color3:
		a = framebuffer.ColorAttachments[3]
	case gfxapi.FramebufferAttachment_Depth:
		a = framebuffer.DepthAttachment
	case gfxapi.FramebufferAttachment_Stencil:
		a = framebuffer.StencilAttachment
	default:
		return 0, 0, 0, fmt.Errorf("Framebuffer attachment %v unsupported by gles", att)
	}

	if a.ObjectType == GLenum_GL_NONE {
		return 0, 0, 0, fmt.Errorf("%s is not bound", att)
	}

	switch a.ObjectType {
	case GLenum_GL_TEXTURE:
		id := TextureId(a.ObjectName)
		t, ok := c.SharedObjects.Textures[id]
		if !ok {
			return 0, 0, 0, fmt.Errorf("Invalid texture attachment %v", id)
		}
		switch t.Kind {
		case GLenum_GL_TEXTURE_2D:
			l := t.Texture2D[a.TextureLevel]
			fmt := getSizedFormat(l.DataFormat, l.DataType)
			return uint32(l.Width), uint32(l.Height), fmt, nil
		case GLenum_GL_TEXTURE_CUBE_MAP:
			l := t.Cubemap[a.TextureLevel]
			f := l.Faces[a.TextureCubeMapFace]
			fmt := getSizedFormat(f.DataFormat, f.DataType)
			return uint32(f.Width), uint32(f.Height), fmt, nil
		default:
			return 0, 0, 0, fmt.Errorf("Unknown texture kind %v", t.Kind)
		}
	case GLenum_GL_RENDERBUFFER:
		id := RenderbufferId(a.ObjectName)
		r, ok := c.SharedObjects.Renderbuffers[id]
		if !ok {
			return 0, 0, 0, fmt.Errorf("Renderbuffer %v not found", id)
		}
		return uint32(r.Width), uint32(r.Height), r.InternalFormat, nil
	default:
		return 0, 0, 0, fmt.Errorf("Unknown framebuffer attachment type %T", a.ObjectType)
	}
}
