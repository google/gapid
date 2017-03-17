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
	"fmt"
	"strings"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/replay"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(api{})
	_ = replay.QueryFramebufferAttachment(api{})
	_ = replay.Support(api{})
)

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct{}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	wireframeMode      replay.WireframeMode
	wireframeOverlayID atom.ID // used when wireframeMode == WireframeMode_Overlay
}

// uniqueConfig returns a replay.Config that is guaranteed to be unique.
// Any requests made with a Config returned from uniqueConfig will not be
// batched with any other request.
func uniqueConfig() replay.Config {
	return &struct{}{}
}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out chan<- replay.Issue
}

// framebufferRequest requests a postback of a framebuffer's attachment.
type framebufferRequest struct {
	after            atom.ID
	width, height    uint32
	attachment       gfxapi.FramebufferAttachment
	out              chan imgRes
	wireframeOverlay bool
}

// imgRes holds the result of an image query.
type imgRes struct {
	img *image.Image2D // The image data.
	err error          // The error that occurred generating the image.
}

// CanReplayOnLocalAndroidDevice returns true if the API can be replayed on a
// locally connected Android device.
func (a api) CanReplayOnLocalAndroidDevice(log.Context) bool { return false }

// CanReplayOn returns true if the API can be replayed on the specified device.
// GLES can currently cannot replay on Android devices.
func (a api) CanReplayOn(ctx log.Context, i *device.Instance) bool {
	return i.GetConfiguration().GetOS().GetKind() != device.Android
}

func (a api) Replay(
	ctx log.Context,
	intent replay.Intent,
	cfg replay.Config,
	requests []replay.Request,
	device *device.Instance,
	capture *capture.Capture,
	out transform.Writer) error {

	ctx = PutUnusedIDMap(ctx)

	atoms, err := capture.Atoms(ctx)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to load atom stream")
	}

	transforms := transform.Transforms{}

	// Gathers and reports any issues found.
	var issues *findIssues

	// Prepare data for dead-code-elimination.
	dependencyGraph, err := GetDependencyGraph(ctx)
	if err != nil {
		return err
	}

	// Skip unnecessary atoms.
	deadCodeElimination := newDeadCodeElimination(ctx, dependencyGraph)

	// Transform for all framebuffer reads.
	readFramebuffer := newReadFramebuffer(ctx)

	optimize := true

	for _, req := range requests {
		switch req := req.(type) {
		case issuesRequest:
			optimize = false
			if issues == nil {
				issues = newFindIssues(ctx, device)
			}
			issues.reportTo(req.out)

		case framebufferRequest:
			deadCodeElimination.Request(req.after)
			// HACK: Also ensure we have framebuffer before the atom.
			// TODO: Remove this and handle swap-buffers better.
			deadCodeElimination.Request(req.after - 1)

			switch req.attachment {
			case gfxapi.FramebufferAttachment_Depth:
				readFramebuffer.Depth(req.after, req.out)
			case gfxapi.FramebufferAttachment_Stencil:
				return fmt.Errorf("Stencil buffer attachments are not currently supported")
			default:
				idx := uint32(req.attachment - gfxapi.FramebufferAttachment_Color0)
				readFramebuffer.Color(req.after, req.width, req.height, idx, req.out)
			}

			cfg := cfg.(drawConfig)
			switch cfg.wireframeMode {
			case replay.WireframeMode_All:
				// TODO: Add only once
				transforms.Add(wireframe(ctx))
			case replay.WireframeMode_Overlay:
				transforms.Add(wireframeOverlay(ctx, req.after))
			}
		}
	}

	if optimize && !config.DisableDeadCodeElimination {
		atoms = atom.NewList() // DeadAtomRemoval generates atoms.
		transforms.Prepend(deadCodeElimination)
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
		jot.Fail(ctx, err, "Creating compatability transform")
	}

	// Cleanup
	transforms.Add(&destroyResourcesAtEOS{})

	if config.DebugReplay {
		ctx.Info().Logf("Replaying %d atoms using transform chain:", len(atoms.Atoms))
		for i, t := range transforms {
			ctx.Info().Logf("(%d) %#v", i, t)
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

	transforms.Transform(ctx, *atoms, out)
	return nil
}

func (a api) QueryIssues(
	ctx log.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	out chan<- replay.Issue) {

	c := issuesConfig{}
	r := issuesRequest{out: out}
	if err := mgr.Replay(ctx, intent, c, r, a); err != nil {
		out <- replay.Issue{Atom: atom.NoID, Error: err}
		close(out)
	}
}

func (a api) QueryFramebufferAttachment(
	ctx log.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	after atom.ID,
	width, height uint32,
	attachment gfxapi.FramebufferAttachment,
	wireframeMode replay.WireframeMode) (*image.Image2D, error) {

	c := drawConfig{wireframeMode: wireframeMode}
	if wireframeMode == replay.WireframeMode_Overlay {
		c.wireframeOverlayID = after
	}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, attachment: attachment, out: out}
	if err := mgr.Replay(ctx, intent, c, r, a); err != nil {
		return nil, err
	}
	select {
	case res := <-out:
		return res.img, res.err
	case <-task.ShouldStop(ctx):
		return nil, task.StopReason(ctx)
	}
}

