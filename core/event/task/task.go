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
	"time"
)

// Task is the unit of work used in the task system.
// Tasks should generally be reentrant, they may be run more than once in more than one executor, and should generally
// be agnostic as to whether they are run in parallel.
type Task func(context.Context) error

// Once wraps a task so that only the first invocation of the outer task invokes the inner task.
func Once(task Task) Task {
	once := sync.Once{}
	var err error
	return func(ctx context.Context) error {
		once.Do(func() { err = task(ctx) })
		return err
	}
}

// Retry repeatedly calls f until f returns a true, the number of attempts
// reaches maxAttempts or the context is cancelled. Retry will sleep for
// retryDelay between retry attempts.
// if maxAttempts <= 0, then there is no maximum limit to the number of times
// f will be called.
func Retry(ctx context.Context, maxAttempts int, retryDelay time.Duration, f func(context.Context) (retry bool, err error)) error {
	var count int
	for {
		done, err := f(ctx)
		if done {
			return err
		}
		count++
		if maxAttempts > 0 && count >= maxAttempts {
			return err
		}
		select {
		case <-ShouldStop(ctx):
			return StopReason(ctx)
		case <-time.After(retryDelay):
		}
	}
}
