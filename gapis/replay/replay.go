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

package replay

import (
	"context"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// Generator is the interface for types that support replay generation.
type Generator interface {
	// Replay is called when a replay pass is ready to be sent to the replay
	// device. Replay may filter or transform the list of commands, satisfying
	// all the specified requests and config, before outputting the final
	// command stream to out.
	Replay(
		ctx context.Context,
		intent Intent,
		cfg Config,
		dependentPayload string,
		requests []RequestAndResult,
		device *device.Instance,
		capture *capture.GraphicsCapture,
		out transform2.Writer) error
}

// SplitGenerator is the interface for types that support
// split-replay generation.
type SplitGenerator interface {
	Generator
	// GetInitialPayload returns a set of instructions
	// that can be used to set up the replay.
	GetInitialPayload(ctx context.Context,
		capture *path.Capture,
		device *device.Instance,
		out transform2.Writer) error
	// CleanupResources returns a set of instructions
	// that can be used to clean up from the Initial payload.
	CleanupResources(ctx context.Context,
		device *device.Instance,
		out transform2.Writer) error
}

// Intent describes the source capture and replay target information used for
// issuing a replay request.
type Intent struct {
	Device  *path.Device  // The path to the device being used for replay.
	Capture *path.Capture // The path to the capture that is being replayed.
}

// Config is a user-defined type used to describe the type of replay being
// requested. Replay requests made with configs that have equality (==) will
// likely be batched into the same replay pass. Configs can be used to force
// requests into different replay passes. For example, by issuing requests with
// different configs we can prevent a profiling Request from being issued in the
// same pass as a Request to render all draw calls in wireframe.
type Config interface{}

// Request is a user-defined type that holds information relevant to a single
// replay request. An example Request would be one that informs ReplayTransforms
// to insert a postback of the currently bound render-target content at a
// specific command.
type Request interface{}

// Result is the function called for the result of a request.
// One of val and err must be nil.
type Result func(val interface{}, err error)

// Do calls f and passes the return value-error pair to Result.
func (r Result) Do(f func() (val interface{}, err error)) error {
	val, err := f()
	if err != nil {
		r(nil, err)
		return err
	}
	r(val, nil)
	return nil
}

// Transform returns a new Result that passes the non-error value through f
// before calling the original r.
func (r Result) Transform(f func(in interface{}) (out interface{}, err error)) Result {
	return func(val interface{}, err error) {
		if err != nil {
			r(nil, err)
			return
		}
		if val, err := f(val); err != nil {
			r(nil, err)
		} else {
			r(val, nil)
		}
	}
}

// RequestAndResult is a pair of Request and Result.
type RequestAndResult struct {
	Request Request
	Result  Result
}

//TODO(apbodnar) move this into whatever eventually calls Profile()
type SignalHandler struct {
	StartSignal task.Signal
	StartFunc   task.Task
	ReadySignal task.Signal
	ReadyFunc   task.Task
	StopSignal  task.Signal
	StopFunc    task.Task
	DoneSignal  task.Signal
	DoneFunc    task.Task
	Written     int64
	Err         error
}

func NewSignalHandler() *SignalHandler {
	startSignal, startFunc := task.NewSignal()
	readySignal, readyFunc := task.NewSignal()
	stopSignal, stopFunc := task.NewSignal()
	doneSignal, doneFunc := task.NewSignal()

	handler := &SignalHandler{
		StartSignal: startSignal,
		StartFunc:   startFunc,
		ReadySignal: readySignal,
		ReadyFunc:   readyFunc,
		StopSignal:  stopSignal,
		StopFunc:    stopFunc,
		DoneSignal:  doneSignal,
		DoneFunc:    doneFunc,
	}
	return handler
}
