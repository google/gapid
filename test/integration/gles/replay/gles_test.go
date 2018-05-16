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

func maybeExportCapture(ctx context.Context, name string, c *path.Capture) {
	if *exportCaptures == "" {
		return
	}
	f, err := os.Create(filepath.Join(*exportCaptures, name+".gfxtrace"))
	assert.With(ctx).ThatError(err).Succeeded()
	defer f.Close()
	err = capture.Export(ctx, c, f)
	assert.With(ctx).ThatError(err).Succeeded()
}

func p(addr uint64) memory.Pointer { return memory.BytePtr(addr) }

type traceVerifier func(context.Context, *path.Capture, *device.Instance)
type traceGenerator func(context.Context, *device.Instance) (*path.Capture, traceVerifier)

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
		assert.With(ctx).ThatError(err).Succeeded()
		lists = append(lists, c.Commands)
		remainingCmds += len(c.Commands)
		threads = append(threads, uint64(0x10000+i))
		if i == 0 {
			d = c.Header.Device
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
	return merged.Capture(ctx)
}

func generateDrawTexturedSquareCapture(ctx context.Context, d *device.Instance) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(128, 128, false, false)
	draw, _ := b.DrawTexturedSquare(ctx)

	verifyTrace := func(ctx context.Context, c *path.Capture, d *device.Instance) {
		defer checkReplay(ctx, c, d, 1)() // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return b.Capture(ctx), verifyTrace
}

func generateDrawTexturedSquareCaptureWithSharedContext(ctx context.Context, d *device.Instance) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(128, 128, true, false)
	draw, _ := b.DrawTexturedSquare(ctx)

	verifyTrace := func(ctx context.Context, c *path.Capture, d *device.Instance) {
		defer checkReplay(ctx, c, d, 1)() // expect a single replay batch.
		checkColorBuffer(ctx, c, d, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return b.Capture(ctx), verifyTrace
}

func generateCaptureWithIssues(ctx context.Context, d *device.Instance) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)

	missingProg := gles.ProgramId(1234)

	textureNames := []gles.TextureId{1}
	textureNamesR := b.Data(ctx, textureNames)

	squareIndicesR := b.Data(ctx, squareIndices)
	squareVerticesR := b.Data(ctx, squareVertices)

	someString := b.Data(ctx, "hello world")

	prog := b.CreateProgram(ctx, textureVSSource, textureFSSource)
	texLoc := b.AddUniformSampler(ctx, prog, "tex")
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	b.Add(
		b.CB.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		b.CB.GlClearColor(0.0, 1.0, 0.0, 1.0),
		b.CB.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_2X_BIT_ATI), // INVALID
		b.CB.GlUseProgram(missingProg),                                                  // INVALID
		b.CB.GlLabelObjectEXT(gles.GLenum_GL_TEXTURE, 123, gles.GLsizei(someString.Range().Size), someString.Ptr()).AddRead(someString.Data()), // INVALID
		b.CB.GlGetError(0),
		b.CB.GlUseProgram(prog),
		b.CB.GlGenTextures(1, textureNamesR.Ptr()).AddWrite(textureNamesR.Data()),
		b.CB.GlActiveTexture(gles.GLenum_GL_TEXTURE0),
		b.CB.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		b.CB.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		b.CB.GlUniform1i(texLoc, 0),
		b.CB.GlEnableVertexAttribArray(posLoc),
		b.CB.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesR.Ptr()),
		b.CB.GlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesR.Ptr()).
			AddRead(squareIndicesR.Data()).
			AddRead(squareVerticesR.Data()),
	)
	b.SwapBuffers()

	verifyTrace := func(ctx context.Context, c *path.Capture, d *device.Instance) {
		defer checkReplay(ctx, c, d, 1)() // expect a single replay batch.
		checkReport(ctx, c, d, b.Cmds, []string{
			"ErrorLevel@[18]: glClear(mask: GLbitfield(16385)): <ERR_INVALID_VALUE_CHECK_EQ [constraint: 16385, value: 16384]>",
			"ErrorLevel@[19]: glUseProgram(program: 1234): <ERR_INVALID_VALUE [value: 1234]>",
			"ErrorLevel@[20]: glLabelObjectEXT(type: GL_TEXTURE, object: 123, length: 12, label: 4216): <ERR_INVALID_OPERATION_OBJECT_DOES_NOT_EXIST [id: 123]>",
		}, nil)
	}

	return b.Capture(ctx), verifyTrace
}

func generateDrawTriangleCapture(ctx context.Context, d *device.Instance) (*path.Capture, traceVerifier) {
	return generateDrawTriangleCaptureEx(ctx, d,
		0.0, 1.0, 0.0,
		1.0, 0.0, 0.0)
}

