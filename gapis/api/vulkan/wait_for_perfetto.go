// Copyright (C) 2019 Google Inc.
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

// Package vulkan implementes the API interface for the Vulkan graphics library.

package vulkan

import (
	"context"

	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

type WaitForPerfetto struct {
	wff replay.WaitForFence
}

func addVkDeviceWaitIdle(ctx context.Context, out transform.Writer) {
	s := out.State()
	so := getStateObject(s)
	id := api.CmdNoID
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}

	// Wait for all queues in all devices to finish their jobs first.
	for handle := range so.Devices().All() {
		out.MutateAndWrite(ctx, id, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}
}

func waitTest(ctx context.Context, id api.CmdID, cmd api.Cmd) bool {
	if id == 0 {
		return true
	}
	return false
}

func (t *WaitForPerfetto) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	if waitTest(ctx, id, cmd) {
		addVkDeviceWaitIdle(ctx, out)
	}
	t.wff.Transform(ctx, id, cmd, out)
}

func (t *WaitForPerfetto) Flush(ctx context.Context, out transform.Writer) {
	addVkDeviceWaitIdle(ctx, out)
	t.wff.Flush(ctx, out)
}

func (t *WaitForPerfetto) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *WaitForPerfetto) PostLoop(ctx context.Context, out transform.Writer) {}

func NewWaitForPerfetto(traceOptions *service.TraceOptions, h *replay.SignalHandler) *WaitForPerfetto {
	tcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		go func() {
			trace.Trace(ctx, traceOptions.Device, h.StartSignal, h.StopSignal, h.ReadyFunc, traceOptions, &h.Written)
			if !h.DoneSignal.Fired() {
				h.DoneFunc(ctx)
			}
		}()
		h.ReadySignal.Wait(ctx)
	}

	fcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		if !h.StopSignal.Fired() {
			h.StopFunc(ctx)
		}
	}

	return &WaitForPerfetto{wff: replay.WaitForFence{tcb, fcb, waitTest}}
}
