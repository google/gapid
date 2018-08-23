// Copyright (C) 2018 Google Inc.
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

package status

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

type taskKeyTy string

const taskKey = taskKeyTy("task")

// PutTask attaches a task to a Context.
func PutTask(ctx context.Context, t *Task) context.Context {
	return keys.WithValue(ctx, taskKey, t)
}

// GetTask retrieves the task from a context previously annotated by PutTask.
func GetTask(ctx context.Context) *Task {
	val := ctx.Value(taskKey)
	if val == nil {
		return nil
	}
	return val.(*Task)
}
