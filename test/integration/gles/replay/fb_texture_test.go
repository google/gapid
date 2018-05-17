// Copyright (C) 2018 Google Inc.
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

package replay

import (
	"testing"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/test/integration/gles/snippets"
)

func TestClearFramebuffer(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)

	tex := gles.TextureId(10)
	fbo := gles.FramebufferId(20)

	texR := b.Data(ctx, tex)
	fboR := b.Data(ctx, fbo)

	b.Add(
		b.CB.GlGenTextures(1, texR.Ptr()).AddRead(texR.Data()),
		b.CB.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, tex),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_WRAP_S, gles.GLint(gles.GLenum_GL_CLAMP_TO_EDGE)),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_WRAP_T, gles.GLint(gles.GLenum_GL_CLAMP_TO_EDGE)),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_LINEAR)),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_LINEAR)),
		b.CB.GlTexImage2D(
			gles.GLenum_GL_TEXTURE_2D,
			0,
			gles.GLint(gles.GLenum_GL_RGBA),
			64, 64, 0,
			gles.GLenum_GL_RGBA,
			gles.GLenum_GL_UNSIGNED_BYTE,
			memory.Nullptr),

		b.CB.GlGenFramebuffers(1, fboR.Ptr()).AddRead(fboR.Data()),
		b.CB.GlBindFramebuffer(gles.GLenum_GL_FRAMEBUFFER, fbo),
		b.CB.GlFramebufferTexture2D(
			gles.GLenum_GL_FRAMEBUFFER,
			gles.GLenum_GL_COLOR_ATTACHMENT0,
			gles.GLenum_GL_TEXTURE_2D,
			tex,
			0),

		b.CB.GlBindFramebuffer(gles.GLenum_GL_FRAMEBUFFER, fbo),
	)
	clear := b.ClearColor(1, 0, 0, 1)

	c := b.Capture(ctx, "clear-framebuffer")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 64, 64, 0, "solid-red", clear, nil)
	})

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkTextureBuffer(ctx, c, d, 64, 64, 0, "solid-red", clear, tex, nil)
	})
}
