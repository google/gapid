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

package unity

import (
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/resolve/cmdgrouper"
)

func newStateResetGrouper() cmdgrouper.Grouper {
	eglGetCurrentContext := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.EglGetCurrentContext)
			return ok
		}
	}
	glDisable := func(capability gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlDisable)
			return ok && c.Capability == capability
		}
	}
	glEnable := func(capability gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlEnable)
			return ok && c.Capability == capability
		}
	}
	glFrontFace := func(orientation gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlFrontFace)
			return ok && c.Orientation == orientation
		}
	}
	glDepthFunc := func(function gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlDepthFunc)
			return ok && c.Function == function
		}
	}
	glColorMask := func(r, g, b, a gles.GLboolean) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlColorMask)
			return ok && c.Red == r && c.Green == g && c.Blue == b && c.Alpha == a
		}
	}
	glBlendFuncSeparate := func(srcFactorRGB, dstFactorRGB, srcFactorA, dstFactorA gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBlendFuncSeparate)
			return ok &&
				c.SrcFactorRgb == srcFactorRGB &&
				c.DstFactorRgb == dstFactorRGB &&
				c.SrcFactorAlpha == srcFactorA &&
				c.DstFactorAlpha == dstFactorA
		}
	}
	glBlendEquationSeparate := func(rgb, alpha gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBlendEquationSeparate)
			return ok && c.Rgb == rgb && c.Alpha == alpha
		}
	}
	glStencilFuncSeparate := func(face, function gles.GLenum, referenceValue gles.GLint, mask gles.GLuint) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlStencilFuncSeparate)
			return ok &&
				c.Face == face &&
				c.Function == function &&
				c.ReferenceValue == referenceValue &&
				c.Mask == mask
		}
	}
	glStencilOpSeparate := func(face, stencilFail, stencilPassDepthFail, stencilPassDepthPass gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlStencilOpSeparate)
			return ok &&
				c.Face == face &&
				c.StencilFail == stencilFail &&
				c.StencilPassDepthFail == stencilPassDepthFail &&
				c.StencilPassDepthPass == stencilPassDepthPass
		}
	}
	glStencilMask := func(mask gles.GLuint) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlStencilMask)
			return ok && c.Mask == mask
		}
	}
	glCullFace := func(mode gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlCullFace)
			return ok && c.Mode == mode
		}
	}
	glDepthMask := func(enabled gles.GLboolean) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlDepthMask)
			return ok && c.Enabled == enabled
		}
	}
	glBindSampler := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBindSampler)
			p, _ := prev.(*gles.GlBindSampler)
			return ok && c.Sampler == 0 && (p == nil || c.Index == p.Index+1)
		}
	}
	glBindBuffer := func(target gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBindBuffer)
			return ok && c.Buffer == 0 && c.Target == target
		}
	}
	glBindBufferBase := func(target gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBindBufferBase)
			p, _ := prev.(*gles.GlBindBufferBase)
			return ok && c.Buffer == 0 && c.Target == target &&
				(p == nil || c.Target != p.Target || c.Index == p.Index+1)
		}
	}
	glUseProgram := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlUseProgram)
			return ok && c.Program == 0
		}
	}
	glActiveTextureOrBindTexture := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			if _, ok := cmd.(*gles.GlActiveTexture); ok {
				return true
			}
			if cmd, ok := cmd.(*gles.GlBindTexture); ok && cmd.Texture == 0 {
				return true
			}
			return false
		}
	}
	glPixelStorei := func(param gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlPixelStorei)
			return ok && c.Parameter == param
		}
	}
	glBindFramebuffer := func(target gles.GLenum) func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlBindFramebuffer)
			return ok && c.Target == target
		}
	}
	glIsVertexArrayRuleOrGenVertexArrays := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			if _, ok := cmd.(*gles.GlIsVertexArray); ok {
				return true
			}
			if _, ok := cmd.(*gles.GlGenVertexArrays); ok {
				return true
			}
			return false
		}
	}
	glBindVertexArray := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			_, ok := cmd.(*gles.GlBindVertexArray)
			return ok
		}
	}
	glDisableVertexAttribArray := func() func(cmd, prev api.Cmd) bool {
		return func(cmd, prev api.Cmd) bool {
			c, ok := cmd.(*gles.GlDisableVertexAttribArray)
			p, _ := prev.(*gles.GlDisableVertexAttribArray)
			return ok && ((p == nil && c.Location == 0) || (p != nil && c.Location == p.Location+1))
		}
	}

	return cmdgrouper.Sequence("Unity state reset",
		cmdgrouper.Rule{Pred: eglGetCurrentContext()},                                                                                   // eglGetCurrentContext()
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_DEPTH_TEST)},                                                                     // glDisable(GL_DEPTH_TEST)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_BLEND)},                                                                          // glDisable(GL_BLEND)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_SAMPLE_ALPHA_TO_COVERAGE)},                                                       // glDisable(GL_SAMPLE_ALPHA_TO_COVERAGE)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_STENCIL_TEST)},                                                                   // glDisable(GL_STENCIL_TEST)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_POLYGON_OFFSET_FILL)},                                                            // glDisable(GL_POLYGON_OFFSET_FILL)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_SCISSOR_TEST)},                                                                   // glDisable(GL_SCISSOR_TEST)
		cmdgrouper.Rule{Pred: glDisable(gles.GLenum_GL_FRAMEBUFFER_SRGB_EXT), Optional: true},                                           // glDisable(GL_FRAMEBUFFER_SRGB_EXT)
		cmdgrouper.Rule{Pred: glEnable(gles.GLenum_GL_DITHER)},                                                                          // glEnable(GL_DITHER)
		cmdgrouper.Rule{Pred: glDepthFunc(gles.GLenum_GL_NEVER), Optional: true},                                                        // glDepthFunc(GL_NEVER)
		cmdgrouper.Rule{Pred: glDepthMask(0)},                                                                                           // glDepthMask(0)
		cmdgrouper.Rule{Pred: glEnable(gles.GLenum_GL_DEPTH_TEST), Optional: true},                                                      // glEnable(GL_DEPTH_TEST)
		cmdgrouper.Rule{Pred: glDepthFunc(gles.GLenum_GL_ALWAYS), Optional: true},                                                       // glDepthFunc(GL_ALWAYS)
		cmdgrouper.Rule{Pred: glColorMask(1, 1, 1, 1)},                                                                                  // glColorMask(1, 1, 1, 1)
		cmdgrouper.Rule{Pred: glBlendFuncSeparate(gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO, gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO)},    // glBlendFuncSeparate(GL_ONE, GL_ZERO, GL_ONE, GL_ZERO)
		cmdgrouper.Rule{Pred: glBlendEquationSeparate(gles.GLenum_GL_FUNC_ADD, gles.GLenum_GL_FUNC_ADD)},                                // glBlendEquationSeparate(GL_FUNC_ADD, GL_FUNC_ADD)
		cmdgrouper.Rule{Pred: glStencilFuncSeparate(gles.GLenum_GL_FRONT, gles.GLenum_GL_ALWAYS, 0, 255)},                               // glStencilFuncSeparate(GL_FRONT, GL_ALWAYS, 0, 255)
		cmdgrouper.Rule{Pred: glStencilOpSeparate(gles.GLenum_GL_FRONT, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP)}, // glStencilOpSeparate(GL_FRONT, GL_KEEP, GL_KEEP, GL_KEEP)
		cmdgrouper.Rule{Pred: glStencilFuncSeparate(gles.GLenum_GL_BACK, gles.GLenum_GL_ALWAYS, 0, 255)},                                // glStencilFuncSeparate(GL_BACK, GL_ALWAYS, 0, 255)
		cmdgrouper.Rule{Pred: glStencilOpSeparate(gles.GLenum_GL_BACK, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP)},  // glStencilOpSeparate(GL_BACK, GL_KEEP, GL_KEEP, GL_KEEP)
		cmdgrouper.Rule{Pred: glStencilMask(255)},                                                                                       // glStencilMask(255)
		cmdgrouper.Rule{Pred: glCullFace(gles.GLenum_GL_BACK)},                                                                          // glCullFace(GL_BACK)
		cmdgrouper.Rule{Pred: glEnable(gles.GLenum_GL_CULL_FACE)},                                                                       // glEnable(GL_CULL_FACE)
		cmdgrouper.Rule{Pred: glFrontFace(gles.GLenum_GL_CW)},                                                                           // glFrontFace(orientation: GL_CW)
		cmdgrouper.Rule{Pred: glBindSampler(), Repeats: true},                                                                           // glBindSampler(0..N, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_ARRAY_BUFFER)},                                                                // glBindBuffer(GL_ARRAY_BUFFER, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_ELEMENT_ARRAY_BUFFER)},                                                        // glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_DRAW_INDIRECT_BUFFER)},                                                        // glBindBuffer(GL_DRAW_INDIRECT_BUFFER, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_COPY_READ_BUFFER)},                                                            // glBindBuffer(GL_COPY_READ_BUFFER, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_COPY_WRITE_BUFFER)},                                                           // glBindBuffer(GL_COPY_WRITE_BUFFER, 0)
		cmdgrouper.Rule{Pred: glBindBufferBase(gles.GLenum_GL_UNIFORM_BUFFER), Repeats: true},                                           // glBindBufferBase(GL_UNIFORM_BUFFER, 0..N, 0)
		cmdgrouper.Rule{Pred: glBindBufferBase(gles.GLenum_GL_TRANSFORM_FEEDBACK_BUFFER)},                                               // glBindBufferBase(GL_TRANSFORM_FEEDBACK_BUFFER, 0, 0)
		cmdgrouper.Rule{Pred: glBindBufferBase(gles.GLenum_GL_SHADER_STORAGE_BUFFER), Repeats: true},                                    // glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 0..N, 0)
		cmdgrouper.Rule{Pred: glBindBufferBase(gles.GLenum_GL_ATOMIC_COUNTER_BUFFER), Repeats: true},                                    // glBindBufferBase(GL_ATOMIC_COUNTER_BUFFER, 0..N, 0)
		cmdgrouper.Rule{Pred: glBindBuffer(gles.GLenum_GL_DISPATCH_INDIRECT_BUFFER)},                                                    // glBindBuffer(GL_DISPATCH_INDIRECT_BUFFER, 0)
		cmdgrouper.Rule{Pred: glUseProgram()},                                                                                           // glUseProgram(0)
		cmdgrouper.Rule{Pred: glActiveTextureOrBindTexture(), Repeats: true},                                                            // glActiveTexture(GL_TEXTURE31 .. 0), glBindTexture(GL_TEXTURE_2D, 0)
		cmdgrouper.Rule{Pred: glPixelStorei(gles.GLenum_GL_UNPACK_ROW_LENGTH)},                                                          // glPixelStorei(GL_UNPACK_ROW_LENGTH, 0)
		cmdgrouper.Rule{Pred: glPixelStorei(gles.GLenum_GL_PACK_ALIGNMENT)},                                                             // glPixelStorei(GL_PACK_ALIGNMENT, 1)
		cmdgrouper.Rule{Pred: glPixelStorei(gles.GLenum_GL_UNPACK_ALIGNMENT)},                                                           // glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
		cmdgrouper.Rule{Pred: glBindFramebuffer(gles.GLenum_GL_DRAW_FRAMEBUFFER)},                                                       // glBindFramebuffer(GL_DRAW_FRAMEBUFFER, 0)
		cmdgrouper.Rule{Pred: glBindFramebuffer(gles.GLenum_GL_READ_FRAMEBUFFER)},                                                       // glBindFramebuffer(GL_READ_FRAMEBUFFER, 0)
		cmdgrouper.Rule{Pred: glIsVertexArrayRuleOrGenVertexArrays(), Repeats: true},                                                    // glIsVertexArray(array: 1) → 0, glGenVertexArrays(count: 1, arrays: 0xcab3ff84)
		cmdgrouper.Rule{Pred: glBindVertexArray()},                                                                                      // glBindVertexArray(array: 2)
		cmdgrouper.Rule{Pred: glDisableVertexAttribArray(), Repeats: true},                                                              // glDisableVertexAttribArray(0 .. N)
		cmdgrouper.Rule{Pred: glFrontFace(gles.GLenum_GL_CW), Optional: true},                                                           // glFrontFace(orientation: GL_CW)
	)
}
