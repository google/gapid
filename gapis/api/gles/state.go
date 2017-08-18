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

// TODO: When gfx api macros produce functions instead of inlining, move this logic
// to the gles.api file.
func (s *State) getFramebufferAttachmentInfo(thread uint64, fb FramebufferId, att api.FramebufferAttachment) (width, height uint32, sizedFormat GLenum, err error) {
	c := s.GetContext(thread)
	if c == nil {
		return 0, 0, 0, fmt.Errorf("No context bound")
	}
	if !c.Info.Initialized {
		return 0, 0, 0, fmt.Errorf("Context not initialized")
	}

	framebuffer, ok := c.Objects.Framebuffers[fb]
	if !ok {
		return 0, 0, 0, fmt.Errorf("Invalid framebuffer %v", fb)
	}

	var a FramebufferAttachment
	switch att {
	case api.FramebufferAttachment_Color0:
		a = framebuffer.ColorAttachments[0]
	case api.FramebufferAttachment_Color1:
		a = framebuffer.ColorAttachments[1]
	case api.FramebufferAttachment_Color2:
		a = framebuffer.ColorAttachments[2]
	case api.FramebufferAttachment_Color3:
		a = framebuffer.ColorAttachments[3]
	case api.FramebufferAttachment_Depth:
		a = framebuffer.DepthAttachment
	case api.FramebufferAttachment_Stencil:
		a = framebuffer.StencilAttachment
	default:
		return 0, 0, 0, fmt.Errorf("Framebuffer attachment %v unsupported by gles", att)
	}

	switch a.Type {
	case GLenum_GL_NONE:
		return 0, 0, 0, fmt.Errorf("%s is not bound", att)
	case GLenum_GL_TEXTURE:
		t := a.Texture
		l := t.Levels[a.TextureLevel].Layers[a.TextureLayer]
		if l == nil {
			return 0, 0, 0, fmt.Errorf("Texture %v does not have Level[%v].Layer[%v]",
				t.ID, a.TextureLevel, a.TextureLayer)
		}
		return uint32(l.Width), uint32(l.Height), l.SizedFormat, nil
	case GLenum_GL_RENDERBUFFER:
		r := a.Renderbuffer
		return uint32(r.Width), uint32(r.Height), r.InternalFormat, nil
	default:
		return 0, 0, 0, fmt.Errorf("Unknown framebuffer attachment type %T", a.Type)
	}
}
