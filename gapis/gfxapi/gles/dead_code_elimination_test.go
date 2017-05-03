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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

func TestLivenessTree(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	//
	//          root
	//         /    \
	//     child1  child2
	//      /  \
	// childA  childB
	//
	root := StateAddress(1)
	child1 := StateAddress(2)
	child2 := StateAddress(3)
	childA := StateAddress(4)
	childB := StateAddress(5)
	tree := newLivenessTree(map[StateAddress]StateAddress{
		nullStateAddress: nullStateAddress,
		root:             nullStateAddress,
		child1:           root,
		child2:           root,
		childA:           child1,
		childB:           child1,
	})

	tree.MarkLive(child1)
	assert.With(ctx).That(tree.IsLive(root)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(true)

	tree.MarkDead(root)
	tree.MarkLive(child1)
	assert.With(ctx).That(tree.IsLive(root)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(true)

	tree.MarkLive(root)
	assert.With(ctx).That(tree.IsLive(root)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(true)

	tree.MarkDead(child1)
	assert.With(ctx).That(tree.IsLive(root)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(false)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(false)

	tree.MarkDead(root)
	assert.With(ctx).That(tree.IsLive(root)).Equals(false)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(false)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(false)

	tree.MarkLive(childA)
	assert.With(ctx).That(tree.IsLive(root)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child1)).Equals(true)
	assert.With(ctx).That(tree.IsLive(child2)).Equals(false)
	assert.With(ctx).That(tree.IsLive(childA)).Equals(true)
	assert.With(ctx).That(tree.IsLive(childB)).Equals(false)
}

func TestDeadAtomRemoval(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	// Keep the given atom alive in the optimization.
	isLive := map[atom.Atom]bool{}
	live := func(a atom.Atom) atom.Atom { isLive[a] = true; return a }

	// Expect the atom to be removed by the optimization.
	isDead := map[atom.Atom]bool{}
	dead := func(a atom.Atom) atom.Atom { isDead[a] = true; return a }

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
	prologue := []atom.Atom{
		NewEglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1),
		atom.WithExtras(
			NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1, 0),
			NewStaticContextState(), NewDynamicContextState(64, 64, false)),
		NewGlCreateProgram(1),
		NewGlCreateProgram(2),
		atom.WithExtras(NewGlLinkProgram(1), programInfo),
		atom.WithExtras(NewGlLinkProgram(2), programInfo),
		NewGlUseProgram(1),
	}
	allBuffers := GLbitfield_GL_COLOR_BUFFER_BIT | GLbitfield_GL_DEPTH_BUFFER_BIT | GLbitfield_GL_STENCIL_BUFFER_BIT
	tests := map[string][]atom.Atom{
		"Draw calls up to the requested point are preserved": {
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 1, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 2, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 3, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 4, 0)),
		},
		"No request in frame kills draw calls": {
			dead(NewGlClear(allBuffers)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 1, 0)),
			dead(NewEglSwapBuffers(memory.Nullptr, memory.Nullptr, EGLBoolean(1))),
			NewGlClear(allBuffers),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Multiple requests": {
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Simple overwrite": {
			dead(NewGlUniform4fv(0, 1, memory.Nullptr)),
			NewGlUniform4fv(1, 1, memory.Nullptr),
			NewGlUniform4fv(0, 1, memory.Nullptr),
			dead(NewGlVertexAttribPointer(0, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr)),
			NewGlVertexAttribPointer(1, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr),
			NewGlVertexAttribPointer(0, 4, GLenum_GL_FLOAT, GLboolean_GL_FALSE, 0, memory.Nullptr),
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Overwrites should be tracked per program": {
			NewGlUseProgram(1),
			dead(NewGlUniform4fv(0, 1, memory.Nullptr)),
			NewGlUseProgram(2),
			NewGlUniform4fv(0, 1, memory.Nullptr), // Unaffected
			NewGlUseProgram(1),
			NewGlUniform4fv(0, 1, memory.Nullptr),
			NewGlUseProgram(1),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			NewGlUseProgram(2),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Arrays should not interact with scalars": {
			NewGlUniform4fv(0, 10, memory.Nullptr),
			NewGlUniform4fv(0, 1, memory.Nullptr), // Unaffected
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Arrays should not interact with scalars (2)": {
			NewGlUniform4fv(0, 1, memory.Nullptr),
			NewGlUniform4fv(0, 10, memory.Nullptr), // Unaffected
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Unsupported atoms are left unmodified": {
			NewGlUseProgram(1),
			dead(NewGlUniform4fv(0, 1, memory.Nullptr)),
			NewGlUniform1f(0, 3.14), // Not handled in the optimization.
			NewGlLinkProgram(1),     // Not handled in the optimization.
			NewGlUniform4fv(0, 1, memory.Nullptr),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
		},
		"Multiple contexts": {
			// Draw in context 1
			dead(NewGlUniform4fv(0, 1, memory.Nullptr)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			NewGlClear(allBuffers),
			NewGlUniform4fv(0, 1, memory.Nullptr),
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			// Draw in context 2
			NewEglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2),
			atom.WithExtras(
				NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2, 0),
				NewStaticContextState(), NewDynamicContextState(64, 64, false)),
			NewGlCreateProgram(1),
			atom.WithExtras(NewGlLinkProgram(1), programInfo),
			NewGlUseProgram(1),
			dead(NewGlUniform4fv(0, 1, memory.Nullptr)),
			dead(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			NewGlClear(allBuffers),
			NewGlUniform4fv(0, 1, memory.Nullptr),
			NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0),
			// Request from both contexts
			NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle1, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
			NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle2, 0),
			live(NewGlDrawArrays(GLenum_GL_TRIANGLES, 0, 0)),
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

		dependencyGraph, err := GetDependencyGraph(ctx)
		if err != nil {
			t.Fatalf("%v", err)
		}
		transform := newDeadCodeElimination(ctx, dependencyGraph)

		expectedAtoms := []atom.Atom{}
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
