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

package replay

import (
	"bytes"
	"context"
	"flag"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/os/device/bind"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/gles"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/test/integration/gles/snippets"
)

const replayTimeout = time.Second * 5

var (
	triangleVertices = []float32{
		+0.0, -0.5, 0.1,
		-0.5, +0.5, 0.5,
		+0.5, +0.5, 0.9,
	}

	squareVertices = []float32{
		-0.5, -0.5, 0.5,
		-0.5, +0.5, 0.5,
		+0.5, +0.5, 0.5,
		+0.5, -0.5, 0.5,
	}

	squareIndices = []uint16{
		0, 1, 2, 0, 2, 3,
	}

	generateReferenceImages = flag.String("generate", "", "directory in which to generate reference images, empty to disable")
	exportCaptures          = flag.String("export-captures", "", "directory to export captures to, empty to disable")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func setup(ctx context.Context) (context.Context, *device.Instance) {
	r := bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, r)
	m := replay.New(ctx)
	ctx = replay.PutManager(ctx, m)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	bind.GetRegistry(ctx).AddDevice(ctx, bind.Host(ctx))
	return ctx, r.DefaultDevice().Instance()
}

func buildAndMaybeExportCapture(ctx context.Context, b *snippets.Builder, name string) *path.Capture {
	c := b.Capture(ctx, name)
	maybeExportCapture(ctx, c)
	return c
}

func maybeExportCapture(ctx context.Context, c *path.Capture) {
	if *exportCaptures == "" {
		return
	}
	cap, err := capture.ResolveFromPath(ctx, c)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	f, err := os.Create(filepath.Join(*exportCaptures, cap.Name()+".gfxtrace"))
	assert.For(ctx, "err").ThatError(err).Succeeded()
	defer f.Close()
	err = capture.Export(ctx, c, f)
	assert.For(ctx, "err").ThatError(err).Succeeded()
}

func p(addr uint64) memory.Pointer { return memory.BytePtr(addr) }

type verifier func(context.Context, *path.Capture, *device.Instance)
type generator func(context.Context, *device.Instance) (*path.Capture, verifier)

// mergeCaptures creates a capture from the cmds of several existing captures,
// by interleaving them arbitrarily, on different threads.
func mergeCaptures(ctx context.Context, captures ...*path.Capture) *path.Capture {
	lists := [][]api.Cmd{}
	threads := []uint64{}
	remainingCmds := 0

	if len(captures) == 0 {
		panic("mergeCaptures requires at least one capture")
	}

	var d *device.Instance
	for i, path := range captures {
		c, err := capture.ResolveFromPath(ctx, path)
		assert.For(ctx, "err").ThatError(err).Succeeded()
		gc := c.(*capture.GraphicsCapture)
		lists = append(lists, gc.Commands)
		remainingCmds += len(gc.Commands)
		threads = append(threads, uint64(0x10000+i))
		if i == 0 {
			d = gc.Header.Device
		}
	}

	merged := snippets.NewBuilder(ctx, d)
	threadIndex := 0
	cmdsUntilSwitchThread, modFourCounter := 4, 3
	for remainingCmds > 0 {
		if cmdsUntilSwitchThread > 0 && len(lists[threadIndex]) > 0 {
			cmd := lists[threadIndex][0]
			cmd.SetThread(threads[threadIndex])
			merged.Add(cmd)
			lists[threadIndex] = lists[threadIndex][1:]
			remainingCmds--
			cmdsUntilSwitchThread--
		} else {
			threadIndex = (threadIndex + 1) % len(lists)
			for len(lists[threadIndex]) == 0 {
				threadIndex = (threadIndex + 1) % len(lists)
			}
			// We don't want to always switch threads after the same number of commands,
			// but we want it to be predictable. This should do.
			cmdsUntilSwitchThread = 2 + modFourCounter
			modFourCounter = (modFourCounter + 1) % 4
		}
	}
	return merged.Capture(ctx, "merged")
}

func generateDrawTriangleCapture(ctx context.Context, d *device.Instance) (*path.Capture, verifier) {
	return generateDrawTriangleCaptureEx(ctx, d,
		0.0, 1.0, 0.0,
		1.0, 0.0, 0.0)
}

