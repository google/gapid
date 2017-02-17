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

package task_test

import (
	"testing"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

func TestExecutorFactory(t *testing.T) {
	ctx := log.Testing(t)
	const channelCount = 2
	tasks := makeTasks(4)
	factory := task.CachedExecutorFactory(ctx, task.PoolExecutorFactory(ctx, 2, 1))
	for i, t := range tasks {
		e, _ := factory(ctx, i%channelCount)
		t.submit(ctx, true, e, nil)
	}
	verifyTaskStates(ctx, tasks, false, false, false, false)
	// 0 and 1 should be running and blocked
	tasks[1].unblock(ctx)
	tasks[2].unblock(ctx)
	// 1 should run, 2 should not
	verifyTaskStates(ctx, tasks, false, true, false, false)
	tasks[0].unblock(ctx)
	verifyTaskStates(ctx, tasks, true, true, true, false)
	tasks[3].unblock(ctx)
	verifyTaskStates(ctx, tasks, true, true, true, true)
}

func TestExecutorFactoryInterlock(t *testing.T) {
	ctx := log.Testing(t)
	tasks := makeTasks(3)
	factory := task.CachedExecutorFactory(ctx, task.PoolExecutorFactory(ctx, 2, 1))
	e, _ := factory(ctx, 0)
	tasks[0].submit(ctx, true, e, nil)
	tasks[1].submit(ctx, true, e, nil)
	verifyTaskStates(ctx, tasks, false, false, false)
	tasks[0].unblock(ctx)
	// now the first one should run, the second should stay queued and the channel completion should enter the wait
	verifyTaskStates(ctx, tasks, true, false, false)
	// submit a new task to make wait complete with extras pending
	tasks[2].submit(ctx, true, e, nil)
	// and allow the current tasks to complete
	tasks[1].unblock(ctx)
	verifyTaskStates(ctx, tasks, true, true, false)
	tasks[2].unblock(ctx)
	verifyTaskStates(ctx, tasks, true, true, true)
}
