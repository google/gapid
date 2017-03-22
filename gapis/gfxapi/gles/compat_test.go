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

package gles_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	_ "github.com/google/gapid/framework/binary/any"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
	"github.com/google/gapid/gapis/memory"
)

var (
	compat     = gles.VisibleForTestingCompat
	glslCompat = gles.VisibleForTestingGlSlCompat
)

func p(addr uint64) memory.Pointer {
	return memory.Pointer{Address: addr, Pool: memory.ApplicationPool}
}

type glShaderSourceCompatTest glslCompatTest

func (c glShaderSourceCompatTest) run(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	capturePath, err := capture.ImportAtomList(ctx, "test", &atom.List{})
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = gles.PutUnusedIDMap(ctx)

	a := device.Little32

	dev := &device.Instance{Configuration: &device.Configuration{
		Drivers: &device.Drivers{
			OpenGL: &device.OpenGLDriver{Version: c.target},
		},
	}}

	transform, err := compat(ctx, dev)
	if err != nil {
		log.E(ctx, "Error creating compatability transform: %v", err)
		return
	}

	shaderType := gles.GLenum_GL_VERTEX_SHADER
	if c.lang == ast.LangFragmentShader {
		shaderType = gles.GLenum_GL_FRAGMENT_SHADER
	}

	mw := &test.MockAtomWriter{S: capture.NewState(ctx)}
	ctxHandle := memory.Pointer{Pool: memory.ApplicationPool, Address: 1}

	eglMakeCurrent := gles.NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle, 0)
	eglMakeCurrent.Extras().Add(gles.NewStaticContextState(), gles.NewDynamicContextState(64, 64, true))
	for _, a := range []atom.Atom{
		gles.NewEglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle),
		eglMakeCurrent,
		gles.NewGlCreateShader(shaderType, 0x10),
		gles.NewGlShaderSource(0x10, 1, p(0x100000), p(0x100010)).
			AddRead(atom.Data(ctx, a, p(0x100000), p(0x100020))).
			AddRead(atom.Data(ctx, a, p(0x100010), int32(len(c.source)))).
			AddRead(atom.Data(ctx, a, p(0x100020), c.source)),
	} {
		transform.Transform(ctx, atom.NoID, a, mw)
	}

	// Find the output glShaderSource atom.
	var cmd *gles.GlShaderSource
	for _, a := range mw.Atoms {
		if a, ok := a.(*gles.GlShaderSource); ok {
			cmd = a
			break
		}
	}

	if cmd == nil {
		t.Error("Transform did not produce a glShaderSource atom. Atoms produced:")
		for i, a := range mw.Atoms {
			t.Errorf("%d %T", i, a)
		}
		return
	}

	if cmd.Count != 1 {
		t.Errorf("Unexpected number of sources: got %d, expected 1.", cmd.Count)
		return
	}

	s := capture.NewState(ctx)
	for _, a := range mw.Atoms {
		a.Mutate(ctx, s, nil)
	}

	srcPtr := cmd.Source.Read(ctx, cmd, s, nil) // 0'th glShaderSource string pointer
	got := strings.TrimRight(string(gfxapi.CharToBytes(srcPtr.StringSlice(ctx, s).Read(ctx, cmd, s, nil))), "\x00")

	expected, err := glslCompat(ctx, c.source, c.lang, dev)
	if err != nil {
		t.Errorf("Unexpected error returned by glslCompat: %v", err)
	}
	if got != expected {
		t.Errorf("Converting to target '%s' produced unexpected output.\nGot:\n%s\nExpected:\n%v",
			c.target, got, expected)
	}
}

func TestGlVertexAttribPointerCompatTest(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	capturePath, err := capture.ImportAtomList(ctx, "test", &atom.List{})
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = gles.PutUnusedIDMap(ctx)

	a := device.Little32

	dev := &device.Instance{Configuration: &device.Configuration{
		Drivers: &device.Drivers{
			OpenGL: &device.OpenGLDriver{Version: OpenGL_3_0},
		},
	}}

	transform, err := compat(ctx, dev)
	if err != nil {
		log.E(ctx, "Error creating compatability transform: %v", err)
		return
	}

	positions := []float32{-1., -1., 1., -1., -1., 1., 1., 1.}
	indices := []uint16{0, 1, 2, 1, 2, 3}
	mw := &test.MockAtomWriter{S: capture.NewState(ctx)}
	ctxHandle := memory.Pointer{Pool: memory.ApplicationPool, Address: 1}
	eglMakeCurrent := gles.NewEglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle, 0)
	eglMakeCurrent.Extras().Add(gles.NewStaticContextState(), gles.NewDynamicContextState(64, 64, true))
	for _, a := range []atom.Atom{
		gles.NewEglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle),
		eglMakeCurrent,
		gles.NewGlEnableVertexAttribArray(0),
		gles.NewGlVertexAttribPointer(0, 2, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 8, p(0x100000)).
			AddRead(atom.Data(ctx, a, p(0x100000), positions)),
		gles.NewGlDrawElements(gles.GLenum_GL_TRIANGLES, gles.GLsizei(len(indices)), gles.GLenum_GL_UNSIGNED_SHORT, p(0x200000)).
			AddRead(atom.Data(ctx, a, p(0x200000), indices)),
	} {
		transform.Transform(ctx, atom.NoID, a, mw)
	}

	// Find glDrawElements and check it is using a buffer instead of client's memory now
	s := capture.NewState(ctx)
	for _, a := range mw.Atoms {
		err := a.Mutate(ctx, s, nil)
		ctx := log.V{"atom": fmt.Sprintf("%T", a)}.Bind(ctx)
		if !assert.For(ctx, "err").ThatError(err).Succeeded() {
			break
		}
		if _, ok := a.(*gles.GlDrawElements); ok {
			ctx := gles.GetContext(s)
			vao := ctx.Instances.VertexArrays[ctx.BoundVertexArray]
			array := vao.VertexAttributeArrays[0]
			binding := vao.VertexBufferBindings[array.Binding]
			if binding.Buffer != 0 && array.Pointer.Address == 0 {
				return // Success
			} else {
				t.Error("glDrawElements does not source vertex data from buffer.")
				return
			}
		}
	}

	t.Error("glDrawElements atom not found.")
	return
}

func TestShaderCompat(t *testing.T) {
	if !config.UseGlslang {
		for _, test := range glslCompatTests {
			glShaderSourceCompatTest(test).run(t)
		}
	}
}
