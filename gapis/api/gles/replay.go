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

	// Transform for all framebuffer reads.
	readFramebuffer := newReadFramebuffer(ctx)

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

		case framebufferRequest:
			deadCodeElimination.Request(req.after)
			// HACK: Also ensure we have framebuffer before the atom.
			// TODO: Remove this and handle swap-buffers better.
			deadCodeElimination.Request(req.after - 1)

			switch req.attachment {
			case api.FramebufferAttachment_Depth:
				readFramebuffer.Depth(req.after, rr.Result)
			case api.FramebufferAttachment_Stencil:
				return fmt.Errorf("Stencil buffer attachments are not currently supported")
			default:
				idx := uint32(req.attachment - api.FramebufferAttachment_Color0)
				readFramebuffer.Color(req.after, req.width, req.height, idx, rr.Result)
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

	transforms.Add(readFramebuffer)

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
	r := framebufferRequest{after: api.CmdID(after[0]), width: width, height: height, attachment: attachment}
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
	id := api.CmdNoID
	s := out.State()

	// Start by unbinding all the contexts from all the threads.
	for t, c := range GetState(s).Contexts {
		if c == nil {
			continue
		}
		cb := CommandBuilder{Thread: t}
		out.MutateAndWrite(ctx, id, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, 1))
	}

	// Now using a single thread, bind each context and delete all objects.
	cb := CommandBuilder{Thread: 0}
	for i, c := range GetState(s).EGLContexts {
		out.MutateAndWrite(ctx, id, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, i, 1))

		// Delete all Renderbuffers.
		renderbuffers := make([]RenderbufferId, 0, len(c.Objects.Shared.Renderbuffers)-3)
		for renderbufferId := range c.Objects.Shared.Renderbuffers {
			// Skip virtual renderbuffers: backbuffer_color(-1), backbuffer_depth(-2), backbuffer_stencil(-3).
			if renderbufferId < 0xf0000000 {
				renderbuffers = append(renderbuffers, renderbufferId)
			}
		}
		if len(renderbuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, renderbuffers)
			out.MutateAndWrite(ctx, id, cb.GlDeleteRenderbuffers(GLsizei(len(renderbuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Textures.
		textures := make([]TextureId, 0, len(c.Objects.Shared.Textures))
		for textureId := range c.Objects.Shared.Textures {
			textures = append(textures, textureId)
		}
		if len(textures) > 0 {
			tmp := s.AllocDataOrPanic(ctx, textures)
			out.MutateAndWrite(ctx, id, cb.GlDeleteTextures(GLsizei(len(textures)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Framebuffers.
		framebuffers := make([]FramebufferId, 0, len(c.Objects.Framebuffers))
		for framebufferId := range c.Objects.Framebuffers {
			framebuffers = append(framebuffers, framebufferId)
		}
		if len(framebuffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, framebuffers)
			out.MutateAndWrite(ctx, id, cb.GlDeleteFramebuffers(GLsizei(len(framebuffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Buffers.
		buffers := make([]BufferId, 0, len(c.Objects.Shared.Buffers))
		for bufferId := range c.Objects.Shared.Buffers {
			buffers = append(buffers, bufferId)
		}
		if len(buffers) > 0 {
			tmp := s.AllocDataOrPanic(ctx, buffers)
			out.MutateAndWrite(ctx, id, cb.GlDeleteBuffers(GLsizei(len(buffers)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all VertexArrays.
		vertexArrays := make([]VertexArrayId, 0, len(c.Objects.VertexArrays))
		for vertexArrayId := range c.Objects.VertexArrays {
			vertexArrays = append(vertexArrays, vertexArrayId)
		}
		if len(vertexArrays) > 0 {
			tmp := s.AllocDataOrPanic(ctx, vertexArrays)
			out.MutateAndWrite(ctx, id, cb.GlDeleteVertexArrays(GLsizei(len(vertexArrays)), tmp.Ptr()).AddRead(tmp.Data()))
		}

		// Delete all Shaders.
		for _, shaderId := range c.Objects.Shared.Shaders.KeysSorted() {
			out.MutateAndWrite(ctx, id, cb.GlDeleteShader(shaderId))
		}

		// Delete all Programs.
		for _, programId := range c.Objects.Shared.Programs.KeysSorted() {
			out.MutateAndWrite(ctx, id, cb.GlDeleteProgram(programId))
		}

		// Delete all Queries.
		queries := make([]QueryId, 0, len(c.Objects.Queries))
		for queryId := range c.Objects.Queries {
			queries = append(queries, queryId)
		}
		if len(queries) > 0 {
			tmp := s.AllocDataOrPanic(ctx, queries)
			out.MutateAndWrite(ctx, id, cb.GlDeleteQueries(GLsizei(len(queries)), tmp.Ptr()).AddRead(tmp.Data()))
		}
	}
	out.MutateAndWrite(ctx, id, cb.EglMakeCurrent(memory.Nullptr, memory.Nullptr, memory.Nullptr, memory.Nullptr, 1))
}
