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
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/core"
	"github.com/google/gapid/gapis/gfxapi/gles"
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
			gl_FragColor = texture(tex, texcoord);
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

func storeAtoms(ctx context.Context, a *atom.List) id.ID {
	id, err := database.Store(ctx, a)
	if err != nil {
		log.F(ctx, "Failed to store atom stream: %v", err)
	}
	return id
}

// storeCapture encodes and writes the atom list to the database, returning an
// identifier to the newly constructed and stored Capture.
func (f *Fixture) storeCapture(ctx context.Context, a *atom.List) *path.Capture {
	dev := f.device.Instance()
	h := &capture.Header{
		Device: dev,
		Abi:    dev.Configuration.ABIs[0],
	}
	out, err := capture.New(ctx, "test-capture", h, a.Atoms)
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
	ctx          context.Context
	mgr          *replay.Manager
	device       bind.Device
	memoryLayout *device.MemoryLayout
	s            *gfxapi.State
	nextID       uint32
}

// p returns a unique pointer. Meant to be used to generate
// pointers representing driver-side data, so the allocation
// itself is not relevant.
func (f *Fixture) p() memory.Pointer {
	base, err := f.s.Allocator.Alloc(8, 8)
	assert.With(f.ctx).ThatError(err).Succeeded()
	return memory.Pointer{Address: base, Pool: memory.ApplicationPool}
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
	s := gfxapi.NewStateWithEmptyAllocator(memoryLayout)

	return ctx, &Fixture{
		ctx:          ctx,
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
	return memory.Pointer{Address: addr, Pool: memory.ApplicationPool}
}

func checkImage(ctx context.Context, name string, got *image.Image2D, threshold float64) {
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
	issues, err := gles.API().(replay.QueryIssues).QueryIssues(ctx, intent, mgr)
	if assert.With(ctx).ThatError(err).Succeeded() {
		assert.With(ctx).ThatSlice(issues).DeepEquals(expected)
	}
}

func checkReport(ctx context.Context, intent replay.Intent, mgr *replay.Manager, atoms *atom.List, expected []string, done *sync.WaitGroup) {
	if done != nil {
		defer done.Done()
	}

	report, err := resolve.Report(ctx, intent.Capture, intent.Device)
	assert.With(ctx).ThatError(err).Succeeded()

	got := []string{}
	for _, e := range report.Items {
		if atom.ID(e.Command) != atom.NoID {
			got = append(got, fmt.Sprintf("%s@%d: %s: %v", e.Severity.String(), e.Command, atoms.Atoms[e.Command], report.Msg(e.Message).Text(nil)))
		} else {
			got = append(got, fmt.Sprintf("%s /%v", e.Severity.String(), report.Msg(e.Message).Text(nil)))
		}
	}
	assert.With(ctx).ThatSlice(got).Equals(expected)
}

func checkColorBuffer(ctx context.Context, intent replay.Intent, mgr *replay.Manager, w, h uint32, threshold float64, name string, after atom.ID, done *sync.WaitGroup) {
	ctx = log.Enter(ctx, "ColorBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	img, err := gles.API().(replay.QueryFramebufferAttachment).QueryFramebufferAttachment(
		ctx, intent, mgr, after, w, h, gfxapi.FramebufferAttachment_Color0, replay.WireframeMode_None, nil)
	if !assert.With(ctx).ThatError(err).Succeeded() {
		return
	}
	checkImage(ctx, name, img, threshold)
}

func checkDepthBuffer(ctx context.Context, intent replay.Intent, mgr *replay.Manager, w, h uint32, threshold float64, name string, after atom.ID, done *sync.WaitGroup) {
	ctx = log.Enter(ctx, "DepthBuffer")
	ctx = log.V{"name": name, "after": after}.Bind(ctx)
	if done != nil {
		defer done.Done()
	}
	ctx, _ = task.WithTimeout(ctx, replayTimeout)
	img, err := gles.API().(replay.QueryFramebufferAttachment).QueryFramebufferAttachment(
		ctx, intent, mgr, after, w, h, gfxapi.FramebufferAttachment_Depth, replay.WireframeMode_None, nil)
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

func initContext(f *Fixture, width, height int, preserveBuffersOnSwap bool) (atoms *atom.List, eglContext memory.Pointer, eglSurface memory.Pointer) {
	eglContext = f.p()
	eglSurface = f.p()

	eglConfig := f.p()

	eglShareContext := memory.Nullptr
	// TODO: We don't observe attribute lists properly. We should.
	atoms = atom.NewList(
		gles.NewEglGetDisplay(gles.EGLNativeDisplayType(0), eglDisplay),
		gles.NewEglInitialize(eglDisplay, memory.Nullptr, memory.Nullptr, gles.EGLBoolean(1)),
		gles.NewEglCreateContext(eglDisplay, eglConfig, eglShareContext, f.p(), eglContext),
		makeCurrent(eglSurface, eglContext, width, height, preserveBuffersOnSwap),
	)
	return atoms, eglContext, eglSurface
}

func makeCurrent(eglSurface, eglContext memory.Pointer, width, height int, preserveBuffersOnSwap bool) atom.Atom {
	eglTrue := gles.EGLBoolean(1)
	return atom.WithExtras(
		gles.NewEglMakeCurrent(eglDisplay, eglSurface, eglSurface, eglContext, eglTrue),
		gles.NewStaticContextState(),
		gles.NewDynamicContextState(width, height, preserveBuffersOnSwap),
	)
}

func TestClear(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))

	atoms, red, green, blue, black := samples.ClearBackbuffer(ctx)

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
type traceGenerator func(f *Fixture) (*path.Capture, traceVerifier)

// mergeCaptures creates a capture from the atoms of several existing captures, by interleaving them
// arbitrarily, with switchThread atoms inserted whenever switching between captures.
func mergeCaptures(f *Fixture, captures ...*path.Capture) *path.Capture {
	lists := [][]atom.Atom{}
	threads := []core.ThreadID{}
	remainingAtoms := 0

	for i, path := range captures {
		c, err := capture.ResolveFromPath(f.ctx, path)
		assert.With(f.ctx).ThatError(err).Succeeded()
		lists = append(lists, c.Atoms)
		remainingAtoms += len(c.Atoms)
		threads = append(threads, core.ThreadID(0x10000+i))
	}

	merged := []atom.Atom{}
	threadIndex, prevThreadIndex := 0, -1
	cmdsUntilSwitchThread, modFourCounter := 4, 3
	for remainingAtoms > 0 {
		if cmdsUntilSwitchThread > 0 && len(lists[threadIndex]) > 0 {
			if threadIndex != prevThreadIndex {
				prevThreadIndex = threadIndex
				merged = append(merged, core.NewSwitchThread(threads[threadIndex]))
			}
			merged = append(merged, lists[threadIndex][0])
			lists[threadIndex] = lists[threadIndex][1:]
			remainingAtoms--
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
	return f.storeCapture(f.ctx, atom.NewList(merged...))
}

func generateDrawTexturedSquareCapture(f *Fixture) (*path.Capture, traceVerifier) {
	ctx := f.ctx
	atoms, _, square := samples.DrawTexturedSquare(ctx, false)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", square, nil)
	}

	return f.storeCapture(ctx, atoms), verifyTrace
}

func generateDrawTexturedSquareCaptureWithSharedContext(f *Fixture) (*path.Capture, traceVerifier) {
	ctx := f.ctx
	atoms, _, square := samples.DrawTexturedSquare(ctx, true)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkColorBuffer(ctx, intent, mgr, 128, 128, 0.01, "textured-square", square, nil)
	}

	return f.storeCapture(ctx, atoms), verifyTrace
}

func generateCaptureWithIssues(f *Fixture) (*path.Capture, traceVerifier) {
	ctx := f.ctx
	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	missingProg := gles.ProgramId(f.newID())
	atoms, _, eglSurface := initContext(f, 128, 128, false)
	texLoc := gles.UniformLocation(0)

	s := gfxapi.NewStateWithEmptyAllocator(f.memoryLayout)

	textureNames := []gles.TextureId{1}
	textureNamesR := atom.Must(atom.AllocData(ctx, f.s, textureNames))

	squareIndicesR := atom.Must(atom.AllocData(ctx, f.s, squareIndices))
	squareVerticesR := atom.Must(atom.AllocData(ctx, f.s, squareVertices))

	someString := atom.Must(atom.AllocData(ctx, f.s, "hello world"))

	atoms.Add(gles.BuildProgram(ctx, s, vs, fs, prog, textureVSSource, textureFSSource)...)
	atoms.Add(
		gles.NewGlEnable(gles.GLenum_GL_DEPTH_TEST), // Required for depth-writing
		gles.NewGlClearColor(0.0, 1.0, 0.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_2X_BIT_ATI),

		atom.WithExtras(
			gles.NewGlLinkProgram(prog),
			&gles.ProgramInfo{
				LinkStatus: gles.GLboolean_GL_TRUE,
				ActiveUniforms: gles.UniformIndexːActiveUniformᵐ{
					0: {
						Type:      gles.GLenum_GL_SAMPLER_2D,
						Name:      "tex",
						ArraySize: 1,
						Location:  texLoc,
					},
				},
			}),
		gles.NewGlUseProgram(missingProg),
		gles.NewGlLabelObjectEXT(gles.GLenum_GL_TEXTURE, 123, gles.GLsizei(someString.Range().Size), someString.Ptr()).AddRead(someString.Data()),
		gles.NewGlGetError(0),
		gles.NewGlUseProgram(prog),
		gles.NewGlGenTextures(1, textureNamesR.Ptr()).AddWrite(textureNamesR.Data()),
		gles.NewGlGetUniformLocation(prog, "tex", texLoc),
		gles.NewGlActiveTexture(gles.GLenum_GL_TEXTURE0),
		gles.NewGlBindTexture(gles.GLenum_GL_TEXTURE_2D, textureNames[0]),
		gles.NewGlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MIN_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		gles.NewGlTexParameteri(gles.GLenum_GL_TEXTURE_2D, gles.GLenum_GL_TEXTURE_MAG_FILTER, gles.GLint(gles.GLenum_GL_NEAREST)),
		gles.NewGlUniform1i(texLoc, 0),
		gles.NewGlGetAttribLocation(prog, "position", gles.GLint(pos)),
		gles.NewGlEnableVertexAttribArray(pos),
		gles.NewGlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, squareVerticesR.Ptr()),
		gles.NewGlDrawElements(gles.GLenum_GL_TRIANGLES, 6, gles.GLenum_GL_UNSIGNED_SHORT, squareIndicesR.Ptr()).
			AddRead(squareIndicesR.Data()).
			AddRead(squareVerticesR.Data()),
		gles.NewEglSwapBuffers(eglDisplay, eglSurface, gles.EGLBoolean(1)),
	)

	verifyTrace := func(ctx context.Context, cap *path.Capture, mgr *replay.Manager, dev bind.Device) {
		intent := replay.Intent{
			Capture: cap,
			Device:  path.NewDevice(dev.Instance().Id.ID()),
		}
		defer checkReplay(ctx, intent, 1)() // expect a single replay batch.

		checkReport(ctx, intent, mgr, atoms, []string{
			"ErrorLevel@15: glClear(mask: GLbitfield(16385)): <ERR_INVALID_VALUE [value: value, variable: variable]>",
			"ErrorLevel@17: glUseProgram(program: 4): <ERR_INVALID_VALUE [value: value, variable: variable]>",
			"ErrorLevel@18: glLabelObjectEXT(type: GL_TEXTURE, object: 123, length: 12, label: {{} 4208 0}): <ERR_INVALID_OPERATION [operation: operation]>",
		}, nil)
	}

	return f.storeCapture(ctx, atoms), verifyTrace
}

