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

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// Generator is the interface for types that support replay generation.
type Generator interface {
	// Replay is called when a replay pass is ready to be sent to the replay
	// device. Replay may filter or transform the list of atoms, satisfying all
	// the specified requests and config, before outputting the final atom stream
	// to out.
	Replay(
		ctx context.Context,
		intent Intent,
		cfg Config,
		requests []RequestAndResult,
		device *device.Instance,
		capture *capture.Capture,
		out transform.Writer) error
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
// specific atom.
type Request interface{}

// Result is the function called for the result of a request.
// One of val and err must be nil.
type Result func(val interface{}, err error)

// RequestAndResult is a pair of Request and Result.
type RequestAndResult struct {
	Request Request
	Result  Result
}
