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

import "context"

// Runner is the type for a task that has been prepared to run by an executor.
// Invoking the runner will execute the underlying task, and trigger the signal when it completes.
type Runner func()

// Prepare is used to build a new Signal,Runner pair for a Task.
// The Signal will be closed when the task completes.
// The same task can be passed to Run multiple times, and will build a new Signal, Runner pair each time, but the
// returned runner should be executed exactly once, which will run the Task.
// In general this method is only used by Executor implementations when scheduling new tasks.
func Prepare(ctx context.Context, task Task) (Handle, Runner) {
	var result error
	signal, fire := NewSignal()
	runner := func() {
		defer fire(ctx)
		if Stopped(ctx) {
			result = StopReason(ctx)
		} else {
			result = task(ctx)
		}
	}
	return Handle{signal, &result}, runner
}
