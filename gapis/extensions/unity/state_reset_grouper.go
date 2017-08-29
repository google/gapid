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

func (g *stateResetGrouper) flush(id api.CmdID) {
	if g.rule == len(stateResetRules) {
		g.groups = append(g.groups, cmdgrouper.Group{
			Start: g.start,
			End:   id,
			Name:  "Unity state reset",
		})
		g.rule, g.passed, g.start = 0, false, id
	}
}

// Process considers the command for inclusion in the group.
func (g *stateResetGrouper) Process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State) {
	prev := g.prev
	g.prev = cmd

	for {
		g.flush(id)
		rule := stateResetRules[g.rule]
		passed := rule.pred(cmd, prev)
		optional := (rule.flags & optional) != 0
		repeats := (rule.flags & repeats) != 0

		switch {
		case passed && repeats:
			g.passed = true
			return
		case passed && !repeats:
			g.rule++
			g.passed = false
			return
		// --- below failed ---
		case optional, g.passed:
			g.rule++
			g.passed = false
			continue
		default: // failed sequence
			g.rule, g.passed, g.start = 0, false, id+1
			return
		}
	}
}

// Build returns the groups built and resets the state of the grouper.
func (g *stateResetGrouper) Build(end api.CmdID) []cmdgrouper.Group {
	g.flush(end)
	out := g.groups
	g.groups = nil
	return out
}

type sequenceRule struct {
	pred  func(cmd, prev api.Cmd) bool
	flags ruleFlags
}

type ruleFlags int

const (
	repeats  ruleFlags = 1
	optional ruleFlags = 2
)

