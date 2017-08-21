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
	"context"
	"strings"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/testcmd"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

var (
	compat     = gles.VisibleForTestingCompat
	glslCompat = gles.VisibleForTestingGlSlCompat
)

func p(addr uint64) memory.Pointer {
	return memory.BytePtr(addr, memory.ApplicationPool)
}

type glShaderSourceCompatTest glslCompatTest

func newState(ctx context.Context) *api.State {
	s, err := capture.NewState(ctx)
	if err != nil {
		panic(err)
	}
	return s
}

func (c glShaderSourceCompatTest) run(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	h := &capture.Header{Abi: device.AndroidARMv7a}
	a := h.Abi.MemoryLayout
	capturePath, err := capture.New(ctx, "test", h, []api.Cmd{})
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = gles.PutUnusedIDMap(ctx)

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

	mw := &testcmd.Writer{S: newState(ctx)}
	ctxHandle := memory.BytePtr(1, memory.ApplicationPool)

	cb := gles.CommandBuilder{Thread: 0}
	eglMakeCurrent := cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle, 0)
	eglMakeCurrent.Extras().Add(gles.NewStaticContextState(), gles.NewDynamicContextState(64, 64, true))
	for _, a := range []api.Cmd{
		cb.EglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle),
		eglMakeCurrent,
		cb.GlCreateShader(shaderType, 0x10),
		cb.GlShaderSource(0x10, 1, p(0x100000), p(0x100010)).
			AddRead(memory.Store(ctx, a, p(0x100000), p(0x100020))).
			AddRead(memory.Store(ctx, a, p(0x100010), int32(len(c.source)))).
			AddRead(memory.Store(ctx, a, p(0x100020), c.source)),
	} {
		transform.Transform(ctx, api.CmdNoID, a, mw)
	}

	// Find the output glShaderSource command.
	var cmd *gles.GlShaderSource
	for _, c := range mw.Cmds {
		if c, ok := c.(*gles.GlShaderSource); ok {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Error("Transform did not produce a glShaderSource command. Commands produced:")
		for i, c := range mw.Cmds {
			t.Errorf("%d %T", i, c)
		}
		return
	}

	if cmd.Count != 1 {
		t.Errorf("Unexpected number of sources: got %d, expected 1.", cmd.Count)
		return
	}

	s := newState(ctx)
	for _, c := range mw.Cmds {
		c.Mutate(ctx, s, nil)
	}

	srcPtr := cmd.Source.Read(ctx, cmd, s, nil) // 0'th glShaderSource string pointer
	got := strings.TrimRight(string(memory.CharToBytes(srcPtr.StringSlice(ctx, s).Read(ctx, cmd, s, nil))), "\x00")

	expected, err := glslCompat(ctx, c.source, c.lang, nil, dev)
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

	h := &capture.Header{Abi: device.AndroidARMv7a}
	a := h.Abi.MemoryLayout
	capturePath, err := capture.New(ctx, "test", h, []api.Cmd{})
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = gles.PutUnusedIDMap(ctx)

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
	mw := &testcmd.Writer{S: newState(ctx)}
	ctxHandle := memory.BytePtr(1, memory.ApplicationPool)
	cb := gles.CommandBuilder{Thread: 0}
	eglMakeCurrent := cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle, 0)
	eglMakeCurrent.Extras().Add(gles.NewStaticContextState(), gles.NewDynamicContextState(64, 64, true))
	api.ForeachCmd(ctx, []api.Cmd{
		cb.EglCreateContext(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle),
		eglMakeCurrent,
		cb.GlEnableVertexAttribArray(0),
		cb.GlVertexAttribPointer(0, 2, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 8, p(0x100000)).
			AddRead(memory.Store(ctx, a, p(0x100000), positions)),
		cb.GlDrawElements(gles.GLenum_GL_TRIANGLES, gles.GLsizei(len(indices)), gles.GLenum_GL_UNSIGNED_SHORT, p(0x200000)).
			AddRead(memory.Store(ctx, a, p(0x200000), indices)),
	}, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		transform.Transform(ctx, api.CmdNoID, cmd, mw)
		return nil
	})

	// Find glDrawElements and check it is using a buffer instead of client's memory now
	s := newState(ctx)
	var found bool
	err = api.ForeachCmd(ctx, mw.Cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, s, nil); err != nil {
			return err
		}

		if _, ok := cmd.(*gles.GlDrawElements); ok {
			ctx := gles.GetContext(s, cmd.Thread())
			vao := ctx.Bound.VertexArray
			array := vao.VertexAttributeArrays[0]
			binding := vao.VertexBufferBindings[array.Binding]
			if binding.Buffer != 0 && array.Pointer.Address() == 0 {
				found = true
				return api.Break // Success
			} else {
				t.Error("glDrawElements does not source vertex data from buffer.")
				return api.Break
			}
		}
		return nil
	})
	assert.For(ctx, "err").ThatError(err).Succeeded()

	if !found {
		t.Error("glDrawElements command not found.")
	}
}

func TestShaderCompat(t *testing.T) {
	if !config.UseGlslang {
		for _, test := range glslCompatTests {
			glShaderSourceCompatTest(test).run(t)
		}
	}
}