func generateDrawTriangleCapture(f *Fixture) (*path.Capture, traceVerifier) {
	return generateDrawTriangleCaptureEx(f, 0.0, 1.0, 0.0, 1.0, 0.0, 0.0)
}

// generateDrawTriangleCaptureEx generates a capture with several frames containing
// a rotating triangle of color RGB(fr, fg, fb) on a RGB(br, bg, bb) background.
func generateDrawTriangleCaptureEx(f *Fixture, br, bg, bb, fr, fg, fb gles.GLfloat) (*path.Capture, traceVerifier) {
	ctx := f.ctx
	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	angleLoc := gles.UniformLocation(0)
	atoms, _, eglSurface := initContext(f, 64, 64, false)
	atoms.Add(gles.NewGlEnable(gles.GLenum_GL_DEPTH_TEST)) // Required for depth-writing

	clear := atoms.Add(
		gles.NewGlClearColor(br, bg, bb, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
	)
	atoms.Add(gles.BuildProgram(ctx, f.s, vs, fs, prog, simpleVSSource, simpleFSSource(fr, fg, fb))...)

	triangleVerticesR := atom.Must(atom.AllocData(ctx, f.s, triangleVertices))

	triangle := atoms.Add(
		atom.WithExtras(
			gles.NewGlLinkProgram(prog),
			&gles.ProgramInfo{
				LinkStatus: gles.GLboolean_GL_TRUE,
				ActiveUniforms: gles.UniformIndexːActiveUniformᵐ{
					0: {
						Type:      gles.GLenum_GL_FLOAT,
						Name:      "angle",
						ArraySize: 1,
						Location:  angleLoc,
					},
				},
			}),
		gles.NewGlUseProgram(prog),
		gles.NewGlGetUniformLocation(prog, "angle", angleLoc),
		gles.NewGlUniform1f(angleLoc, gles.GLfloat(0)),
		gles.NewGlGetAttribLocation(prog, "position", gles.GLint(pos)),
		gles.NewGlEnableVertexAttribArray(pos),
		gles.NewGlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
		gles.NewGlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)

	angle := 0.0
	for i := 0; i < 30; i++ {
		angle += math.Pi / 30.0
		atoms.Add(
			gles.NewEglSwapBuffers(eglDisplay, eglSurface, gles.EGLBoolean(1)),
			gles.NewGlUniform1f(angleLoc, gles.GLfloat(angle)),
			gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT|gles.GLbitfield_GL_DEPTH_BUFFER_BIT),
			gles.NewGlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
		)
	}
	rotatedTriangle := atoms.Add()

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

	return f.storeCapture(ctx, atoms), verifyTrace
}

