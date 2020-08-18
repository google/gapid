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

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

var _ transform2.Transform = &waitForPerfetto{}

// waitForPerfetto adds a fence to the trace to be able to wait perfetto
type waitForPerfetto struct {
	traceOptions      *service.TraceOptions
	signalHandler     *replay.SignalHandler
	buffer            *bytes.Buffer
	beginProfileCmdID api.CmdID
}

type FenceCallback = func(ctx context.Context, request *gapir.FenceReadyRequest)

func newWaitForPerfetto(traceOptions *service.TraceOptions, signalHandler *replay.SignalHandler, buffer *bytes.Buffer, beginProfileCmdID api.CmdID) *waitForPerfetto {
	return &waitForPerfetto{
		traceOptions:      traceOptions,
		signalHandler:     signalHandler,
		buffer:            buffer,
		beginProfileCmdID: beginProfileCmdID,
	}
}

func (perfettoTransform *waitForPerfetto) RequiresAccurateState() bool {
	return false
}

func (perfettoTransform *waitForPerfetto) RequiresInnerStateMutation() bool {
	return false
}

func (perfettoTransform *waitForPerfetto) SetInnerStateMutationFunction(mutator transform2.StateMutator) {
	// This transform do not require inner state mutation
}

func (perfettoTransform *waitForPerfetto) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (perfettoTransform *waitForPerfetto) ClearTransformResources(ctx context.Context) {
	// Do nothing
}

func (perfettoTransform *waitForPerfetto) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	cmds := make([]api.Cmd, 0)
	cmds = append(cmds, createVkDeviceWaitIdleCommandsForDevices(ctx, inputState)...)

	fenceID := uint32(0x3ffffff)
	waitForFenceCmd := createWaitForFence(ctx, fenceID, func(ctx context.Context, request *gapir.FenceReadyRequest) {
		perfettoTransform.endOfTransformCallback(ctx, request)
	})
	cmds = append(cmds, waitForFenceCmd)
	return cmds, nil
}

func (perfettoTransform *waitForPerfetto) TransformCommand(ctx context.Context, id transform2.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if id.GetCommandType() != transform2.TransformCommand {
		return inputCommands, nil
	}

	if id.GetID() != perfettoTransform.beginProfileCmdID {
		return inputCommands, nil
	}

	inputCommands = append(inputCommands, createVkDeviceWaitIdleCommandsForDevices(ctx, inputState)...)
	waitForFenceCmd := createWaitForFence(ctx, uint32(id.GetID()), func(ctx context.Context, request *gapir.FenceReadyRequest) {
		perfettoTransform.transformCallback(ctx, request)
	})
	inputCommands = append(inputCommands, waitForFenceCmd)
	return inputCommands, nil
}

func (perfettoTransform *waitForPerfetto) transformCallback(ctx context.Context, request *gapir.FenceReadyRequest) {
	errChannel := make(chan error)
	signalHandler := perfettoTransform.signalHandler

	go func() {
		err := trace.TraceBuffered(ctx,
			perfettoTransform.traceOptions.Device,
			signalHandler.StartSignal,
			signalHandler.StopSignal,
			signalHandler.ReadyFunc,
			perfettoTransform.traceOptions,
			perfettoTransform.buffer)
		if err != nil {
			errChannel <- err
		}
		if !signalHandler.DoneSignal.Fired() {
			signalHandler.DoneFunc(ctx)
		}
	}()

	select {
	case err := <-errChannel:
		log.W(ctx, "Profiling error: %v", err)
		return
	case <-task.ShouldStop(ctx):
		return
	case <-signalHandler.ReadySignal:
		return
	}
}

func (perfettoTransform *waitForPerfetto) endOfTransformCallback(ctx context.Context, request *gapir.FenceReadyRequest) {
	if !perfettoTransform.signalHandler.StopSignal.Fired() {
		perfettoTransform.signalHandler.StopFunc(ctx)
	}
}

func createVkDeviceWaitIdleCommandsForDevices(ctx context.Context, inputState *api.GlobalState) []api.Cmd {
	cb := CommandBuilder{Thread: 0, Arena: inputState.Arena}
	allDevices := GetState(inputState).Devices().All()

	waitCmds := make([]api.Cmd, 0, len(allDevices))

	// Wait for all queues in all devices to finish their jobs first.
	for handle := range allDevices {
		waitCmds = append(waitCmds, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}

	return waitCmds
}

func createWaitForFence(ctx context.Context, id uint32, callback FenceCallback) api.Cmd {
	return replay.Custom{T: 0, F: func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		fenceID := id
		b.Wait(fenceID)
		tcb := func(p *gapir.FenceReadyRequest) {
			callback(ctx, p)
		}
		return b.RegisterFenceReadyRequestCallback(fenceID, tcb)
	}}
}
