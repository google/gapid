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
	"context"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/resolve/cmdgrouper"
)

var _ cmdgrouper.Grouper = &stateResetGrouper{}

type stateResetGrouper struct {
	rule   int
	passed bool
	prev   api.Cmd
	start  api.CmdID
	groups []cmdgrouper.Group
}

// Process considers the command for inclusion in the group.
func (g *stateResetGrouper) Process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State) {
	prev := g.prev
	g.prev = cmd

	if g.passed {
		if stateResetRules[g.rule](cmd, prev) {
			return
		}
		g.rule++
		g.passed = false
	}

	if g.rule == len(stateResetRules) {
		g.groups = append(g.groups, cmdgrouper.Group{
			Start: g.start,
			End:   id,
			Name:  "Unity state reset",
		})
		g.rule, g.passed, g.start = 0, false, id
	}

	if stateResetRules[g.rule](cmd, prev) {
		g.passed = true
	} else {
		g.rule, g.passed, g.start = 0, false, id+1
	}
}

// Build returns the groups built and resets the state of the grouper.
func (g *stateResetGrouper) Build(end api.CmdID) []cmdgrouper.Group {
	return g.groups
}

type sequenceRule func(cmd, prev api.Cmd) bool

var stateResetRules = []sequenceRule{
	eglGetCurrentContextRule(),                                                                                   //eglGetCurrentContext()
	glDisableRule(gles.GLenum_GL_DEPTH_TEST),                                                                     // glDisable(GL_DEPTH_TEST)
	glDisableRule(gles.GLenum_GL_BLEND),                                                                          // glDisable(GL_BLEND)
	glDisableRule(gles.GLenum_GL_SAMPLE_ALPHA_TO_COVERAGE),                                                       // glDisable(GL_SAMPLE_ALPHA_TO_COVERAGE)
	glDisableRule(gles.GLenum_GL_STENCIL_TEST),                                                                   // glDisable(GL_STENCIL_TEST)
	glDisableRule(gles.GLenum_GL_POLYGON_OFFSET_FILL),                                                            // glDisable(GL_POLYGON_OFFSET_FILL)
	glDisableRule(gles.GLenum_GL_SCISSOR_TEST),                                                                   // glDisable(GL_SCISSOR_TEST)
	glEnableRule(gles.GLenum_GL_DITHER),                                                                          // glEnable(GL_DITHER)
	glDepthMaskRule(0),                                                                                           // glDepthMask(0)
	glEnableRule(gles.GLenum_GL_DEPTH_TEST),                                                                      // glEnable(GL_DEPTH_TEST)
	glDepthFuncRule(gles.GLenum_GL_ALWAYS),                                                                       // glDepthFunc(GL_ALWAYS)
	glColorMaskRule(1, 1, 1, 1),                                                                                  // glColorMask(1, 1, 1, 1)
	glBlendFuncSeparateRule(gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO, gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO),    // glBlendFuncSeparate(GL_ONE, GL_ZERO, GL_ONE, GL_ZERO)
	glBlendEquationSeparateRule(gles.GLenum_GL_FUNC_ADD, gles.GLenum_GL_FUNC_ADD),                                // glBlendEquationSeparate(GL_FUNC_ADD, GL_FUNC_ADD)
	glStencilFuncSeparateRule(gles.GLenum_GL_FRONT, gles.GLenum_GL_ALWAYS, 0, 255),                               // glStencilFuncSeparate(GL_FRONT, GL_ALWAYS, 0, 255)
	glStencilOpSeparateRule(gles.GLenum_GL_FRONT, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP), // glStencilOpSeparate(GL_FRONT, GL_KEEP, GL_KEEP, GL_KEEP)
	glStencilFuncSeparateRule(gles.GLenum_GL_BACK, gles.GLenum_GL_ALWAYS, 0, 255),                                // glStencilFuncSeparate(GL_BACK, GL_ALWAYS, 0, 255)
	glStencilOpSeparateRule(gles.GLenum_GL_BACK, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP),  // glStencilOpSeparate(GL_BACK, GL_KEEP, GL_KEEP, GL_KEEP)
	glStencilMaskRule(255),                                         // glStencilMask(255)
	glCullFaceRule(gles.GLenum_GL_BACK),                            // glCullFace(GL_BACK)
	glEnableRule(gles.GLenum_GL_CULL_FACE),                         // glEnable(GL_CULL_FACE)
	glFrontFaceRule(gles.GLenum_GL_CW),                             // glFrontFace(orientation: GL_CW)
	glBindSamplerRule(),                                            // glBindSampler(0..N, 0)
	glBindBufferRule(gles.GLenum_GL_ARRAY_BUFFER),                  // glBindBuffer(GL_ARRAY_BUFFER, 0)
	glBindBufferRule(gles.GLenum_GL_ELEMENT_ARRAY_BUFFER),          // glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, 0)
	glBindBufferRule(gles.GLenum_GL_DRAW_INDIRECT_BUFFER),          // glBindBuffer(GL_DRAW_INDIRECT_BUFFER, 0)
	glBindBufferRule(gles.GLenum_GL_COPY_READ_BUFFER),              // glBindBuffer(GL_COPY_READ_BUFFER, 0)
	glBindBufferRule(gles.GLenum_GL_COPY_WRITE_BUFFER),             // glBindBuffer(GL_COPY_WRITE_BUFFER, 0)
	glBindBufferBaseRule(gles.GLenum_GL_UNIFORM_BUFFER),            // glBindBufferBase(GL_UNIFORM_BUFFER, 0..N, 0)
	glBindBufferBaseRule(gles.GLenum_GL_TRANSFORM_FEEDBACK_BUFFER), // glBindBufferBase(GL_TRANSFORM_FEEDBACK_BUFFER, 0, 0)
	glBindBufferBaseRule(gles.GLenum_GL_SHADER_STORAGE_BUFFER),     // glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 0..N, 0)
	glBindBufferBaseRule(gles.GLenum_GL_ATOMIC_COUNTER_BUFFER),     // glBindBufferBase(GL_ATOMIC_COUNTER_BUFFER, 0..N, 0)
	glBindBufferRule(gles.GLenum_GL_DISPATCH_INDIRECT_BUFFER),      // glBindBuffer(GL_DISPATCH_INDIRECT_BUFFER, 0)
	glUseProgramRule(),                                             // glUseProgram(0)
	glActiveTextureOrBindTextureRule(),                             // glActiveTexture(GL_TEXTURE31 .. 0), glBindTexture(GL_TEXTURE_2D, 0)
	glPixelStoreiRule(gles.GLenum_GL_UNPACK_ROW_LENGTH),            // glPixelStorei(GL_UNPACK_ROW_LENGTH, 0)
	glPixelStoreiRule(gles.GLenum_GL_PACK_ALIGNMENT),               // glPixelStorei(GL_PACK_ALIGNMENT, 1)
	glPixelStoreiRule(gles.GLenum_GL_UNPACK_ALIGNMENT),             // glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	glBindFramebufferRule(gles.GLenum_GL_DRAW_FRAMEBUFFER),         // glBindFramebuffer(GL_DRAW_FRAMEBUFFER, 0)
	glBindFramebufferRule(gles.GLenum_GL_READ_FRAMEBUFFER),         // glBindFramebuffer(GL_READ_FRAMEBUFFER, 0)
	glIsVertexArrayRuleOrGenVertexArraysRule(),                     // glIsVertexArray(array: 1) → 0, glGenVertexArrays(count: 1, arrays: 0xcab3ff84)
	glBindVertexArrayRule(),                                        // glBindVertexArray(array: 2)
	glDisableVertexAttribArrayRule(),                               // glDisableVertexAttribArray(0 .. N)
	glFrontFaceRule(gles.GLenum_GL_CW),                             // glFrontFace(orientation: GL_CW)
}

