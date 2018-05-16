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
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/image"
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

// storeCapture encodes and writes the command list to the database, returning
// an identifier to the newly constructed and stored Capture.
func (f *Fixture) storeCapture(ctx context.Context, cmds []api.Cmd) *path.Capture {
	dev := f.device.Instance()
	h := &capture.Header{
		Device: dev,
		Abi:    dev.Configuration.ABIs[0],
	}
	out, err := capture.New(ctx, "test-capture", h, cmds)
	assert.With(ctx).ThatError(err).Succeeded()
	return out
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

type Fixture struct {
	mgr          *replay.Manager
	device       bind.Device
	memoryLayout *device.MemoryLayout
	cb           gles.CommandBuilder
}

func newFixture(ctx context.Context) (context.Context, *Fixture) {
	r := bind.NewRegistry()
	ctx = bind.PutRegistry(ctx, r)
	m := replay.New(ctx)
	ctx = replay.PutManager(ctx, m)
	ctx = database.Put(ctx, database.NewInMemory(ctx))
	bind.GetRegistry(ctx).AddDevice(ctx, bind.Host(ctx))

	dev := r.DefaultDevice()
	memoryLayout := dev.Instance().GetConfiguration().ABIs[0].MemoryLayout

	return ctx, &Fixture{
		mgr:          m,
		device:       dev,
		memoryLayout: memoryLayout,
		cb:           gles.CommandBuilder{},
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func p(addr uint64) memory.Pointer { return memory.BytePtr(addr) }

func checkImage(ctx context.Context, name string, got *image.Data, threshold float64) {
	if *generateReferenceImages != "" {
		storeReferenceImage(ctx, *generateReferenceImages, name, got)
	} else {
		quantized := quantizeImage(got)
		expected := loadReferenceImage(ctx, name)
		diff, err := image.Difference(quantized, expected)
		assert.For(ctx, "CheckImage").ThatError(err).Succeeded()
		assert.For(ctx, "CheckImage").ThatFloat(float64(diff)).IsAtMost(threshold)
	}
}

func checkIssues(ctx context.Context, intent replay.Intent, mgr *replay.Manager, expected []replay.Issue, done *sync.WaitGroup) {
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	issues, err := gles.API{}.QueryIssues(ctx, intent, mgr, nil)
	if assert.With(ctx).ThatError(err).Succeeded() {
		assert.With(ctx).ThatSlice(issues).DeepEquals(expected)
	}
}

func checkReport(ctx context.Context, intent replay.Intent, mgr *replay.Manager, cmds []api.Cmd, expected []string, done *sync.WaitGroup) {
	if done != nil {
		defer done.Done()
	}

	report, err := resolve.Report(ctx, intent.Capture.Report(intent.Device, nil))
	assert.With(ctx).ThatError(err).Succeeded()

	got := []string{}
	for _, e := range report.Items {
		if e.Command != nil {
			got = append(got, fmt.Sprintf("%s@%d: %s: %v", e.Severity.String(), e.Command.Indices, cmds[e.Command.Indices[0]], report.Msg(e.Message).Text(nil)))
		} else {
			got = append(got, fmt.Sprintf("%s /%v", e.Severity.String(), report.Msg(e.Message).Text(nil)))
		}
	}
	assert.With(ctx).ThatSlice(got).Equals(expected)
}

func checkColorBuffer(ctx context.Context, intent replay.Intent, mgr *replay.Manager, w, h uint32, threshold float64, name string, after api.CmdID, done *sync.WaitGroup) {
	ctx = log.Enter(ctx, "ColorBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	img, err := gles.API{}.QueryFramebufferAttachment(
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Color0, 0, replay.WireframeMode_None, false, nil)
	if !assert.With(ctx).ThatError(err).Succeeded() {
		return
	}
	checkImage(ctx, name, img, threshold)
}

func checkDepthBuffer(ctx context.Context, intent replay.Intent, mgr *replay.Manager, w, h uint32, threshold float64, name string, after api.CmdID, done *sync.WaitGroup) {
	ctx = log.Enter(ctx, "DepthBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	img, err := gles.API{}.QueryFramebufferAttachment(
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Depth, 0, replay.WireframeMode_None, false, nil)
	if !assert.With(ctx).ThatError(err).Succeeded() {
		return
	}
	checkImage(ctx, name, img, threshold)
}

type intentCfg struct {
	intent replay.Intent
	config replay.Config
}

func (c intentCfg) String() string {
	return fmt.Sprintf("Context: %+v, Config: %+v", c.intent, c.config)
}

func checkReplay(ctx context.Context, expectedIntent replay.Intent, expectedBatchCount int) func() {
	batchCount := 0
	uniqueIntentConfigs := map[intentCfg]struct{}{}
	replay.Events.OnReplay = func(device bind.Device, intent replay.Intent, config replay.Config) {
		assert.For(ctx, "Replay intent").That(intent).DeepEquals(expectedIntent)
		batchCount++
		uniqueIntentConfigs[intentCfg{intent, config}] = struct{}{}
	}
	return func() {
		replay.Events.OnReplay = nil // Avoid stale assertions in subsequent tests that don't use checkReplay.
		if assert.For(ctx, "Batch count").That(batchCount).Equals(expectedBatchCount) {
			log.I(ctx, "%d unique intent-config pairs:", len(uniqueIntentConfigs))
			for cc := range uniqueIntentConfigs {
				log.I(ctx, " • %v", cc)
			}
		}
	}
}

type traceVerifier func(context.Context, *path.Capture, *replay.Manager, bind.Device)
type traceGenerator func(Fixture, context.Context) (*path.Capture, traceVerifier)

// mergeCaptures creates a capture from the cmds of several existing captures, by interleaving them
// arbitrarily, on different threads.
func (f *Fixture) mergeCaptures(ctx context.Context, captures ...*path.Capture) *path.Capture {
	lists := [][]api.Cmd{}
	threads := []uint64{}
	remainingCmds := 0

	for i, path := range captures {
		c, err := capture.ResolveFromPath(ctx, path)
		assert.With(ctx).ThatError(err).Succeeded()
		lists = append(lists, c.Commands)
		remainingCmds += len(c.Commands)
		threads = append(threads, uint64(0x10000+i))
	}

	merged := []api.Cmd{}
	threadIndex := 0
	cmdsUntilSwitchThread, modFourCounter := 4, 3
	for remainingCmds > 0 {
		if cmdsUntilSwitchThread > 0 && len(lists[threadIndex]) > 0 {
			cmd := lists[threadIndex][0]
			cmd.SetThread(threads[threadIndex])
			merged = append(merged, cmd)
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
	return f.storeCapture(ctx, merged)
}

func (f Fixture) generateDrawTexturedSquareCapture(ctx context.Context) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(128, 128, false, false)
	draw, _ := b.DrawTexturedSquare(ctx)
	cmds := b.Cmds

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return f.storeCapture(ctx, cmds), verifyTrace
}

func (f Fixture) generateDrawTexturedSquareCaptureWithSharedContext(ctx context.Context) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(128, 128, true, false)
	draw, _ := b.DrawTexturedSquare(ctx)
	cmds := b.Cmds

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return f.storeCapture(ctx, cmds), verifyTrace
}

func (f Fixture) generateCaptureWithIssues(ctx context.Context) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
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
		f.cb.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		f.cb.GlClearColor(0.0, 1.0, 0.0, 1.0),
		f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_2X_BIT_ATI), // INVALID
		f.cb.GlUseProgram(missingProg),                                                  // INVALID
		f.cb.GlLabelObjectEXT(gles.GLenum_GL_TEXTURE, 123, gles.GLsizei(someString.Range().Size), someString.Ptr()).AddRead(someString.Data()), // INVALID
		f.cb.GlGetError(0),
		f.cb.GlUseProgram(prog),
		f.cb.GlGenTextures(1, textureNamesR.Ptr()).AddWrite(textureNamesR.Data()),
		f.cb.GlActiveTexture(gles.GLenum_GL_TEXTURE0),
		f.cb.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		f.cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		f.cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		f.cb.GlUniform1i(texLoc, 0),
		f.cb.GlEnableVertexAttribArray(posLoc),
		f.cb.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesR.Ptr()),
		f.cb.GlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesR.Ptr()).
			AddRead(squareIndicesR.Data()).
			AddRead(squareVerticesR.Data()),
	)
	b.SwapBuffers()

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkReport(ctx, intent, mgr, b.Cmds, []string{
			"ErrorLevel@[18]: glClear(mask: GLbitfield(16385)): <ERR_INVALID_VALUE_CHECK_EQ [constraint: 16385, value: 16384]>",
			"ErrorLevel@[19]: glUseProgram(program: 1234): <ERR_INVALID_VALUE [value: 1234]>",
			"ErrorLevel@[20]: glLabelObjectEXT(type: GL_TEXTURE, object: 123, length: 12, label: 4216): <ERR_INVALID_OPERATION_OBJECT_DOES_NOT_EXIST [id: 123]>",
		}, nil)
	}

	return f.storeCapture(ctx, b.Cmds), verifyTrace
}

