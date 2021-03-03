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
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

func getPerfettoLoopCallbacks(traceOptions *service.TraceOptions, signalHandler *replay.SignalHandler, buffer *bytes.Buffer) loopCallbacks {

	preLoop := func(ctx context.Context, request *gapir.FenceReadyRequest) {

		errChannel := make(chan error)

		go func() {

			err := trace.TraceBuffered(ctx,
				traceOptions.Device,
				signalHandler.StartSignal,
				signalHandler.StopSignal,
				signalHandler.ReadyFunc,
				traceOptions,
				buffer)
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

	postLoop := func(ctx context.Context, request *gapir.FenceReadyRequest) {
		if !signalHandler.StopSignal.Fired() {
			signalHandler.StopFunc(ctx)
		}
	}

	return loopCallbacks{preLoop: preLoop, postLoop: postLoop}
}
