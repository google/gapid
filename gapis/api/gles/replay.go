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
	"context"
	"fmt"
	"strings"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(API{})
	_ = replay.QueryFramebufferAttachment(API{})
	_ = replay.Support(API{})
	_ = replay.Profiler(API{})
)

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct{}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	drawMode                  service.DrawMode
	wireframeOverlayID        api.CmdID     // used when drawMode == DrawMode_WIREFRAME_OVERLAY
	wireframeFramebufferID    FramebufferId // used when drawMode == DrawMode_WIREFRAME_ALL
	disableReplayOptimization bool
}

// uniqueConfig returns a replay.Config that is guaranteed to be unique.
// Any requests made with a Config returned from uniqueConfig will not be
// batched with any other request.
func uniqueConfig() replay.Config {
	return &struct{}{}
}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct{}

// framebufferRequest requests a postback of a framebuffer's attachment.
type framebufferRequest struct {
	after            api.CmdID
	width, height    uint32
	fb               FramebufferId
	attachment       api.FramebufferAttachment
	wireframeOverlay bool
}

type profileRequest struct {
}

// GetReplayPriority returns a uint32 representing the preference for
// replaying this trace on the given device.
// A lower number represents a higher priority, and zero represents
// an inability for the trace to be replayed on the given device.
func (a API) GetReplayPriority(ctx context.Context, i *device.Instance, h *capture.Header) uint32 {
	v, err := ParseVersion(i.GetConfiguration().GetDrivers().GetOpengl().GetVersion())
	if err != nil {
		return 0 // Can't figure out what we're dealing with.
	}

	switch {
	case v.AtLeastES(3, 0):
		traceDev := h.GetDevice()
		devHardware, traceHardware := i.GetConfiguration().GetHardware(), traceDev.GetConfiguration().GetHardware()
		devOS, traceOS := i.GetConfiguration().GetOS(), traceDev.GetConfiguration().GetOS()

		if s1, s2 := i.GetSerial(), traceDev.GetSerial(); s1 != "" && s1 == s2 {
			return 1 // Serial matches that of device the trace was captured on.
		}
		if h1, h2 := devHardware.GetName(), traceHardware.GetName(); h1 != "" && h1 == h2 {
			return 2 // Same hardware device name.
		}

		for _, abi := range i.GetConfiguration().GetABIs() {
			if abi.SameAs(h.GetABI()) {
				if b1, b2 := devOS.GetBuild(), traceOS.GetBuild(); b1 != "" && b1 == b2 {
					return 3 // Same OS build.
				}
				switch devOS.CompareVersions(traceOS) {
				case device.CompleteMatch:
					return 4 // Same OS version
				case device.MajorAndMinorMatch:
					return 5 // Same major.minor OS version.
				case device.MajorMatch:
					return 6 // Same major OS version.
				default:
					return 7 // Different major version.
				}
			}
		}
		return 0 // Device does not support capture ABI.
	case v.IsES:
		return 0 // Can't replay on this version of an ES device.
	default:
		return 8 // Desktop GL can be used with heavy use of compat.
	}
}