func (f Fixture) generateDrawTriangleCapture(ctx context.Context) (*path.Capture, traceVerifier) {
	return f.generateDrawTriangleCaptureEx(ctx, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0)
}

// generateDrawTriangleCaptureEx generates a capture with several frames containing
// a rotating triangle of color RGB(fr, fg, fb) on a RGB(br, bg, bb) background.
func (f Fixture) generateDrawTriangleCaptureEx(ctx context.Context, br, bg, bb, fr, fg, fb gles.GLfloat) (*path.Capture, traceVerifier) {
	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(64, 64, false, false)

	b.Add(f.cb.GlEnable(gles.GLenum_GL_DEPTH_TEST)) // Required for depth-writing
	b.ClearColor(br, bg, bb, 1.0)
	clear := b.ClearDepth()

	prog := b.CreateProgram(ctx, simpleVSSource, simpleFSSource(fr, fg, fb))
	angleLoc := b.AddUniformSampler(ctx, prog, "angle")
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	triangleVerticesR := b.Data(ctx, triangleVertices)

	b.Add(
		f.cb.GlUseProgram(prog),
		f.cb.GlUniform1f(angleLoc, gles.GLfloat(0)),
		f.cb.GlEnableVertexAttribArray(posLoc),
		f.cb.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
		f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)
	triangle := b.Last()

	angle := 0.0
	for i := 0; i < 30; i++ {
		angle += math.Pi / 30.0
		b.SwapBuffers()
		b.Add(
			f.cb.GlUniform1f(angleLoc, gles.GLfloat(angle)),
			f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
			f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
		)
	}
	rotatedTriangle := b.Last()

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		done := &sync.WaitGroup{}
		done.Add(5)

		go checkColorBuffer(ctx, intent, mgr, 64, 64, 0.0, "solid-green", clear, done)
		go checkDepthBuffer(ctx, intent, mgr, 64, 64, 0.0, "one-depth", clear, done)
		go checkColorBuffer(ctx, intent, mgr, 64, 64, 0.01, "triangle", triangle, done)
		go checkColorBuffer(ctx, intent, mgr, 64, 64, 0.01, "triangle-180", rotatedTriangle, done)
		go checkDepthBuffer(ctx, intent, mgr, 64, 64, 0.01, "triangle-depth", triangle, done)
		done.Wait()
	}

	return f.storeCapture(ctx, b.Cmds), verifyTrace
}

