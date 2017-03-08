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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

const (
	ExpectBlocking    = time.Millisecond
	ExpectNonBlocking = time.Second
)

type testTask struct {
	name    string
	unblock task.Task
	in      task.Signal
	handle  task.Handle
	ran     bool
}

func (t *testTask) submit(ctx context.Context, blocked bool, e task.Executor, expect error) {
	t.in = nil
	if blocked {
		t.in, t.unblock = task.NewSignal()
	}
	t.ran = false
	ctx = log.V{"Task": t.name}.Bind(ctx)
	handle := e(ctx, t.run)
	t.handle = handle
	if expect != nil {
		assert.For(ctx, "Submit").ThatError(t.handle.Result(ctx)).Equals(expect)
	}
}

func (t *testTask) run(ctx context.Context) error {
	if t.in != nil {
		t.in.Wait(ctx)
	}
	t.ran = true
	return nil
}

func makeTasks(count int) []*testTask {
	tasks := make([]*testTask, count)
	for i := range tasks {
		tasks[i] = &testTask{name: fmt.Sprintf("task:%d", i)}
	}
	return tasks
}

func yieldTasks(ctx context.Context, tasks []*testTask) {
	for _, t := range tasks {
		if !t.handle.TryWait(ctx, ExpectBlocking) {
			return
		}
	}
}

func verifyTaskStates(ctx context.Context, tasks []*testTask, states ...bool) {
	if len(tasks) != len(states) {
		log.E(ctx, "State count does not match task count")
	}
	yieldTasks(ctx, tasks)
	for i, t := range tasks {
		ctx := log.Enter(ctx, t.name)
		assert.For(ctx, "Run state").That(t.ran).Equals(states[i])
		assert.For(ctx, "Fired state").That(t.handle.Fired()).Equals(t.ran)
	}
}

func TestOnce(t *testing.T) {
	ctx := log.Testing(t)
	count := 0
	counter := func(context.Context) error { count++; return nil }
	assert.For(ctx, "Count before run").That(count).Equals(0)
	counter(ctx)
	assert.For(ctx, "Count after run").That(count).Equals(1)
	once := task.Once(counter)
	once(ctx)
	assert.For(ctx, "Count after once").That(count).Equals(2)
	once(ctx)
	assert.For(ctx, "Count after repeat").That(count).Equals(2)
}