func (a API) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	dependentPayload string,
	rrs []replay.RequestAndResult,
	device *device.Instance,
	capture *capture.GraphicsCapture,
	out transform.Writer) error {
	if dependentPayload != "" {
		return log.Errf(ctx, nil, "GLES does not support dependent payloads")
	}
	if a.GetReplayPriority(ctx, device, capture.Header) == 0 {
		return log.Errf(ctx, nil, "Cannot replay GLES commands on device '%v'", device.Name)
	}

	ctx = PutUnusedIDMap(ctx)

	cmds := capture.Commands

	// Gathers and reports any issues found.
	var issues *findIssues

	// Prepare data for dead-code-elimination.
	dependencyGraph, err := dependencygraph.GetDependencyGraph(ctx, intent.Device)
	if err != nil {
		return err
	}

	// Skip unnecessary commands.
	deadCodeElimination := dependencygraph.NewDeadCodeElimination(ctx, dependencyGraph)
	deadCodeElimination.KeepAllAlive = config.DisableDeadCodeElimination

	var rf *readFramebuffer // Transform for all framebuffer reads.
	var rt *readTexture     // Transform for all texture reads.

	var wire transform.Transformer

	transforms := transform.Transforms{deadCodeElimination}

	onCompatError := func(ctx context.Context, id api.CmdID, cmd api.Cmd, err error) {
		ctx = log.Enter(ctx, "Compat")
		log.E(ctx, "%v: %v - %v", id, cmd, err)
	}

	var profile *replay.EndOfReplay

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case issuesRequest:
			deadCodeElimination.KeepAllAlive = true
			if issues == nil {
				issues = newFindIssues(ctx, capture, device)
			}
			issues.AddResult(rr.Result)
			onCompatError = func(ctx context.Context, id api.CmdID, cmd api.Cmd, err error) {
				issues.onIssue(cmd, id, service.Severity_ErrorLevel, err)
			}

		case textureRequest:
			if rt == nil {
				rt = newReadTexture(ctx, device)
			}
			after := api.CmdID(req.data.After)
			deadCodeElimination.Request(after)
			rt.add(ctx, req.data, rr.Result)

		case framebufferRequest:
			if rf == nil {
				rf = newReadFramebuffer(ctx, device)
			}
			deadCodeElimination.Request(req.after)

			thread := cmds[req.after].Thread()
			switch req.attachment {
			case api.FramebufferAttachment_Depth:
				rf.depth(req.after, thread, req.fb, rr.Result)
			case api.FramebufferAttachment_Stencil:
				return fmt.Errorf("Stencil buffer attachments are not currently supported")
			default:
				idx := uint32(req.attachment - api.FramebufferAttachment_Color0)
				rf.color(req.after, thread, req.width, req.height, req.fb, idx, rr.Result)
			}

			cfg := cfg.(drawConfig)
			if cfg.disableReplayOptimization {
				deadCodeElimination.KeepAllAlive = true
			}
			switch cfg.drawMode {
			case service.DrawMode_WIREFRAME_ALL:
				wire = wireframe(ctx, cfg.wireframeFramebufferID)
			case service.DrawMode_WIREFRAME_OVERLAY:
				wire = wireframeOverlay(ctx, req.after)
			case service.DrawMode_OVERDRAW:
				return fmt.Errorf("Overdraw is not currently supported for GLES")
			}

		case profileRequest:
			if profile == nil {
				profile = &replay.EndOfReplay{}
			}
			profile.AddResult(rr.Result)
		}
	}

	if wire != nil {
		transforms.Add(wire)
	}

	if issues != nil {
		transforms.Add(issues) // Issue reporting required.
	}

	// Render pattern for undefined framebuffers.
	// Needs to be after 'issues' which uses absence of draw calls to find undefined framebuffers.
	transforms.Add(undefinedFramebuffer(ctx, device))

	if rt != nil {
		transforms.Add(rt)
	}
	if rf != nil {
		transforms.Add(rf)
	}

	if profile != nil {
		// Don't use DCE.
		transforms = transform.Transforms{profile}
	}

	// Device-dependent transforms.
	compatTransform, err := compat(ctx, device, onCompatError)
	if err == nil {
		transforms.Add(compatTransform)
	} else {
		log.E(ctx, "Error creating compatability transform: %v", err)
	}

	// Cleanup
	transforms.Add(&destroyResourcesAtEOS{})

	if config.DebugReplay {
		log.I(ctx, "Replaying %d commands using transform chain:", len(cmds))
		for i, t := range transforms {
			log.I(ctx, "(%d) %#v", i, t)
		}
	}

	if config.LogTransformsToFile {
		newTransforms := transform.Transforms{}
		for i, t := range transforms {
			name := fmt.Sprintf("%T", t)
			if n, ok := t.(interface {
				Name() string
			}); ok {
				name = n.Name()
			} else if dot := strings.LastIndex(name, "."); dot != -1 {
				name = name[dot+1:]
			}
			newTransforms.Add(t, transform.NewFileLog(ctx, fmt.Sprintf("%v.%v_%v", capture.Name, i, name)))
		}
		transforms = newTransforms
	}
	if config.LogTransformsToCapture {
		transforms.Add(transform.NewCaptureLog(ctx, capture, "replay_log.gfxtrace"))
	}

	if profile == nil {
		cmds = []api.Cmd{} // DeadCommandRemoval generates commands.
	}
	return transforms.TransformAll(ctx, cmds, 0, out)
}

func (a API) QueryIssues(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	loopCount int32,
	displayToSurface bool,
	hints *service.UsageHints) ([]replay.Issue, error) {

	if loopCount != 1 {
		return nil, log.Errf(ctx, nil, "GLES does not support frame looping")
	}

	c, r := issuesConfig{}, issuesRequest{}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, true)
	if err != nil {
		return nil, err
	}
	return res.([]replay.Issue), nil
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	drawMode service.DrawMode,
	disableReplayOptimization bool,
	displayToSurface bool,
	hints *service.UsageHints) (*image.Data, error) {

	if len(after) > 1 {
		return nil, log.Errf(ctx, nil, "GLES does not support subcommands")
	}

	c := drawConfig{drawMode: drawMode, disableReplayOptimization: disableReplayOptimization}
	switch drawMode {
	case service.DrawMode_WIREFRAME_OVERLAY:
		c.wireframeOverlayID = api.CmdID(after[0])

	case service.DrawMode_WIREFRAME_ALL:
		c.wireframeFramebufferID = FramebufferId(framebufferIndex)
	}

	r := framebufferRequest{
		after:      api.CmdID(after[0]),
		width:      width,
		height:     height,
		fb:         FramebufferId(framebufferIndex),
		attachment: attachment,
	}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, true)
	if err != nil {
		return nil, err
	}
	return res.(*image.Data), nil
}

