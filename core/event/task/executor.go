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

package task

import (
	"context"

	"github.com/google/gapid/core/app/crash"
)

// Executor is the signature for a function that executes a Task.
// When the task is invoked depends on the specific Executor.
// The returned handle can be used to wait for the task to complete and collect it's error return value.
type Executor func(ctx context.Context, task Task) Handle

// Direct is a synchronous implementation of an Executor that runs the task before returning.
// In general it is easier to just invoke the task directly, so this is only used in cases where you want to hand
// the Exectuor to something that is agnostic about how tasks are scheduled.
func Direct(ctx context.Context, task Task) Handle {
	h, r := Prepare(ctx, task)
	r()
	return h
}

// Go is an asynchronous implementation of an Executor that starts a new go routine to run the task.
func Go(ctx context.Context, task Task) Handle {
	h, r := Prepare(ctx, task)
	crash.Go(r)
	return h
}

// Pool returns a new Executor that uses a pool of goroutines to run the tasks, and a Task that shuts down the pool.
// The number of goroutines in the pool is controlled by parallel, and it must be greater than 0.
// The length of the submission queue is controlled by queue. It may be 0, in which case the executor will block until
// a goroutine is ready to accept the task. It will also block if the queue fills up.
// The shutdown task may only be called once, and it is an error to call the executor again after the shutdown task has
// run.
func Pool(queue int, parallel int) (Executor, Task) {
	q := make(chan Runner, queue)
	for i := 0; i < parallel; i++ {
		crash.Go(func() {
			for r := range q {
				r()
			}
		})
	}
	executor := func(ctx context.Context, task Task) Handle {
		h, r := Prepare(ctx, task)
		q <- r
		return h
	}
	shutdown := func(context.Context) error {
		close(q)
		return nil
	}
	return executor, shutdown
}

// Batch returns an executor that uses the supplied executor to run tasks, and automatically adds the completion signals
// for those tasks to the supplied Signals list.
func Batch(executor Executor, signals *Events) Executor {
	batch := func(ctx context.Context, task Task) Handle {
		s := executor(ctx, task)
		signals.Add(s)
		return s
	}
	return batch
}
