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
)

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct{}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	wireframeMode      replay.WireframeMode
	wireframeOverlayID api.CmdID // used when wireframeMode == WireframeMode_Overlay
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

// GetReplayPriority returns a uint32 representing the preference for
// replaying this trace on the given device.
// A lower number represents a higher priority, and Zero represents
// an inability for the trace to be replayed on the given device.
func (a API) GetReplayPriority(ctx context.Context, i *device.Instance, l *device.MemoryLayout) uint32 {
	if i.GetConfiguration().GetOS().GetKind() != device.Android {
		return 1
	}

	return 2
}

func (a API) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	rrs []replay.RequestAndResult,
	device *device.Instance,
	capture *capture.Capture,
	out transform.Writer) error {

	if a.GetReplayPriority(ctx, device, capture.Header.Abi.MemoryLayout) == 0 {
		return log.Errf(ctx, nil, "Cannot replay GLES commands on device '%v'", device.Name)
	}

	ctx = PutUnusedIDMap(ctx)

	cmds := capture.Commands

	transforms := transform.Transforms{}

	// Gathers and reports any issues found.
	var issues *findIssues

	// Prepare data for dead-code-elimination.
	dependencyGraph, err := dependencygraph.GetDependencyGraph(ctx)
	if err != nil {
		return err
	}

	// Skip unnecessary commands.
	deadCodeElimination := transform.NewDeadCodeElimination(ctx, dependencyGraph)

	var rf *readFramebuffer // Transform for all framebuffer reads.
	var rt *readTexture     // Transform for all texture reads.

	optimize := true
	wire := false

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case issuesRequest:
			optimize = false
			if issues == nil {
				issues = newFindIssues(ctx, capture, device)
			}
			issues.reportTo(rr.Result)

		case textureRequest:
			if rt == nil {
				rt = &readTexture{}
			}
			after := api.CmdID(req.data.After)
			deadCodeElimination.Request(after)
			rt.add(ctx, req.data, rr.Result)

		case framebufferRequest:
			if rf == nil {
				rf = &readFramebuffer{}
			}
			deadCodeElimination.Request(req.after)
			// HACK: Also ensure we have framebuffer before the command.
			// TODO: Remove this and handle swap-buffers better.
			deadCodeElimination.Request(req.after - 1)

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
			switch cfg.wireframeMode {
			case replay.WireframeMode_All:
				wire = true
			case replay.WireframeMode_Overlay:
				transforms.Add(wireframeOverlay(ctx, req.after))
			}
		}
	}

	if optimize && !config.DisableDeadCodeElimination {
		cmds = []api.Cmd{} // DeadCommandRemoval generates commands.
		transforms.Prepend(deadCodeElimination)
	}

	if wire {
		transforms.Add(wireframe(ctx))
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

	// Device-dependent transforms.
	if c, err := compat(ctx, device); err == nil {
		transforms.Add(c)
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

	transforms.Transform(ctx, cmds, out)
	return nil
}

func (a API) QueryIssues(
	ctx context.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	hints *service.UsageHints) ([]replay.Issue, error) {

	c, r := issuesConfig{}, issuesRequest{}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
	if err != nil {
		return nil, err
	}
	return res.([]replay.Issue), nil
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	wireframeMode replay.WireframeMode,
	hints *service.UsageHints) (*image.Data, error) {

	if len(after) > 1 {
		return nil, log.Errf(ctx, nil, "GLES does not support subcommands")
	}

	c := drawConfig{wireframeMode: wireframeMode}
	if wireframeMode == replay.WireframeMode_Overlay {
		c.wireframeOverlayID = api.CmdID(after[0])
	}
	r := framebufferRequest{
		after:      api.CmdID(after[0]),
		width:      width,
		height:     height,
		fb:         FramebufferId(framebufferIndex),
		attachment: attachment,
	}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
	if err != nil {
		return nil, err
	}
	return res.(*image.Data), nil
}

// destroyResourcesAtEOS is a transform that destroys all textures,
// framebuffers, buffers, shaders, programs and vertex-arrays that were not
// destroyed by EOS.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *destroyResourcesAtEOS) Flush(ctx context.Context, out transform.Writer) {
	s := out.State()
	cmds := []api.Cmd{}

	// Start by unbinding all the contexts from all the threads.
	for t, c := range GetState(s).Contexts.Range() {
		if c == nil {
			continue
		}
		cb := CommandBuilder{Thread: t}
		cmds = append(cmds, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, 1))
	}

	// Now using a single thread, bind each context and delete all objects.
	cb := CommandBuilder{Thread: 0}
	for i, c := range GetState(s).EGLContexts.Range() {
		cmds = append(cmds, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, i, 1))

		// Delete all Renderbuffers.
		renderbuffers := make([]RenderbufferId, 0, c.Objects.Renderbuffers.Len())
		for renderbufferId := range c.Objects.Renderbuffers.Range() {
			// Skip virtual renderbuffers: backbuffer_color(-1), backbuffer_depth(-2), backbuffer_stencil(-3).
			if renderbufferId < 0xf0000000 {
				renderbuffers = append(renderbuffers, renderbufferId)
			}
		}
		if len(renderbuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, renderbuffers)
			cmds = append(cmds, cb.GlDeleteRenderbuffers(GLsizei(len(renderbuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Textures.
		textures := make([]TextureId, 0, c.Objects.Textures.Len())
		for textureId := range c.Objects.Textures.Range() {
			textures = append(textures, textureId)
		}
		if len(textures) > 0 {
			tmp := s.AllocDataOrPanic(ctx, textures)
			cmds = append(cmds, cb.GlDeleteTextures(GLsizei(len(textures)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Framebuffers.
		framebuffers := make([]FramebufferId, 0, c.Objects.Framebuffers.Len())
		for framebufferId := range c.Objects.Framebuffers.Range() {
			framebuffers = append(framebuffers, framebufferId)
		}
		if len(framebuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, framebuffers)
			cmds = append(cmds, cb.GlDeleteFramebuffers(GLsizei(len(framebuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Buffers.
		buffers := make([]BufferId, 0, c.Objects.Buffers.Len())
		for bufferId := range c.Objects.Buffers.Range() {
			buffers = append(buffers, bufferId)
		}
		if len(buffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, buffers)
			cmds = append(cmds, cb.GlDeleteBuffers(GLsizei(len(buffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all VertexArrays.
		vertexArrays := make([]VertexArrayId, 0, c.Objects.VertexArrays.Len())
		for vertexArrayId := range c.Objects.VertexArrays.Range() {
			vertexArrays = append(vertexArrays, vertexArrayId)
		}
		if len(vertexArrays) > 0 {
			tmp := s.AllocDataOrPanic(ctx, vertexArrays)
			cmds = append(cmds, cb.GlDeleteVertexArrays(GLsizei(len(vertexArrays)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Shaders.
		for _, shaderId := range c.Objects.Shaders.KeysSorted() {
			cmds = append(cmds, cb.GlDeleteShader(shaderId))
		}

		// Delete all Programs.
		for _, programId := range c.Objects.Programs.KeysSorted() {
			cmds = append(cmds, cb.GlDeleteProgram(programId))
		}

		// Delete all Queries.
		queries := make([]QueryId, 0, c.Objects.Queries.Len())
		for queryId := range c.Objects.Queries.Range() {
			queries = append(queries, queryId)
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
		api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
			out.MutateAndWrite(ctx, api.CmdNoID, cmd)
			return nil
		})
		cmds = []api.Cmd{}
	}
}