func (a API) Profile(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	hints *service.UsageHints,
	traceOptions *service.TraceOptions) (*service.ProfilingData, error) {

	c := uniqueConfig()
	r := profileRequest{}

	_, err := mgr.Replay(ctx, intent, c, r, a, hints, true)
	return nil, err
}

// destroyResourcesAtEOS is a transform that destroys all textures,
// framebuffers, buffers, shaders, programs and vertex-arrays that were not
// destroyed by EOS.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *destroyResourcesAtEOS) Flush(ctx context.Context, out transform.Writer) error {
	s := out.State()
	cmds := []api.Cmd{}

	// Start by unbinding all the contexts from all the threads.
	for t, c := range GetState(s).Contexts().All() {
		if c.IsNil() {
			continue
		}
		cb := CommandBuilder{Thread: t, Arena: s.Arena}
		cmds = append(cmds, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, 1))
	}

	// Now using a single thread, bind each context and delete all objects.
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}
	for i, c := range GetState(s).EGLContexts().All() {
		if !c.Other().Initialized() {
			// This context was never bound. Skip it.
			continue
		}

		cmds = append(cmds, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, i, 1))

		// Delete all Renderbuffers.
		renderbuffers := make([]RenderbufferId, 0, c.Objects().Renderbuffers().Len())
		for renderbufferID := range c.Objects().Renderbuffers().All() {
			// Skip virtual renderbuffers: backbuffer_color(-1), backbuffer_depth(-2), backbuffer_stencil(-3).
			if renderbufferID < 0xf0000000 {
				renderbuffers = append(renderbuffers, renderbufferID)
			}
		}
		if len(renderbuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, renderbuffers)
			cmds = append(cmds, cb.GlDeleteRenderbuffers(GLsizei(len(renderbuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Textures.
		textures := make([]TextureId, 0, c.Objects().Textures().Len())
		for textureID := range c.Objects().Textures().All() {
			textures = append(textures, textureID)
		}
		if len(textures) > 0 {
			tmp := s.AllocDataOrPanic(ctx, textures)
			cmds = append(cmds, cb.GlDeleteTextures(GLsizei(len(textures)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Framebuffers.
		framebuffers := make([]FramebufferId, 0, c.Objects().Framebuffers().Len())
		for framebufferID := range c.Objects().Framebuffers().All() {
			framebuffers = append(framebuffers, framebufferID)
		}
		if len(framebuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, framebuffers)
			cmds = append(cmds, cb.GlDeleteFramebuffers(GLsizei(len(framebuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Buffers.
		buffers := make([]BufferId, 0, c.Objects().Buffers().Len())
		for bufferID := range c.Objects().Buffers().All() {
			buffers = append(buffers, bufferID)
		}
		if len(buffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, buffers)
			cmds = append(cmds, cb.GlDeleteBuffers(GLsizei(len(buffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all VertexArrays.
		vertexArrays := make([]VertexArrayId, 0, c.Objects().VertexArrays().Len())
		for vertexArrayID := range c.Objects().VertexArrays().All() {
			vertexArrays = append(vertexArrays, vertexArrayID)
		}
		if len(vertexArrays) > 0 {
			tmp := s.AllocDataOrPanic(ctx, vertexArrays)
			cmds = append(cmds, cb.GlDeleteVertexArrays(GLsizei(len(vertexArrays)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Shaders.
		for _, shaderID := range c.Objects().Shaders().Keys() {
			cmds = append(cmds, cb.GlDeleteShader(shaderID))
		}

		// Delete all Programs.
		for _, programID := range c.Objects().Programs().Keys() {
			cmds = append(cmds, cb.GlDeleteProgram(programID))
		}

		// Delete all Queries.
		queries := make([]QueryId, 0, c.Objects().Queries().Len())
		for queryID := range c.Objects().Queries().All() {
			queries = append(queries, queryID)
		}
		if len(queries) > 0 {
			tmp := s.AllocDataOrPanic(ctx, queries)
			cmds = append(cmds, cb.GlDeleteQueries(GLsizei(len(queries)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Flush all buffered commands before proceeding to the next context.
		// Contexts can share objects - e.g. several contexts can contain the same buffer.
		// Mutating the delete command ensures the object is removed from all maps,
		// and that we will not try to remove it again when iterating over the second context.
		cmds = append(cmds, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, 1))
		err := api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
			return out.MutateAndWrite(ctx, api.CmdNoID, cmd)
		})
		if err != nil {
			return err
		}
		cmds = []api.Cmd{}
	}
	return nil
}

func (t *destroyResourcesAtEOS) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *destroyResourcesAtEOS) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *destroyResourcesAtEOS) BuffersCommands() bool                              { return false }
