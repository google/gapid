// Copyright (C) 2020 Google Inc.
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

package vulkan

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(API{})
	_ = replay.QueryFramebufferAttachment(API{})
	_ = replay.Support(API{})
	_ = replay.QueryTimestamps(API{})
	_ = replay.Profiler(API{})
)

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	startScope                api.CmdID
	endScope                  api.CmdID
	subindices                string // drawConfig needs to be comparable, so we cannot use a slice
	drawMode                  path.DrawMode
	disableReplayOptimization bool
}

type imgRes struct {
	img *image.Data // The image data.
	err error       // The error that occurred generating the image.
}

// framebufferRequest requests a postback of a framebuffer's attachment.
type framebufferRequest struct {
	after            []uint64
	width, height    uint32
	attachment       api.FramebufferAttachmentType
	framebufferIndex uint32
	out              chan imgRes
	wireframeOverlay bool
	displayToSurface bool
}

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct {
}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out              chan<- replay.Issue
	displayToSurface bool
	loopCount        int32
}

type timestampsConfig struct {
}

type timestampsRequest struct {
	handler   service.TimeStampsHandler
	loopCount int32
}

// uniqueConfig returns a replay.Config that is guaranteed to be unique.
// Any requests made with a Config returned from uniqueConfig will not be
// batched with any other request.
func uniqueConfig() replay.Config {
	return &struct{}{}
}

type profileRequest struct {
	traceOptions   *service.TraceOptions
	handler        *replay.SignalHandler
	buffer         *bytes.Buffer
	handleMappings *map[uint64][]service.VulkanHandleMappingItem
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachmentType,
	framebufferIndex uint32,
	drawMode path.DrawMode,
	disableReplayOptimization bool,
	displayToSurface bool,
	hints *path.UsageHints) (*image.Data, error) {

	beginIndex := api.CmdID(0)
	endIndex := api.CmdID(0)
	subcommand := ""
	// We cant break up overdraw right now, but we can break up
	// everything else.
	if drawMode == path.DrawMode_OVERDRAW {
		if len(after) > 1 { // If we are replaying subcommands, then we can't batch at all
			beginIndex = api.CmdID(after[0])
			endIndex = api.CmdID(after[0])
			for i, j := range after[1:] {
				if i != 0 {
					subcommand += ":"
				}
				subcommand += fmt.Sprintf("%d", j)
			}
		}
	}

	c := drawConfig{beginIndex, endIndex, subcommand, drawMode, disableReplayOptimization}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, framebufferIndex: framebufferIndex, attachment: attachment, out: out, displayToSurface: displayToSurface}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, false)
	if err != nil {
		return nil, err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil, nil
	}
	return res.(*image.Data), nil
}

func (a API) QueryIssues(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	loopCount int32,
	displayToSurface bool,
	hints *path.UsageHints) ([]replay.Issue, error) {

	c, r := issuesConfig{}, issuesRequest{displayToSurface: displayToSurface, loopCount: loopCount}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, true)

	if err != nil {
		return nil, err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil, nil
	}
	return res.([]replay.Issue), nil
}

func (a API) QueryTimestamps(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	loopCount int32,
	handler service.TimeStampsHandler,
	hints *path.UsageHints) error {

	c, r := timestampsConfig{}, timestampsRequest{
		handler:   handler,
		loopCount: loopCount}
	_, err := mgr.Replay(ctx, intent, c, r, a, hints, false)
	if err != nil {
		return err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil
	}
	return nil
}

func (a API) QueryProfile(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	hints *path.UsageHints,
	traceOptions *service.TraceOptions) (*service.ProfilingData, error) {

	c := uniqueConfig()
	handler := replay.NewSignalHandler()
	var buffer bytes.Buffer
	handleMappings := make(map[uint64][]service.VulkanHandleMappingItem)
	r := profileRequest{traceOptions, handler, &buffer, &handleMappings}
	_, err := mgr.Replay(ctx, intent, c, r, a, hints, true)
	if err != nil {
		return nil, err
	}
	handler.DoneSignal.Wait(ctx)

	s, err := resolve.SyncData(ctx, intent.Capture)
	if err != nil {
		return nil, err
	}

	d, err := trace.ProcessProfilingData(ctx, intent.Device, intent.Capture, &buffer, &handleMappings, s)
	return d, err
}
