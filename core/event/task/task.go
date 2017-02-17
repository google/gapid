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
	"sync"

	"github.com/google/gapid/core/log"
)

// Task is the unit of work used in the task system.
// Tasks should generally be reentrant, they may be run more than once in more than one executor, and should generally
// be agnostic as to whether they are run in parallel.
type Task func(log.Context) error

// Once wraps a task so that only the first invocation of the outer task invokes the inner task.
func Once(task Task) Task {
	once := sync.Once{}
	var err error
	return func(ctx log.Context) error {
		once.Do(func() { err = task(ctx) })
		return err
	}
}
