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
	"bytes"
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/id"
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
	"github.com/google/gapid/test/integration/replay/gles/samples"
)

const (
	replayTimeout = time.Second * 5

	simpleVSSource = `
		precision mediump float;
		attribute vec3 position;
		uniform float angle;

		void main() {
			float c = cos(angle);
			float s = sin(angle);
		        mat3 rotation = mat3(vec3(c, -s, 0.0), vec3(s, c, 0.0), vec3(0.0, 0.0, 1.0));
			gl_Position = vec4(rotation * position, 1.0);
		}`

	simpleFSSourceTemplate = `
		precision mediump float;
		void main() {
			gl_FragColor = vec4(%f, %f, %f, 1.0);
		}`

	textureVSSource = `
		precision mediump float;
		attribute vec3 position;
		varying vec2 texcoord;
		void main() {
			gl_Position = vec4(position, 1.0);
			texcoord = position.xy + vec2(0.5, 0.5);
		}`

	textureFSSource = `
		precision mediump float;
		uniform sampler2D tex;
		varying vec2 texcoord;
		void main() {
			gl_FragColor = texture2D(tex, texcoord);
		}`
)

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
	rootCtx                 context.Context

	eglDisplay = p(1)
)

func storeCommands(ctx context.Context, cmds []api.Cmd) id.ID {
	id, err := database.Store(ctx, cmds)
	if err != nil {
		log.F(ctx, "Failed to store command stream: %v", err)
	}
	return id
}

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

func (f *Fixture) newID() uint {
	return uint(atomic.AddUint32(&f.nextID, 1))
}

func simpleFSSource(fr, fg, fb gles.GLfloat) string {
	return fmt.Sprintf(simpleFSSourceTemplate, fr, fg, fb)
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
	s            *api.GlobalState
	nextID       uint32
	cb           gles.CommandBuilder
}

// p returns a unique pointer. Meant to be used to generate
// pointers representing driver-side data, so the allocation
// itself is not relevant.
func (f *Fixture) p(ctx context.Context) memory.Pointer {
	base, err := f.s.Allocator.Alloc(8, 8)
	assert.With(ctx).ThatError(err).Succeeded()
	return memory.BytePtr(base, memory.ApplicationPool)
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
	s := api.NewStateWithEmptyAllocator(memoryLayout)

	return ctx, &Fixture{
		mgr:          m,
		device:       dev,
		memoryLayout: memoryLayout,
		s:            s,
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	var cancel task.CancelFunc
	rootCtx, cancel = task.WithCancel(context.Background())
	code := m.Run()
	cancel()
	app.WaitForCleanup(rootCtx)
	os.Exit(code)
}

func p(addr uint64) memory.Pointer {
	return memory.BytePtr(addr, memory.ApplicationPool)
}

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
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Color0, 0, replay.WireframeMode_None, nil)
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
		ctx, intent, mgr, []uint64{uint64(after)}, w, h, api.FramebufferAttachment_Depth, 0, replay.WireframeMode_None, nil)
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

func (f *Fixture) initContext(ctx context.Context, width, height int, preserveBuffersOnSwap bool) (cmds []api.Cmd, eglContext, eglSurface memory.Pointer) {
	eglContext = f.p(ctx)
	eglSurface = f.p(ctx)
	eglConfig := f.p(ctx)

	eglShareContext := memory.Nullptr
	// TODO: We don't observe attribute lists properly. We should.
	cmds = []api.Cmd{
		f.cb.EglGetDisplay(gles.EGLNativeDisplayType(0), eglDisplay),
		f.cb.EglInitialize(eglDisplay, memory.Nullptr, memory.Nullptr, gles.EGLBoolean(1)),
		f.cb.EglCreateContext(eglDisplay, eglConfig, eglShareContext, f.p(ctx), eglContext),
		f.makeCurrent(eglSurface, eglContext, width, height, preserveBuffersOnSwap),
	}
	return cmds, eglContext, eglSurface
}

func (f *Fixture) makeCurrent(eglSurface, eglContext memory.Pointer, width, height int, preserveBuffersOnSwap bool) api.Cmd {
	eglTrue := gles.EGLBoolean(1)
	return api.WithExtras(
		f.cb.EglMakeCurrent(eglDisplay, eglSurface, eglSurface, eglContext, eglTrue),
		gles.NewStaticContextState(),
		gles.NewDynamicContextState(width, height, preserveBuffersOnSwap),
	)
}

func TestClear(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	atoms, red, green, blue, black := samples.ClearBackbuffer(ctx, f.cb, f.memoryLayout)

	capture := f.storeCapture(ctx, atoms)

	intent := replay.Intent{
		Capture: capture,
		Device:  path.NewDevice(f.device.Instance().Id.ID()),
	}

	defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

	done := &sync.WaitGroup{}
	done.Add(4)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0, "solid-red", red, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0, "solid-green", green, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0, "solid-blue", blue, done)
	go checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0, "solid-black", black, done)
	done.Wait()

	maybeExportCapture(ctx, "clear", capture)
}

