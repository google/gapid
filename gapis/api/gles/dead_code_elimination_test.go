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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

func TestDeadAtomRemoval(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	// Keep the given atom alive in the optimization.
	isLive := map[api.Cmd]bool{}
	live := func(cmd api.Cmd) api.Cmd { isLive[cmd] = true; return cmd }

	// Expect the atom to be removed by the optimization.
	isDead := map[api.Cmd]bool{}
	dead := func(cmd api.Cmd) api.Cmd { isDead[cmd] = true; return cmd }

	programInfo := &ProgramInfo{
		LinkStatus: GLboolean_GL_TRUE,
		ActiveUniforms: UniformIndexːActiveUniformᵐ{
			0: {
				Name:      "uniforms",
				Type:      GLenum_GL_FLOAT_VEC4,
				Location:  0,
				ArraySize: 10,
			},
		},
	}

	ctxHandle1 := memory.BytePtr(1, memory.ApplicationPool)
	ctxHandle2 := memory.BytePtr(2, memory.ApplicationPool)
	cb := CommandBuilder{Thread: 0}
	prologue := []api.Cmd{
		cb.EglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1),
		api.WithExtras(
			cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1, 0),
			NewStaticContextState(), NewDynamicContextState(64, 64, false)),
		cb.GlCreateProgram(1),
		cb.GlCreateProgram(2),
		api.WithExtras(cb.GlLinkProgram(1), programInfo),
		api.WithExtras(cb.GlLinkProgram(2), programInfo),
		cb.GlUseProgram(1),
	}
	allBuffers := GLbitfield_GL_COLOR_BUFFER_BIT | GLbitfield_GL_DEPTH_BUFFER_BIT | GLbitfield_GL_STENCIL_BUFFER_BIT
	tests := map[string][]api.Cmd{
		"Draw calls up to the requested point are preserved": {
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 1, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 2, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 3, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 4, 0)),
		},
		"No request in frame kills draw calls": {
			dead(cb.GlClear(allBuffers)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 1, 0)),
			dead(cb.EglSwapBuffers(memory.Nullptr, memory.Nullptr, EGLBoolean(1))),
			cb.GlClear(allBuffers),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Multiple requests": {
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Simple overwrite": {
			dead(cb.GlUniform4fv(0, 1, memory.Nullptr)),
			cb.GlUniform4fv(1, 1, memory.Nullptr),
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			dead(cb.GlVertexAttribPointer(0, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr)),
			cb.GlVertexAttribPointer(1, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr),
			cb.GlVertexAttribPointer(0, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr),
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Overwrites should be tracked per program": {
			cb.GlUseProgram(1),
			dead(cb.GlUniform4fv(0, 1, memory.Nullptr)),
			cb.GlUseProgram(2),
			cb.GlUniform4fv(0, 1, memory.Nullptr), // Unaffected
			cb.GlUseProgram(1),
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			cb.GlUseProgram(1),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			cb.GlUseProgram(2),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Arrays should not interact with scalars": {
			cb.GlUniform4fv(0, 10, memory.Nullptr),
			cb.GlUniform4fv(0, 1, memory.Nullptr), // Unaffected
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Arrays should not interact with scalars (2)": {
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			cb.GlUniform4fv(0, 10, memory.Nullptr), // Unaffected
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Unsupported atoms are left unmodified": {
			cb.GlUseProgram(1),
			dead(cb.GlUniform4fv(0, 1, memory.Nullptr)),
			cb.GlUniform1f(0, 3.14), // Not handled in the optimization.
			cb.GlLinkProgram(1),     // Not handled in the optimization.
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Multiple contexts": {
			// Draw in context 1
			dead(cb.GlUniform4fv(0, 1, memory.Nullptr)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			cb.GlClear(allBuffers),
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			// Draw in context 2
			cb.EglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2),
			api.WithExtras(
				cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2, 0),
				NewStaticContextState(), NewDynamicContextState(64, 64, false)),
			cb.GlCreateProgram(1),
			api.WithExtras(cb.GlLinkProgram(1), programInfo),
			cb.GlUseProgram(1),
			dead(cb.GlUniform4fv(0, 1, memory.Nullptr)),
			dead(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			cb.GlClear(allBuffers),
			cb.GlUniform4fv(0, 1, memory.Nullptr),
			cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			// Request from both contexts
			cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2, 0),
			live(cb.GlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
	}

	for name, atoms := range tests {
		inputAtoms := append(prologue, atoms...)

		h := &capture.Header{Abi: device.WindowsX86_64}
		capturePath, err := capture.New(ctx, name, h, inputAtoms)
		if err != nil {
			panic(err)
		}
		ctx = capture.Put(ctx, capturePath)

		dependencyGraph, err := dependencygraph.GetDependencyGraph(ctx)
		if err != nil {
			t.Fatalf("%v", err)
		}
		transform := transform.NewDeadCodeElimination(ctx, dependencyGraph)

		expectedAtoms := []api.Cmd{}
		for i, a := range inputAtoms {
			if isLive[a] {
				transform.Request(atom.ID(i))
			}
			if !isDead[a] {
				expectedAtoms = append(expectedAtoms, a)
			}
		}

		w := &test.MockAtomWriter{}
		transform.Flush(ctx, w)

		assert.For(ctx, "Test '%v'", name).ThatSlice(w.Atoms).Equals(expectedAtoms)
	}
}