func testTrace(t *testing.T, name string, tg traceGenerator) {
	ctx, f := newFixture(log.Testing(t))
	capture, verifyTrace := tg(f)
	maybeExportCapture(ctx, name, capture)
	verifyTrace(ctx, capture, f.mgr, f.device)
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
	ctx, f := newFixture(log.Testing(t))

	t1, _ := generateDrawTriangleCaptureEx(f, 1.0, 0.0, 0.0, 1.0, 1.0, 0.0)
	t2, _ := generateDrawTriangleCaptureEx(f, 0.0, 1.0, 0.0, 0.0, 0.0, 1.0)
	t3, _ := generateDrawTriangleCaptureEx(f, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0)
	capture := mergeCaptures(f, t1, t2, t3)
	maybeExportCapture(ctx, "multi_context", capture)

	contexts, err := resolve.Contexts(ctx, capture.Contexts())
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).That(len(contexts)).Equals(3)
}

func TestTraceWithIssues(t *testing.T) {
	testTrace(t, "with_issues", generateCaptureWithIssues)
}

func TestExportAndImportCapture(t *testing.T) {
	ctx, f := newFixture(log.Testing(t))
	c, verifyTrace := generateDrawTriangleCapture(f)

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

	triangleVerticesR := atom.Must(atom.AllocData(ctx, f.s, triangleVertices))

	vs, fs, prog, pos := gles.ShaderId(f.newID()), gles.ShaderId(f.newID()), gles.ProgramId(f.newID()), gles.AttributeLocation(0)
	atoms, eglContext, eglSurface := initContext(f, 8, 8, false) // start with a small backbuffer
	atoms.Add(gles.BuildProgram(ctx, f.s, vs, fs, prog, simpleVSSource, simpleFSSource(1.0, 0.0, 0.0))...)
	atoms.Add(
		gles.NewGlLinkProgram(prog),
		gles.NewGlUseProgram(prog),
		gles.NewGlGetAttribLocation(prog, "position", gles.GLint(pos)),
		gles.NewGlEnableVertexAttribArray(pos),
		gles.NewGlVertexAttribPointer(pos, 3, gles.GLenum_GL_FLOAT, gles.GLboolean(0), 0, triangleVerticesR.Ptr()),
	)
	triangle := atoms.Add(
		makeCurrent(eglSurface, eglContext, 64, 64, false),
		gles.NewGlClearColor(0.0, 0.0, 1.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
		gles.NewGlDrawArrays(gles.GLenum_GL_TRIANGLES, 0, 3).AddRead(triangleVerticesR.Data()),
	)
	capture := f.storeCapture(ctx, atoms)
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

	atoms, _, _ := initContext(f, 64, 64, true)
	clear := atoms.Add(
		gles.NewGlClearColor(0.0, 0.0, 1.0, 1.0),
		gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
	)
	swapA := atoms.Add(gles.NewEglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))
	swapB := atoms.Add(gles.NewEglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))
	swapC := atoms.Add(gles.NewEglSwapBuffers(memory.Nullptr, memory.Nullptr, 1))

	intent := replay.Intent{
		Capture: f.storeCapture(ctx, atoms),
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
		atoms    []atom.Atom
		expected []replay.Issue
	}{
		{
			"glClear - no errors",
			[]atom.Atom{
				gles.NewGlClearColor(0.0, 0.0, 1.0, 1.0),
				gles.NewGlClear(gles.GLbitfield_GL_COLOR_BUFFER_BIT),
			},
			[]replay.Issue{},
		},
	} {
		atoms, _, _ := initContext(f, 64, 64, true)
		atoms.Add(test.atoms...)
		intent := replay.Intent{
			Capture: f.storeCapture(ctx, atoms),
			Device:  path.NewDevice(f.device.Instance().Id.ID()),
		}
		done.Add(1)
		go checkIssues(ctx, intent, f.mgr, test.expected, done)
	}

	done.Wait()
}
