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
func (s *State) getFramebufferAttachmentInfo(att gfxapi.FramebufferAttachment) (width, height uint32, ifmt imgfmt, err error) {
	c := s.getContext()
	if c == nil {
		return 0, 0, imgfmt{}, fmt.Errorf("No context bound")
	}

	framebuffer, ok := c.Instances.Framebuffers[c.BoundReadFramebuffer]
	if !ok {
		return 0, 0, imgfmt{}, fmt.Errorf("No GL_FRAMEBUFFER bound")
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
		return 0, 0, imgfmt{}, fmt.Errorf("Framebuffer attachment %v unsupported by gles", att)
	}

	if a.ObjectType == GLenum_GL_NONE {
		return 0, 0, imgfmt{}, fmt.Errorf("%s is not bound", att)
	}

	switch a.ObjectType {
	case GLenum_GL_TEXTURE:
		id := TextureId(a.ObjectName)
		t := c.Instances.Textures[id]
		switch t.Kind {
		case GLenum_GL_TEXTURE_2D:
			l := t.Texture2D[a.TextureLevel]
			return uint32(l.Width), uint32(l.Height), newImgfmt(t.TexelFormat, t.TexelType), nil
		case GLenum_GL_TEXTURE_CUBE_MAP:
			l := t.Cubemap[a.TextureLevel]
			f := l.Faces[a.TextureCubeMapFace]
			return uint32(f.Width), uint32(f.Height), newImgfmt(f.TexelFormat, f.TexelType), nil
		default:
			return 0, 0, imgfmt{}, fmt.Errorf("Unknown texture kind %v", t.Kind)
		}
	case GLenum_GL_RENDERBUFFER:
		id := RenderbufferId(a.ObjectName)
		r, ok := c.Instances.Renderbuffers[id]
		if !ok {
			return 0, 0, imgfmt{}, fmt.Errorf("Renderbuffer %v not found", id)
		}
		fmt, ty := extractSizedInternalFormat(r.InternalFormat)
		ifmt := imgfmt{r.InternalFormat, fmt, ty}
		return uint32(r.Width), uint32(r.Height), ifmt, nil
	default:
		return 0, 0, imgfmt{}, fmt.Errorf("Unknown framebuffer attachment type %T", a.ObjectType)
	}
}
