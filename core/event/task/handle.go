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

// Handle is a reference to a running task submitted to an executor.
// It can be used to check if the task has completed and get it's error result if it has one.
type Handle struct {
	Signal
	err *error
}

// Result returns the error result of the task.
// It will block until the task has completed or the context is cancelled.
func (h Handle) Result(ctx context.Context) error {
	if !h.Signal.Wait(ctx) {
		return StopReason(ctx)
	}
	return *h.err
}