var stateResetRules = []sequenceRule{
	{eglGetCurrentContextRule(), 0},                                                                                   // eglGetCurrentContext()
	{glDisableRule(gles.GLenum_GL_DEPTH_TEST), 0},                                                                     // glDisable(GL_DEPTH_TEST)
	{glDisableRule(gles.GLenum_GL_BLEND), 0},                                                                          // glDisable(GL_BLEND)
	{glDisableRule(gles.GLenum_GL_SAMPLE_ALPHA_TO_COVERAGE), 0},                                                       // glDisable(GL_SAMPLE_ALPHA_TO_COVERAGE)
	{glDisableRule(gles.GLenum_GL_STENCIL_TEST), 0},                                                                   // glDisable(GL_STENCIL_TEST)
	{glDisableRule(gles.GLenum_GL_POLYGON_OFFSET_FILL), 0},                                                            // glDisable(GL_POLYGON_OFFSET_FILL)
	{glDisableRule(gles.GLenum_GL_SCISSOR_TEST), 0},                                                                   // glDisable(GL_SCISSOR_TEST)
	{glDisableRule(gles.GLenum_GL_FRAMEBUFFER_SRGB_EXT), optional},                                                    // glDisable(GL_FRAMEBUFFER_SRGB_EXT)
	{glEnableRule(gles.GLenum_GL_DITHER), 0},                                                                          // glEnable(GL_DITHER)
	{glDepthFuncRule(gles.GLenum_GL_NEVER), optional},                                                                 // glDepthFunc(GL_NEVER)
	{glDepthMaskRule(0), 0},                                                                                           // glDepthMask(0)
	{glEnableRule(gles.GLenum_GL_DEPTH_TEST), optional},                                                               // glEnable(GL_DEPTH_TEST)
	{glDepthFuncRule(gles.GLenum_GL_ALWAYS), optional},                                                                // glDepthFunc(GL_ALWAYS)
	{glColorMaskRule(1, 1, 1, 1), 0},                                                                                  // glColorMask(1, 1, 1, 1)
	{glBlendFuncSeparateRule(gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO, gles.GLenum_GL_ONE, gles.GLenum_GL_ZERO), 0},    // glBlendFuncSeparate(GL_ONE, GL_ZERO, GL_ONE, GL_ZERO)
	{glBlendEquationSeparateRule(gles.GLenum_GL_FUNC_ADD, gles.GLenum_GL_FUNC_ADD), 0},                                // glBlendEquationSeparate(GL_FUNC_ADD, GL_FUNC_ADD)
	{glStencilFuncSeparateRule(gles.GLenum_GL_FRONT, gles.GLenum_GL_ALWAYS, 0, 255), 0},                               // glStencilFuncSeparate(GL_FRONT, GL_ALWAYS, 0, 255)
	{glStencilOpSeparateRule(gles.GLenum_GL_FRONT, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP), 0}, // glStencilOpSeparate(GL_FRONT, GL_KEEP, GL_KEEP, GL_KEEP)
	{glStencilFuncSeparateRule(gles.GLenum_GL_BACK, gles.GLenum_GL_ALWAYS, 0, 255), 0},                                // glStencilFuncSeparate(GL_BACK, GL_ALWAYS, 0, 255)
	{glStencilOpSeparateRule(gles.GLenum_GL_BACK, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP, gles.GLenum_GL_KEEP), 0},  // glStencilOpSeparate(GL_BACK, GL_KEEP, GL_KEEP, GL_KEEP)
	{glStencilMaskRule(255), 0},                                           // glStencilMask(255)
	{glCullFaceRule(gles.GLenum_GL_BACK), 0},                              // glCullFace(GL_BACK)
	{glEnableRule(gles.GLenum_GL_CULL_FACE), 0},                           // glEnable(GL_CULL_FACE)
	{glFrontFaceRule(gles.GLenum_GL_CW), 0},                               // glFrontFace(orientation: GL_CW)
	{glBindSamplerRule(), repeats},                                        // glBindSampler(0..N, 0)
	{glBindBufferRule(gles.GLenum_GL_ARRAY_BUFFER), 0},                    // glBindBuffer(GL_ARRAY_BUFFER, 0)
	{glBindBufferRule(gles.GLenum_GL_ELEMENT_ARRAY_BUFFER), 0},            // glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, 0)
	{glBindBufferRule(gles.GLenum_GL_DRAW_INDIRECT_BUFFER), 0},            // glBindBuffer(GL_DRAW_INDIRECT_BUFFER, 0)
	{glBindBufferRule(gles.GLenum_GL_COPY_READ_BUFFER), 0},                // glBindBuffer(GL_COPY_READ_BUFFER, 0)
	{glBindBufferRule(gles.GLenum_GL_COPY_WRITE_BUFFER), 0},               // glBindBuffer(GL_COPY_WRITE_BUFFER, 0)
	{glBindBufferBaseRule(gles.GLenum_GL_UNIFORM_BUFFER), repeats},        // glBindBufferBase(GL_UNIFORM_BUFFER, 0..N, 0)
	{glBindBufferBaseRule(gles.GLenum_GL_TRANSFORM_FEEDBACK_BUFFER), 0},   // glBindBufferBase(GL_TRANSFORM_FEEDBACK_BUFFER, 0, 0)
	{glBindBufferBaseRule(gles.GLenum_GL_SHADER_STORAGE_BUFFER), repeats}, // glBindBufferBase(GL_SHADER_STORAGE_BUFFER, 0..N, 0)
	{glBindBufferBaseRule(gles.GLenum_GL_ATOMIC_COUNTER_BUFFER), repeats}, // glBindBufferBase(GL_ATOMIC_COUNTER_BUFFER, 0..N, 0)
	{glBindBufferRule(gles.GLenum_GL_DISPATCH_INDIRECT_BUFFER), 0},        // glBindBuffer(GL_DISPATCH_INDIRECT_BUFFER, 0)
	{glUseProgramRule(), 0},                                               // glUseProgram(0)
	{glActiveTextureOrBindTextureRule(), repeats},                         // glActiveTexture(GL_TEXTURE31 .. 0), glBindTexture(GL_TEXTURE_2D, 0)
	{glPixelStoreiRule(gles.GLenum_GL_UNPACK_ROW_LENGTH), 0},              // glPixelStorei(GL_UNPACK_ROW_LENGTH, 0)
	{glPixelStoreiRule(gles.GLenum_GL_PACK_ALIGNMENT), 0},                 // glPixelStorei(GL_PACK_ALIGNMENT, 1)
	{glPixelStoreiRule(gles.GLenum_GL_UNPACK_ALIGNMENT), 0},               // glPixelStorei(GL_UNPACK_ALIGNMENT, 1)
	{glBindFramebufferRule(gles.GLenum_GL_DRAW_FRAMEBUFFER), 0},           // glBindFramebuffer(GL_DRAW_FRAMEBUFFER, 0)
	{glBindFramebufferRule(gles.GLenum_GL_READ_FRAMEBUFFER), 0},           // glBindFramebuffer(GL_READ_FRAMEBUFFER, 0)
	{glIsVertexArrayRuleOrGenVertexArraysRule(), repeats},                 // glIsVertexArray(array: 1) → 0, glGenVertexArrays(count: 1, arrays: 0xcab3ff84)
	{glBindVertexArrayRule(), 0},                                          // glBindVertexArray(array: 2)
	{glDisableVertexAttribArrayRule(), repeats},                           // glDisableVertexAttribArray(0 .. N)
	{glFrontFaceRule(gles.GLenum_GL_CW), optional},                        // glFrontFace(orientation: GL_CW)
}