type traceVerifier func(context.Context, *path.Capture, *replay.Manager, bind.Device)
type traceGenerator func(Fixture, context.Context) (*path.Capture, traceVerifier)

// mergeCaptures creates a capture from the atoms of several existing captures, by interleaving them
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
	atoms, draw, _ := samples.DrawTexturedSquare(ctx, f.cb, false, f.memoryLayout)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return f.storeCapture(ctx, atoms), verifyTrace
}

func (f Fixture) generateDrawTexturedSquareCaptureWithSharedContext(ctx context.Context) (*path.Capture, traceVerifier) {
	atoms, draw, _ := samples.DrawTexturedSquare(ctx, f.cb, true, f.memoryLayout)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", draw, nil)
	}

	return f.storeCapture(ctx, atoms), verifyTrace
}

func (f Fixture) generateCaptureWithIssues(ctx context.Context) (*path.Capture, traceVerifier) {
	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	missingProg := gles.ProgramId(f.newID())
	cmds, _, eglSurface := f.initContext(ctx, 128, 128, false)
	texLoc := gles.UniformLocation(0)

	s := api.NewStateWithEmptyAllocator(f.memoryLayout)

	textureNames := []gles.TextureId{1}
	textureNamesR := f.s.AllocDataOrPanic(ctx, textureNames)

	squareIndicesR := f.s.AllocDataOrPanic(ctx, squareIndices)
	squareVerticesR := f.s.AllocDataOrPanic(ctx, squareVertices)

	someString := f.s.AllocDataOrPanic(ctx, "hello world")

	cmds = append(cmds,
		gles.BuildProgram(ctx, s, f.cb, vs, fs, prog, textureVSSource, textureFSSource)...,
	)

	cmds = append(cmds,
		f.cb.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		f.cb.GlClearColor(0.0, 1.0, 0.0, 1.0),
		f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_2X_BIT_ATI),

		api.WithExtras(
			f.cb.GlLinkProgram(prog),
			&gles.ProgramInfo{
				LinkStatus: gles.GLboolean_GL_TRUE,
				ActiveUniforms: gles.NewUniformIndexːActiveUniformᵐ().Add(0, gles.ActiveUniform{
					Type:      gles.GLenum_GL_SAMPLER_2D,
					Name:      "tex",
					ArraySize: 1,
					Location:  texLoc,
				}),
			}),
		f.cb.GlUseProgram(missingProg),
		f.cb.GlLabelObjectEXT(gles.GLenum_GL_TEXTURE, 123, gles.GLsizei(someString.Range().Size), someString.Ptr()).AddRead(someString.Data()),
		f.cb.GlGetError(0),
		f.cb.GlUseProgram(prog),
		f.cb.GlGenTextures(1, textureNamesR.Ptr()).AddWrite(textureNamesR.Data()),
		f.cb.GlGetUniformLocation(prog, "tex", texLoc),
		f.cb.GlActiveTexture(gles.GLenum_GL_TEXTURE0),
		f.cb.GlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		f.cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		f.cb.GlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		f.cb.GlUniform1i(texLoc, 0),
		f.cb.GlGetAttribLocation(prog, "position", gles.GLint(pos)),
		f.cb.GlEnableVertexAttribArray(pos),
		f.cb.GlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesR.Ptr()),
		f.cb.GlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesR.Ptr()).
			AddRead(squareIndicesR.Data()).
			AddRead(squareVerticesR.Data()),
		f.cb.EglSwapBuffers(eglDisplay, eglSurface, gles.EGLBoolean(1)),
	)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkReport(ctx, intent, mgr, cmds, []string{
			"ErrorLevel@[15]: glClear(mask: GLbitfield(16385)): <ERR_INVALID_VALUE_CHECK_EQ [constraint: 16385, value: 16384]>",
			"ErrorLevel@[17]: glUseProgram(program: 4): <ERR_INVALID_VALUE [value: 4]>",
			"ErrorLevel@[18]: glLabelObjectEXT(type: GL_TEXTURE, object: 123, length: 12, label: {4208 0}): <ERR_INVALID_OPERATION_OBJECT_DOES_NOT_EXIST [id: 123]>",
		}, nil)
	}

	return f.storeCapture(ctx, cmds), verifyTrace
}

func (f Fixture) generateDrawTriangleCapture(ctx context.Context) (*path.Capture, traceVerifier) {
	return f.generateDrawTriangleCaptureEx(ctx, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0)
}