func testTrace(t *testing.T, name string, tg traceGenerator) {
	ctx, f := newFixture(log.Testing(t))
	capture, verifyTrace := tg(*f, ctx)
	maybeExportCapture(ctx, name, capture)
	verifyTrace(ctx, capture, f.mgr, f.device)
}

func TestDrawTexturedSquare(t *testing.T) {
	testTrace(t, "textured_square", Fixture.generateDrawTexturedSquareCapture)
}

func TestDrawTexturedSquareWithSharedContext(t *testing.T) {
	testTrace(t, "textured_square_with_shared_context",
		Fixture.generateDrawTexturedSquareCaptureWithSharedContext)
}

func TestDrawTriangle(t *testing.T) {
	testTrace(t, "draw_triangle", Fixture.generateDrawTriangleCapture)
}

func TestMultiContextCapture(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	t1, _ := f.generateDrawTriangleCaptureEx(ctx, 1.0, 0.0, 0.0, 1.0, 1.0, 0.0)
	t2, _ := f.generateDrawTriangleCaptureEx(ctx, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0)
	t3, _ := f.generateDrawTriangleCaptureEx(ctx, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0)
	capture := f.mergeCaptures(ctx, t1, t2, t3)
	maybeExportCapture(ctx, "multi_context", capture)

	contexts, err := resolve.Contexts(ctx, capture.Contexts())
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(len(contexts)).Equals(3)
}