func eglGetCurrentContextRule() func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		_, ok := cmd.(*gles.EglGetCurrentContext)
		return ok
	}
}

func glDisableRule(capability gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDisable)
		return ok && c.Capability == capability
	}
}

func glEnableRule(capability gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlEnable)
		return ok && c.Capability == capability
	}
}

func glFrontFaceRule(orientation gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlFrontFace)
		return ok && c.Orientation == orientation
	}
}

func glDepthFuncRule(function gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDepthFunc)
		return ok && c.Function == function
	}
}

func glColorMaskRule(r, g, b, a gles.GLboolean) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlColorMask)
		return ok && c.Red == r && c.Green == g && c.Blue == b && c.Alpha == a
	}
}

func glBlendFuncSeparateRule(srcFactorRGB, dstFactorRGB, srcFactorA, dstFactorA gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBlendFuncSeparate)
		return ok &&
			c.SrcFactorRgb == srcFactorRGB &&
			c.DstFactorRgb == dstFactorRGB &&
			c.SrcFactorAlpha == srcFactorA &&
			c.DstFactorAlpha == dstFactorA
	}
}

func glBlendEquationSeparateRule(rgb, alpha gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBlendEquationSeparate)
		return ok && c.Rgb == rgb && c.Alpha == alpha
	}
}

func glStencilFuncSeparateRule(face, function gles.GLenum, referenceValue gles.GLint, mask gles.GLuint) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilFuncSeparate)
		return ok &&
			c.Face == face &&
			c.Function == function &&
			c.ReferenceValue == referenceValue &&
			c.Mask == mask
	}
}

func glStencilOpSeparateRule(face, stencilFail, stencilPassDepthFail, stencilPassDepthPass gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilOpSeparate)
		return ok &&
			c.Face == face &&
			c.StencilFail == stencilFail &&
			c.StencilPassDepthFail == stencilPassDepthFail &&
			c.StencilPassDepthPass == stencilPassDepthPass
	}
}

func glStencilMaskRule(mask gles.GLuint) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlStencilMask)
		return ok && c.Mask == mask
	}
}

func glCullFaceRule(mode gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlCullFace)
		return ok && c.Mode == mode
	}
}

func glDepthMaskRule(enabled gles.GLboolean) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDepthMask)
		return ok && c.Enabled == enabled
	}
}

func glBindSamplerRule() func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindSampler)
		p, _ := prev.(*gles.GlBindSampler)
		return ok && c.Sampler == 0 && (p == nil || c.Index == p.Index+1)
	}
}

func glBindBufferRule(target gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindBuffer)
		return ok && c.Buffer == 0 && c.Target == target
	}
}

func glBindBufferBaseRule(target gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindBufferBase)
		p, _ := prev.(*gles.GlBindBufferBase)
		return ok && c.Buffer == 0 && c.Target == target &&
			(p == nil || c.Target != p.Target || c.Index == p.Index+1)
	}
}

func glUseProgramRule() func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlUseProgram)
		return ok && c.Program == 0
	}
}

func glActiveTextureOrBindTextureRule() func(cmd, prev api.Cmd) bool {
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

func glPixelStoreiRule(param gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlPixelStorei)
		return ok && c.Parameter == param
	}
}

func glBindFramebufferRule(target gles.GLenum) func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlBindFramebuffer)
		return ok && c.Target == target
	}
}

func glIsVertexArrayRuleOrGenVertexArraysRule() func(cmd, prev api.Cmd) bool {
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

func glBindVertexArrayRule() func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		_, ok := cmd.(*gles.GlBindVertexArray)
		return ok
	}
}

func glDisableVertexAttribArrayRule() func(cmd, prev api.Cmd) bool {
	return func(cmd, prev api.Cmd) bool {
		c, ok := cmd.(*gles.GlDisableVertexAttribArray)
		p, _ := prev.(*gles.GlDisableVertexAttribArray)
		return ok && ((p == nil && c.Location == 0) || (p != nil && c.Location == p.Location+1))
	}
}