// generateDrawTriangleCaptureEx generates a capture with several frames containing
// a rotating triangle of color RGB(fr, fg, fb) on a RGB(br, bg, bb) background.
func generateDrawTriangleCaptureEx(ctx context.Context, d *device.Instance,
	br, bg, bb gles.GLfloat,
	fr, fg, fb gles.GLfloat) (*path.Capture, verifier) {

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)

	b.Add(b.CB.GlEnable(gles.GLenum_GL_DEPTH_TEST)) // Required for depth-writing
	b.ClearColor(br, bg, bb, 1.0)
	clear := b.ClearDepth()

	prog := b.CreateProgram(ctx, simpleVSSource, simpleFSSource(fr, fg, fb))
	angleLoc := b.AddUniformSampler(ctx, prog, "angle")
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	triangleVerticesR := b.Data(ctx, triangleVertices)

	b.Add(
		b.CB.GlUseProgram(prog),
		b.CB.GlUniform1f(angleLoc, gles.GLfloat(0)),
		b.CB.GlEnableVertexAttribArray(posLoc),
		b.CB.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
		b.CB.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)
	triangle := b.Last()

	angle := 0.0
	for i := 0; i < 30; i++ {
		angle += math.Pi / 30.0
		b.SwapBuffers()
		b.Add(
			b.CB.GlUniform1f(angleLoc, gles.GLfloat(angle)),
			b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
			b.CB.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
		)
	}
	rotatedTriangle := b.Last()

	verify := func(ctx context.Context, c *path.Capture, d *device.Instance) {
		checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
			done := &sync.WaitGroup{}
			done.Add(5)
			go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-green", clear, done)
			go checkDepthBuffer(ctx, c, d, 64, 64, 0.0, "one-depth", clear, done)
			go checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle", triangle, done)
			go checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle-180", rotatedTriangle, done)
			go checkDepthBuffer(ctx, c, d, 64, 64, 0.01, "triangle-depth", triangle, done)
			done.Wait()
		})
	}

	return buildAndMaybeExportCapture(ctx, b, "draw-triangle"), verify
}

func test(t *testing.T, name string, tg generator) {
	ctx, d := setup(log.Testing(t))
	c, verify := tg(ctx, d)
	verify(ctx, c, d)
}

func TestMultiContextCapture(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	t1, _ := generateDrawTriangleCaptureEx(ctx, d, 1.0, 0.0, 0.0, 1.0, 1.0, 0.0)
	t2, _ := generateDrawTriangleCaptureEx(ctx, d, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0)
	t3, _ := generateDrawTriangleCaptureEx(ctx, d, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0)
	c := mergeCaptures(ctx, t1, t2, t3)
	maybeExportCapture(ctx, c)

	contexts, err := resolve.Contexts(ctx, c.Contexts(), nil)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	assert.For(ctx, "len").That(len(contexts)).Equals(3)
}

func TestExportAndImportCapture(t *testing.T) {
	ctx, d := setup(log.Testing(t))
	c, verify := generateDrawTriangleCapture(ctx, d)

	var exported bytes.Buffer
	err := capture.Export(ctx, c, &exported)
	assert.For(ctx, "err").ThatError(err).Succeeded()

	ctx, d = setup(log.Testing(t))
	src := &capture.Blob{Data: exported.Bytes()}
	recoveredCapture, err := capture.Import(ctx, "key", "recovered", src)
	assert.For(ctx, "err").ThatError(err).Succeeded()
	verify(ctx, recoveredCapture, d)
}

// TestResizeRenderer checks that backbuffers can be resized without destroying
// the current context.
func TestResizeRenderer(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(8, 8, false, false) // start with a small backbuffer

	triangleVerticesR := b.Data(ctx, triangleVertices)

	prog := b.CreateProgram(ctx, simpleVSSource, simpleFSSource(1.0, 0.0, 0.0))
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	b.Add(
		b.CB.GlUseProgram(prog),
		b.CB.GlEnableVertexAttribArray(posLoc),
		b.CB.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
	)

	b.ResizeBackbuffer(64, 64)
	b.Add(b.CB.GlViewport(0, 0, 64, 64))
	b.ClearColor(0, 0, 1, 1)
	triangle := b.Add(b.CB.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()))
	c := buildAndMaybeExportCapture(ctx, b, "resize-renderer")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle_2", triangle, nil)
	})
}

// TestNewContextUndefined checks that a new context is filled with the
// undefined framebuffer pattern.
func TestNewContextUndefined(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)
	makeCurrent := b.Last()
	c := buildAndMaybeExportCapture(ctx, b, "new-context-undefined")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 64, 64, 0.01, "undef-fb", makeCurrent, nil)
	})
}

// TestPreserveBuffersOnSwap checks that when the preserveBuffersOnSwap flag is
// set, the backbuffer is preserved between calls to eglSwapBuffers().
func TestPreserveBuffersOnSwap(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, true)

	clear := b.ClearColor(0, 0, 1, 1)
	swapA := b.SwapBuffers()
	swapB := b.SwapBuffers()
	swapC := b.SwapBuffers()
	c := buildAndMaybeExportCapture(ctx, b, "preserve-buffers-on-swap")

	checkReplay(ctx, c, d, 1, func() { // expect a single replay batch.
		done := &sync.WaitGroup{}
		done.Add(4)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", clear, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapA, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapB, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapC, done)
		done.Wait()
	})
}