// destroyResourcesAtEOS is a transform that destroys all textures,
// framebuffers, buffers, shaders, programs and vertex-arrays that were not
// destroyed by EOS.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx log.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	out.MutateAndWrite(ctx, id, a)
}

func (t *destroyResourcesAtEOS) Flush(ctx log.Context, out transform.Writer) {
	id := atom.NoID
	s := out.State()
	c := GetContext(s)
	if c == nil {
		return
	}

	// Delete all Renderbuffers.
	renderbuffers := make([]RenderbufferId, 0, len(c.Instances.Renderbuffers)-3)
	for renderbufferId := range c.Instances.Renderbuffers {
		// Skip virtual renderbuffers: backbuffer_color(-1), backbuffer_depth(-2), backbuffer_stencil(-3).
		if renderbufferId < 0xf0000000 {
			renderbuffers = append(renderbuffers, renderbufferId)
		}
	}
	if len(renderbuffers) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, renderbuffers))
		out.MutateAndWrite(ctx, id, NewGlDeleteRenderbuffers(GLsizei(len(renderbuffers)), tmp.Ptr()).AddRead(tmp.Data()))
	}

	// Delete all Textures.
	textures := make([]TextureId, 0, len(c.Instances.Textures))
	for textureId := range c.Instances.Textures {
		textures = append(textures, textureId)
	}
	if len(textures) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, textures))
		out.MutateAndWrite(ctx, id, NewGlDeleteTextures(GLsizei(len(textures)), tmp.Ptr()).AddRead(tmp.Data()))
	}

	// Delete all Framebuffers.
	framebuffers := make([]FramebufferId, 0, len(c.Instances.Framebuffers))
	for framebufferId := range c.Instances.Framebuffers {
		framebuffers = append(framebuffers, framebufferId)
	}
	if len(framebuffers) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, framebuffers))
		out.MutateAndWrite(ctx, id, NewGlDeleteFramebuffers(GLsizei(len(framebuffers)), tmp.Ptr()).AddRead(tmp.Data()))
	}

	// Delete all Buffers.
	buffers := make([]BufferId, 0, len(c.Instances.Buffers))
	for bufferId := range c.Instances.Buffers {
		buffers = append(buffers, bufferId)
	}
	if len(buffers) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, buffers))
		out.MutateAndWrite(ctx, id, NewGlDeleteBuffers(GLsizei(len(buffers)), tmp.Ptr()).AddRead(tmp.Data()))
	}

	// Delete all VertexArrays.
	vertexArrays := make([]VertexArrayId, 0, len(c.Instances.VertexArrays))
	for vertexArrayId := range c.Instances.VertexArrays {
		vertexArrays = append(vertexArrays, vertexArrayId)
	}
	if len(vertexArrays) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, vertexArrays))
		out.MutateAndWrite(ctx, id, NewGlDeleteVertexArrays(GLsizei(len(vertexArrays)), tmp.Ptr()).AddRead(tmp.Data()))
	}

	// Delete all Shaders.
	for _, shaderId := range c.Instances.Shaders.KeysSorted() {
		out.MutateAndWrite(ctx, id, NewGlDeleteShader(shaderId))
	}

	// Delete all Programs.
	for _, programId := range c.Instances.Programs.KeysSorted() {
		out.MutateAndWrite(ctx, id, NewGlDeleteProgram(programId))
	}

	// Delete all Queries.
	queries := make([]QueryId, 0, len(c.Instances.Queries))
	for queryId := range c.Instances.Queries {
		queries = append(queries, queryId)
	}
	if len(queries) > 0 {
		tmp := atom.Must(atom.AllocData(ctx, s, queries))
		out.MutateAndWrite(ctx, id, NewGlDeleteQueries(GLsizei(len(queries)), tmp.Ptr()).AddRead(tmp.Data()))
	}
}
