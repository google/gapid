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
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

var compat = gles.VisibleForTestingCompat

const OpenGL_3_0 = "3.0"

var p = memory.BytePtr

func newState(ctx context.Context) *api.GlobalState {
	s, err := capture.NewState(ctx)
	if err != nil {
		panic(err)
	}
	return s
}

func TestGlVertexAttribPointerCompatTest(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	a := arena.New()
	defer a.Dispose()

	h := &capture.Header{ABI: device.AndroidARMv7a}
	ml := h.ABI.MemoryLayout
	cap, err := capture.NewGraphicsCapture(ctx, a, "test", h, nil, []api.Cmd{})
	if err != nil {
		panic(err)
	}

	capturePath, err := cap.Path(ctx)
	if err != nil {
		panic(err)
	}

	ctx = capture.Put(ctx, capturePath)
	ctx = gles.PutUnusedIDMap(ctx)

	dev := &device.Instance{Configuration: &device.Configuration{
		Drivers: &device.Drivers{
			Opengl: &device.OpenGLDriver{Version: OpenGL_3_0},
		},
	}}

	onCompatError := func(ctx context.Context, id api.CmdID, cmd api.Cmd, err error) {
		log.E(ctx, "%v %v: %v", id, cmd, err)
	}

	ct, err := compat(ctx, dev, onCompatError)
	if err != nil {
		log.E(ctx, "Error creating compatability transform: %v", err)
		return
	}

	positions := []float32{-1., -1., 1., -1., -1., 1., 1., 1.}
	indices := []uint16{0, 1, 2, 1, 2, 3}
	r := &transform.Recorder{S: newState(ctx)}
	ctxHandle, displayHandle, surfaceHandle := p(1), p(2), p(3)
	cb := gles.CommandBuilder{Thread: 0, Arena: a}
	eglMakeCurrent := cb.EglMakeCurrent(displayHandle, surfaceHandle, surfaceHandle, ctxHandle, 0)
	eglMakeCurrent.Extras().Add(gles.NewStaticContextStateForTest(a), gles.NewDynamicContextStateForTest(a, 64, 64, true))

	cmds := []api.Cmd{
		cb.EglCreateContext(displayHandle, memory.Nullptr, memory.Nullptr, memory.Nullptr, ctxHandle),
		eglMakeCurrent,
	}

	vertId := gles.ShaderId(1)
	fragId := gles.ShaderId(2)
	progId := gles.ProgramId(3)
	attIdx := gles.AttributeLocation(0)
	vertSrc := "void main() {}"
	fragSrc := vertSrc

	cmds = append(cmds, gles.BuildProgram(ctx, r.S, cb, vertId, fragId, progId, vertSrc, fragSrc)...)

	resources := gles.MakeActiveProgramResourcesʳ(a)
	attribute := gles.MakeProgramResourceʳ(a)
	attribute.Locations().Add(0, gles.GLint(attIdx))
	resources.ProgramInputs().Add(uint32(attIdx), attribute)
	lpe := gles.MakeLinkProgramExtra(a)
	lpe.SetLinkStatus(gles.GLboolean_GL_TRUE)
	lpe.SetActiveResources(resources)
	cmds = append(cmds, api.WithExtras(cb.GlLinkProgram(progId), lpe))

	cmds = append(cmds, []api.Cmd{
		cb.GlUseProgram(progId),
		cb.GlEnableVertexAttribArray(attIdx),
		cb.GlVertexAttribPointer(attIdx, 2, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 8, p(0x100000)).
			AddRead(memory.Store(ctx, ml, p(0x100000), positions)),
		cb.GlDrawElements(gles.GLenum_GL_TRIANGLES, gles.GLsizei(len(indices)), gles.GLenum_GL_UNSIGNED_SHORT, p(0x200000)).
			AddRead(memory.Store(ctx, ml, p(0x200000), indices)),
	}...)

	api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		ct.Transform(ctx, api.CmdNoID, cmd, r)
		return nil
	})

	// Find glDrawElements and check it is using a buffer instead of client's memory now
	s := newState(ctx)
	var found bool
	err = api.ForeachCmd(ctx, r.Cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}

		if _, ok := cmd.(*gles.GlDrawElements); ok {
			ctx := gles.GetContext(s, cmd.Thread())
			vao := ctx.Bound().VertexArray()
			array := vao.VertexAttributeArrays().Get(0)
			binding := array.Binding()
			if !binding.Buffer().IsNil() && array.Pointer() == 0 {
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
