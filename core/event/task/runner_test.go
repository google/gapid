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

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

func TestPrepare(t *testing.T) {
	ctx := log.Testing(t)
	tester := testTask{}
	handle, runner := task.Prepare(ctx, tester.run)
	assert.For(ctx, "Ran before run").That(tester.ran).Equals(false)
	assert.For(ctx, "Handle before run").That(handle.Fired()).Equals(false)
	runner()
	assert.For(ctx, "Ran after run").That(tester.ran).Equals(true)
	assert.For(ctx, "Handle after run").That(handle.Fired()).Equals(true)
	assert.For(ctx, "Prepare").ThatError(handle.Result(ctx)).Succeeded()

	child, cancel := task.WithCancel(ctx)
	cancel()
	handle, runner = task.Prepare(child, tester.run)
	runner()
	assert.For(ctx, "Prepare after cancel").ThatError(handle.Result(ctx)).Equals(context.Canceled)
}