func eglGetCurrentContextRule() sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		_, ok := cmd.(*gles.EglGetCurrentContext)
		return ok
	}
}

func glDisableRule(capability gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDisable)
		return ok && c.Capability == capability
	}
}

func glEnableRule(capability gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlEnable)
		return ok && c.Capability == capability
	}
}

func glFrontFaceRule(orientation gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlFrontFace)
		return ok && c.Orientation == orientation
	}
}

func glDepthFuncRule(function gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDepthFunc)
		return ok && c.Function == function
	}
}

func glColorMaskRule(r, g, b, a gles.GLboolean) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlColorMask)
		return ok && c.Red == r && c.Green == g && c.Blue == b && c.Alpha == a
	}
}

func glBlendFuncSeparateRule(srcFactorRGB, dstFactorRGB, srcFactorA, dstFactorA gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBlendFuncSeparate)
		return ok &&
			c.SrcFactorRgb == srcFactorRGB &&
			c.DstFactorRgb == dstFactorRGB &&
			c.SrcFactorAlpha == srcFactorA &&
			c.DstFactorAlpha == dstFactorA
	}
}

func glBlendEquationSeparateRule(rgb, alpha gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBlendEquationSeparate)
		return ok && c.Rgb == rgb && c.Alpha == alpha
	}
}

