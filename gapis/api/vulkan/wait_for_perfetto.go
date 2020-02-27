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
	"bytes"
	"context"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

type WaitForPerfetto struct {
	wff   replay.WaitForFence
	cmdId api.CmdID
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

func (t *WaitForPerfetto) waitTest(ctx context.Context, id api.CmdID, cmd api.Cmd) bool {
	// Set the command id we care about once we're in the "real" commands
	if id.IsReal() && t.cmdId == api.CmdNoID {
		t.cmdId = id
	}

	if id.IsReal() && id == t.cmdId {
		return true
	}
	return false
}

func (t *WaitForPerfetto) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	if t.waitTest(ctx, id, cmd) {
		addVkDeviceWaitIdle(ctx, out)
	}
	return t.wff.Transform(ctx, id, cmd, out)
}

func (t *WaitForPerfetto) Flush(ctx context.Context, out transform.Writer) error {
	addVkDeviceWaitIdle(ctx, out)
	return t.wff.Flush(ctx, out)
}

func (t *WaitForPerfetto) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *WaitForPerfetto) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *WaitForPerfetto) BuffersCommands() bool                              { return false }

func NewWaitForPerfetto(traceOptions *service.TraceOptions, h *replay.SignalHandler, buffer *bytes.Buffer) *WaitForPerfetto {
	tcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		errChannel := make(chan error)
		go func() {
			err := trace.TraceBuffered(ctx, traceOptions.Device, h.StartSignal, h.StopSignal, h.ReadyFunc, traceOptions, buffer)
			if err != nil {
				errChannel <- err
			}
			if !h.DoneSignal.Fired() {
				h.DoneFunc(ctx)
			}
		}()

		select {
		case err := <-errChannel:
			log.W(ctx, "Profiling error: %v", err)
			return
		case <-task.ShouldStop(ctx):
			return
		case <-h.ReadySignal:
			return
		}
	}

	fcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		if !h.StopSignal.Fired() {
			h.StopFunc(ctx)
		}
	}
	wfp := WaitForPerfetto{cmdId: api.CmdNoID}
	wfp.wff = replay.WaitForFence{tcb, fcb, wfp.waitTest}

	return &wfp
}
