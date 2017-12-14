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

package report

import (
	"context"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/search"
)

// ActionHandler is a function that handles a stream of Actions.
type ActionHandler func(context.Context, *Action) error

// TaskHandler is a function that handles a stream of Tasks.
type TaskHandler func(context.Context, *Task) error

// Manager is the interface to a trace manager.
type Manager interface {
	// Search invokes handler with each output that matches the query.
	Search(ctx context.Context, query *search.Query, handler ActionHandler) error
	// Register registers a handler that will accept incoming tasks.
	Register(ctx context.Context, host *device.Instance, target *device.Instance, handler TaskHandler) error
	// Do asks the manager to send a task to a device.
	Do(ctx context.Context, device string, input *Input) (string, error)
	// Update adjusts the state of an action.
	Update(ctx context.Context, action string, status job.Status, output *Output) error
}