func glStencilFuncSeparateRule(face, function gles.GLenum, referenceValue gles.GLint, mask gles.GLuint) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilFuncSeparate)
		return ok &&
			c.Face == face &&
			c.Function == function &&
			c.ReferenceValue == referenceValue &&
			c.Mask == mask
	}
}

func glStencilOpSeparateRule(face, stencilFail, stencilPassDepthFail, stencilPassDepthPass gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilOpSeparate)
		return ok &&
			c.Face == face &&
			c.StencilFail == stencilFail &&
			c.StencilPassDepthFail == stencilPassDepthFail &&
			c.StencilPassDepthPass == stencilPassDepthPass
	}
}

func glStencilMaskRule(mask gles.GLuint) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilMask)
		return ok && c.Mask == mask
	}
}

func glCullFaceRule(mode gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlCullFace)
		return ok && c.Mode == mode
	}
}

func glDepthMaskRule(enabled gles.GLboolean) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDepthMask)
		return ok && c.Enabled == enabled
	}
}

func glBindSamplerRule() sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindSampler)
		p, _ := prev.(*gles.GlBindSampler)
		return ok && c.Sampler == 0 && (p == nil || c.Index == p.Index+1)
	}
}

func glBindBufferRule(target gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindBuffer)
		return ok && c.Buffer == 0 && c.Target == target
	}
}

func glBindBufferBaseRule(target gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindBufferBase)
		p, _ := prev.(*gles.GlBindBufferBase)
		return ok && c.Buffer == 0 && c.Target == target &&
			(p == nil || c.Target != p.Target || c.Index == p.Index+1)
	}
}

func glUseProgramRule() sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlUseProgram)
		return ok && c.Program == 0
	}
}

func glActiveTextureOrBindTextureRule() sequenceRule {
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

func glPixelStoreiRule(param gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlPixelStorei)
		return ok && c.Parameter == param
	}
}

func glBindFramebufferRule(target gles.GLenum) sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindFramebuffer)
		return ok && c.Target == target
	}
}

func glIsVertexArrayRuleOrGenVertexArraysRule() sequenceRule {
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

func glBindVertexArrayRule() sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		_, ok := cmd.(*gles.GlBindVertexArray)
		return ok
	}
}

func glDisableVertexAttribArrayRule() sequenceRule {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDisableVertexAttribArray)
		p, _ := prev.(*gles.GlDisableVertexAttribArray)
		return ok && ((p == nil && c.Location == 0) || (p != nil && c.Location == p.Location+1))
	}
}
