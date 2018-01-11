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
	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
)

type local struct {
	w worker.Manager
}

func (a *Action) JobID() string          { return a.Id }
func (a *Action) JobHost() string        { return a.Host }
func (a *Action) JobTarget() string      { return a.Target }
func (a *Action) JobInput() worker.Input { return a.Input }
func (a *Action) Init(id string, input worker.Input, w *job.Worker) {
	a.Id = id
	a.Input = input.(*Input)
	a.Host = w.Host
	a.Target = w.Target
}
func (t *Task) Init(id string, input worker.Input, w *job.Worker) {
	t.Action = id
	t.Input = input.(*Input)
}

// NewLocal builds a new local manager.
func NewLocal(ctx context.Context, library record.Library, jobManager job.Manager) (Manager, error) {
	l := &local{}
	return l, l.w.Init(ctx, library, jobManager, job.Report, &Action{}, &Task{})
}

// Search implements Manager.Search
// It searches the set of persisted actions, and supports monitoring of actions as they arrive.
func (l *local) Search(ctx context.Context, query *search.Query, handler ActionHandler) error {
	return l.w.Actions.Search(ctx, query, handler)
}

// Register implements Manager.Register
// See Workers.Register for more details on the implementation.
func (l *local) Register(ctx context.Context, host *device.Instance,
	target *device.Instance, handler TaskHandler) error {
	return l.w.Workers.Register(ctx, host, target, handler)
}

// Do implements Manager.Do
// See Workers.Do for more details on the implementation.
func (l *local) Do(ctx context.Context, device string, input *Input) (string, error) {
	return l.w.Do(ctx, device, input)
}

// Update implements Manager.Update
// See Workers.Update for more details on the implementation.
func (l *local) Update(ctx context.Context, action string, status job.Status, output *Output) error {
	return l.w.Update(ctx, &Action{Id: action, Status: status, Output: output})
}
