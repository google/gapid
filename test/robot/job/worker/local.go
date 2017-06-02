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

package worker

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/record"
)

// Manager is a helper for writing local manager implementation of worker types (such as trace and replay).
// It has all the code needed to manage and persist the list of workers and actions those workers are performing.
type Manager struct {
	//Actions holds the persisted list of actions the service has had added to it.
	Actions Actions
	// Workers holds the dynamic list of live workers for the service.
	Workers Workers
}

// Init is called to initialised the action store from the supplied library.
// It loads the action ledger and prepares the empty worker list.
// The op is the type of action this manager supports, the nullAction is the empty version of the action the manager
// holds, and the nullTask is the empty version of the task sent to the managers workers.
func (m *Manager) Init(ctx context.Context, library record.Library, jobs job.Manager, op job.Operation, nullAction Action, nullTask Task) error {
	m.Workers.init(ctx, jobs, op)
	return m.Actions.init(ctx, library, op, nullAction, nullTask)
}

// Do is called to add a new action to the service.
// It is handed the id of the target device to run it on, and the inputs to the action.
// The active worker that owns the target device will be looked up, and then a new action
// created for the target worker and provided input will be added to the store.
func (m *Manager) Do(ctx context.Context, device string, input Input) (string, error) {
	ctx = log.V{"device": device}.Bind(ctx)
	w := m.Workers.Find(ctx, device)
	if w == nil {
		return "", log.Err(ctx, nil, "Could not find worker device")
	}
	return m.Actions.do(ctx, w, input)
}

// Update is used to update an existing action, normally to update it status and/or set it's outputs.
func (m *Manager) Update(ctx context.Context, action Action) error {
	return m.Actions.update(ctx, action)
}
