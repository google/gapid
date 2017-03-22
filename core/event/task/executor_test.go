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
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

func TestDirect(t *testing.T) {
	ctx := log.Testing(t)
	tasks := makeTasks(1)
	tasks[0].submit(ctx, false, task.Direct, nil)
	verifyTaskStates(ctx, tasks, true)

	child, cancel := task.WithCancel(ctx)
	cancel()
	tasks[0].submit(child, false, task.Direct, context.Canceled)
}

func TestGo(t *testing.T) {
	ctx := log.Testing(t)
	tasks := makeTasks(1)
	tasks[0].submit(ctx, true, task.Go, nil)
	verifyTaskStates(ctx, tasks, false)
	tasks[0].unblock(ctx)
	verifyTaskStates(ctx, tasks, true)

	child, cancel := task.WithCancel(ctx)
	cancel()
	tasks[0].submit(child, false, task.Go, context.Canceled)
}

func TestPool(t *testing.T) {
	ctx := log.Testing(t)
	tasks := makeTasks(3)
	pool, shutdown := task.Pool(len(tasks), 1)
	defer shutdown(ctx)
	for _, t := range tasks {
		t.submit(ctx, true, pool, nil)
	}
	verifyTaskStates(ctx, tasks, false, false, false)
	tasks[0].unblock(ctx)
	tasks[2].unblock(ctx)
	// 0 should run, 2 should not
	verifyTaskStates(ctx, tasks, true, false, false)
	tasks[1].unblock(ctx)
	verifyTaskStates(ctx, tasks, true, true, true)

	child, cancel := task.WithCancel(ctx)
	cancel()
	tasks[0].submit(child, false, pool, context.Canceled)
}

func TestBatch(t *testing.T) {
	ctx := log.Testing(t)
	tasks := makeTasks(4)
	pool, shutdown := task.Pool(0, len(tasks))
	defer shutdown(ctx)
	batches := [2]struct {
		executor task.Executor
		group    task.Events
		signal   task.Signal
	}{}
	verifySignalStates := func(expect ...bool) {
		for i, b := range batches {
			ctx := log.V{"Batch": i}.Bind(ctx)
			assert.For(ctx, "Signal").That(b.signal.TryWait(ctx, time.Millisecond)).Equals(expect[i])
		}
	}
	for i := range batches {
		batches[i].executor = task.Batch(pool, &batches[i].group)
	}
	for i, t := range tasks {
		t.submit(ctx, true, batches[i%len(batches)].executor, nil)
	}
	verifyTaskStates(ctx, tasks, false, false, false, false)
	for i := range batches {
		assert.For(ctx, "Batch length").That(batches[i].group.Pending()).Equals(2)
		batches[i].signal = batches[i].group.Join(ctx)
	}
	verifySignalStates(false, false)
	tasks[0].unblock(ctx)
	tasks[1].unblock(ctx)
	verifySignalStates(false, false)
	tasks[3].unblock(ctx)
	verifySignalStates(false, true)
	tasks[2].unblock(ctx)
	verifySignalStates(true, true)

	child, cancel := task.WithCancel(ctx)
	cancel()
	tasks[0].submit(child, false, batches[0].executor, context.Canceled)
}