func TestTraceWithIssues(t *testing.T) {
	testTrace(t, "with_issues", Fixture.generateCaptureWithIssues)
}

func TestExportAndImportCapture(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))
	c, verifyTrace := f.generateDrawTriangleCapture(ctx)

	var exported bytes.Buffer
	err := capture.Export(ctx, c, &exported)
	assert.With(ctx).ThatError(err).Succeeded()

	ctx, f = newFixture(log.Testing(t))
	recoveredCapture, err := capture.Import(ctx, "recovered", exported.Bytes())
	assert.With(ctx).ThatError(err).Succeeded()
	verifyTrace(ctx, recoveredCapture, f.mgr, f.device)
}

// TestResizeRenderer checks that backbuffers can be resized without destroying
// the current context.
func TestResizeRenderer(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(8, 8, false, false) // start with a small backbuffer

	triangleVerticesR := b.Data(ctx, triangleVertices)

	prog := b.CreateProgram(ctx, simpleVSSource, simpleFSSource(1.0, 0.0, 0.0))
	posLoc := b.AddAttributeVec3(ctx, prog, "position")

	b.Add(
		f.cb.GlUseProgram(prog),
		f.cb.GlEnableVertexAttribArray(posLoc),
		f.cb.GlVertexAttribPointer(posLoc, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
	)

	b.ResizeBackbuffer(64, 64)

	b.ClearColor(0, 0, 1, 1)

	triangle := b.Add(f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()))

	capture := f.storeCapture(ctx, b.Cmds)
	intent := replay.Intent{
		Capture: capture,
		Device:  path.NewDevice(f.device.Instance().Id.ID()),
	}

	checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.01, "triangle_2", triangle, nil)

	maybeExportCapture(ctx, "resize_renderer", capture)
}

// TestNewContextUndefined checks that a new context is filled with the
// undefined framebuffer pattern.
func TestNewContextUndefined(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(64, 64, false, false)
	makeCurrent := b.Last()

	intent := replay.Intent{
		Capture: f.storeCapture(ctx, b.Cmds),
		Device:  path.NewDevice(f.device.Instance().Id.ID()),
	}

	checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.0, "undef-fb", makeCurrent, nil)
}

// TestPreserveBuffersOnSwap checks that when the preserveBuffersOnSwap flag is
// set, the backbuffer is preserved between calls to eglSwapBuffers().
func TestPreserveBuffersOnSwap(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
	b.CreateContext(64, 64, false, true)

	clear := b.ClearColor(0, 0, 1, 1)
	swapA := b.SwapBuffers()
	swapB := b.SwapBuffers()
	swapC := b.SwapBuffers()

	intent := replay.Intent{
		Capture: f.storeCapture(ctx, b.Cmds),
		Device:  path.NewDevice(f.device.Instance().Id.ID()),
	}

	done := &sync.WaitGroup{}
	done.Add(4)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.0, "solid-blue", clear, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.0, "solid-blue", swapA, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.0, "solid-blue", swapB, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.0, "solid-blue", swapC, done)
	done.Wait()
}

// TestIssues tests the QueryIssues replay command with various streams.
func TestIssues(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	done := &sync.WaitGroup{}

	for _, test := range []struct {
		name     string
		cmds     []api.Cmd
		expected []replay.Issue
	}{
		{
			"glClear - no errors",
			[]api.Cmd{
				f.cb.GlClearColor(0.0, 0.0, 1.0, 1.0),
				f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
			},
			[]replay.Issue{},
		},
	} {
		b := snippets.NewBuilder(ctx, f.cb, f.memoryLayout)
		b.CreateContext(64, 64, false, true)
		b.Add(test.cmds...)

		intent := replay.Intent{
			Capture: f.storeCapture(ctx, b.Cmds),
			Device:  path.NewDevice(f.device.Instance().Id.ID()),
		}
		done.Add(1)
		go checkIssues(ctx, intent, f.mgr, test.expected, done)
	}

	done.Wait()
}
