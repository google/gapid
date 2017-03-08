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
	"sync"
)

// ExecutorFactory returns an executor for the specified channel id, and a Task to shut the executor down again.
// The Task may be nil for executors that don't need to be shutdown.
type ExecutorFactory func(ctx context.Context, id interface{}) (Executor, Task)

// PoolExecutorFactory builds an ExecutorFactory that makes a new Pool executor each time it is invoked.
// The returned ExecutorFactory ignores the channel id.
func PoolExecutorFactory(ctx context.Context, queue int, parallel int) ExecutorFactory {
	return func(ctx context.Context, id interface{}) (Executor, Task) {
		return Pool(queue, parallel)
	}
}

// CachedExecutorFactory builds an ExecutorFactory that makes new executors on demand using the underlying factory.
// It guarantees that all concurrent tasks delivered to the same id go to the same Executor, but it will drop
// executors that have no active tasks any more.
func CachedExecutorFactory(ctx context.Context, factory ExecutorFactory) ExecutorFactory {
	mutex := sync.Mutex{}
	pipes := map[interface{}]Executor{}
	return func(ctx context.Context, id interface{}) (Executor, Task) {
		mutex.Lock()
		defer mutex.Unlock()
		pipe, found := pipes[id]
		if found {
			return pipe, nil
		}
		// build a new executor
		executor, shutdown := factory(ctx, id)
		signals := Events{}
		batch := Batch(executor, &signals)
		first, start := NewSignal()
		start = Once(start)
		pipe = func(ctx context.Context, task Task) Handle {
			h := batch(ctx, task)
			// Tell the cleanup we have an entry
			start(ctx)
			return h
		}
		pipes[id] = pipe
		go func() {
			// Wait for the first submission to the new pipe
			first.Wait(ctx)
			// always shutdown the underlying executor when we quit
			if shutdown != nil {
				defer shutdown(ctx)
			}
			for {
				// wait for current group to finish
				signals.Wait(ctx)
				mutex.Lock()
				// Now check again under the mutex in case a new Submit since the Wait started
				if signals.Pending() == 0 {
					// All done, remove pipe while still under the lock
					// This means if a new Submit happens for the same id, it gets a brand new pipe
					pipes[id] = nil
					mutex.Unlock()
					return
				}
				// More to do, unlock and go round again
				mutex.Unlock()
			}
		}()
		return pipe, nil
	}
}