// generateDrawTriangleCaptureEx generates a capture with several frames containing
// a rotating triangle of color RGB(fr, fg, fb) on a RGB(br, bg, bb) background.
func (f Fixture) generateDrawTriangleCaptureEx(ctx context.Context, br, bg, bb, fr, fg, fb gles.GLfloat) (*path.Capture, traceVerifier) {
	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	angleLoc := gles.UniformLocation(0)
	cmds, _, eglSurface := f.initContext(ctx, 64, 64, false)

	cmds = append(cmds,
		f.cb.GlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
	)

	cmds = append(cmds,
		f.cb.GlClearColor(br, bg, bb, 1.0),
		f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
	)
	clear := api.CmdID(len(cmds) - 1)
	cmds = append(cmds,
		gles.BuildProgram(ctx, f.s, f.cb, vs, fs, prog, simpleVSSource, simpleFSSource(fr, fg, fb))...,
	)

	triangleVerticesR := f.s.AllocDataOrPanic(ctx, triangleVertices)

	cmds = append(cmds,
		api.WithExtras(
			f.cb.GlLinkProgram(prog),
			&gles.ProgramInfo{
				LinkStatus: gles.GLboolean_GL_TRUE,
				ActiveUniforms: gles.NewUniformIndexːActiveUniformᵐ().Add(0, gles.ActiveUniform{
					Type:      gles.GLenum_GL_FLOAT,
					Name:      "angle",
					ArraySize: 1,
					Location:  angleLoc,
				}),
			}),
		f.cb.GlUseProgram(prog),
		f.cb.GlGetUniformLocation(prog, "angle", angleLoc),
		f.cb.GlUniform1f(angleLoc, gles.GLfloat(0)),
		f.cb.GlGetAttribLocation(prog, "position", gles.GLint(pos)),
		f.cb.GlEnableVertexAttribArray(pos),
		f.cb.GlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
		f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)
	triangle := api.CmdID(len(cmds) - 1)

	angle := 0.0
	for i := 0; i < 30; i++ {
		angle += math.Pi / 30.0
		cmds = append(cmds,
			f.cb.EglSwapBuffers(eglDisplay, eglSurface, gles.EGLBoolean(1)),
			f.cb.GlUniform1f(angleLoc, gles.GLfloat(angle)),
			f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
			f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
		)
	}
	rotatedTriangle := api.CmdID(len(cmds) - 1)

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

	return f.storeCapture(ctx, cmds), verifyTrace
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

	triangleVerticesR := f.s.AllocDataOrPanic(ctx, triangleVertices)

	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	cmds, eglContext, eglSurface := f.initContext(ctx, 8, 8, false) // start with a small backbuffer
	cmds = append(cmds,
		gles.BuildProgram(ctx, f.s, f.cb, vs, fs, prog, simpleVSSource, simpleFSSource(1.0, 0.0, 0.0))...,
	)
	cmds = append(cmds,
		f.cb.GlLinkProgram(prog),
		f.cb.GlUseProgram(prog),
		f.cb.GlGetAttribLocation(prog, "position", gles.GLint(pos)),
		f.cb.GlEnableVertexAttribArray(pos),
		f.cb.GlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
	)
	cmds = append(cmds,
		f.makeCurrent(eglSurface, eglContext, 64, 64, false),
		f.cb.GlClearColor(0.0, 0.0, 1.0, 1.0),
		f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
		f.cb.GlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)
	triangle := api.CmdID(len(cmds) - 1)
	capture := f.storeCapture(ctx, cmds)
	intent := replay.Intent{
		Capture: capture,
		Device:  path.NewDevice(f.device.Instance().Id.ID()),
	}

	checkColorBuffer(ctx, intent, f.mgr, 64, 64, 0.01, "triangle_2", triangle, nil)

	maybeExportCapture(ctx, "resize_renderer", capture)
}

// TestPreserveBuffersOnSwap checks that when the preserveBuffersOnSwap flag is
// set, the backbuffer is preserved between calls to eglSwapBuffers().
func TestPreserveBuffersOnSwap(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	cmds, _, _ := f.initContext(ctx, 64, 64, true)
	cmds = append(cmds,
		f.cb.GlClearColor(0.0, 0.0, 1.0, 1.0),
		f.cb.GlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	clear := api.CmdID(len(cmds) - 1)
	cmds = append(cmds, f.cb.EglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))
	swapA := api.CmdID(len(cmds) - 1)
	cmds = append(cmds, f.cb.EglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))
	swapB := api.CmdID(len(cmds) - 1)
	cmds = append(cmds, f.cb.EglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))
	swapC := api.CmdID(len(cmds) - 1)

	intent := replay.Intent{
		Capture: f.storeCapture(ctx, cmds),
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
		cmds, _, _ := f.initContext(ctx, 64, 64, true)
		cmds = append(cmds, test.cmds...)
		intent := replay.Intent{
			Capture: f.storeCapture(ctx, cmds),
			Device:  path.NewDevice(f.device.Instance().Id.ID()),
		}
		done.Add(1)
		go checkIssues(ctx, intent, f.mgr, test.expected, done)
	}

	done.Wait()
}