// generateDrawTriangleCaptureEx generates a capture with several frames containing
// a rotating triangle of color RGB(fr, fg, fb) on a RGB(br, bg, bb) background.
func generateDrawTriangleCaptureEx(ctx context.Context, d *device.Instance,
	br, bg, bb gles.GLfloat,
	fr, fg, fb gles.GLfloat) (*path.Capture, traceVerifier) {

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

	verifyTrace := func(ctx context.Context, c *path.Capture, d *device.Instance) {
		defer checkReplay(ctx, c, d, 1)() // expect a single replay batch.

		done := &sync.WaitGroup{}
		done.Add(5)

		go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-green", clear, done)
		go checkDepthBuffer(ctx, c, d, 64, 64, 0.0, "one-depth", clear, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle", triangle, done)
		go checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle-180", rotatedTriangle, done)
		go checkDepthBuffer(ctx, c, d, 64, 64, 0.01, "triangle-depth", triangle, done)
		done.Wait()
	}

	return b.Capture(ctx), verifyTrace
}

func testTrace(t *testing.T, name string, tg traceGenerator) {
	ctx, d := setup(log.Testing(t))
	c, verifyTrace := tg(ctx, d)
	maybeExportCapture(ctx, name, c)
	verifyTrace(ctx, c, d)
}

func TestDrawTexturedSquare(t *testing.T) {
	testTrace(t, "textured_square", generateDrawTexturedSquareCapture)
}

func TestDrawTexturedSquareWithSharedContext(t *testing.T) {
	testTrace(t, "textured_square_with_shared_context",
		generateDrawTexturedSquareCaptureWithSharedContext)
}

func TestDrawTriangle(t *testing.T) {
	testTrace(t, "draw_triangle", generateDrawTriangleCapture)
}

func TestMultiContextCapture(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	t1, _ := generateDrawTriangleCaptureEx(ctx, d, 1.0, 0.0, 0.0, 1.0, 1.0, 0.0)
	t2, _ := generateDrawTriangleCaptureEx(ctx, d, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0)
	t3, _ := generateDrawTriangleCaptureEx(ctx, d, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0)
	capture := mergeCaptures(ctx, t1, t2, t3)
	maybeExportCapture(ctx, "multi_context", capture)

	contexts, err := resolve.Contexts(ctx, capture.Contexts())
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(len(contexts)).Equals(3)
}

func TestTraceWithIssues(t *testing.T) {
	testTrace(t, "with_issues", generateCaptureWithIssues)
}

func TestExportAndImportCapture(t *testing.T) {
	ctx, d := setup(log.Testing(t))
	c, verifyTrace := generateDrawTriangleCapture(ctx, d)

	var exported bytes.Buffer
	err := capture.Export(ctx, c, &exported)
	assert.With(ctx).ThatError(err).Succeeded()

	ctx, d = setup(log.Testing(t))
	recoveredCapture, err := capture.Import(ctx, "recovered", exported.Bytes())
	assert.With(ctx).ThatError(err).Succeeded()
	verifyTrace(ctx, recoveredCapture, d)
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

	b.ClearColor(0, 0, 1, 1)

	triangle := b.Add(b.CB.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()))

	c := b.Capture(ctx)
	checkColorBuffer(ctx, c, d, 64, 64, 0.01, "triangle_2", triangle, nil)

	maybeExportCapture(ctx, "resize_renderer", c)
}

// TestNewContextUndefined checks that a new context is filled with the
// undefined framebuffer pattern.
func TestNewContextUndefined(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	b := snippets.NewBuilder(ctx, d)
	b.CreateContext(64, 64, false, false)
	makeCurrent := b.Last()
	c := b.Capture(ctx)

	checkColorBuffer(ctx, c, d, 64, 64, 0.0, "undef-fb", makeCurrent, nil)
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
	c := b.Capture(ctx)

	done := &sync.WaitGroup{}
	done.Add(4)
	go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", clear, done)
	go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapA, done)
	go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapB, done)
	go checkColorBuffer(ctx, c, d, 64, 64, 0.0, "solid-blue", swapC, done)
	done.Wait()
}

// TestIssues tests the QueryIssues replay command with various streams.
func TestIssues(t *testing.T) {
	ctx, d := setup(log.Testing(t))

	done := &sync.WaitGroup{}
	cb := gles.CommandBuilder{}

	for _, test := range []struct {
		name     string
		cmds     []api.Cmd
		expected []replay.Issue
	}{
		{
			"glClear - no errors",
			[]api.Cmd{
				cb.GlClearColor(0.0, 0.0, 1.0, 1.0),
				cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
			},
			[]replay.Issue{},
		},
	} {
		b := snippets.NewBuilder(ctx, d)
		b.CreateContext(64, 64, false, true)
		b.Add(test.cmds...)
		c := b.Capture(ctx)

		done.Add(1)

		go checkIssues(ctx, c, d, test.expected, done)
	}

	done.Wait()
}
