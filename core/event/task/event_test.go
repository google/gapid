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
	"testing"
	"time"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

func TestEvents(t *testing.T) {
	ctx := log.Testing(t)
	signals := task.Events{}
	in1, in1fire := task.NewSignal()
	in2, in2fire := task.NewSignal()
	out, outfire := task.NewSignal()
	signals.Add(in1)
	assert.For(ctx, "Size after 1").That(signals.Pending()).Equals(1)
	signals.Add(in2)
	assert.For(ctx, "Size after 2").That(signals.Pending()).Equals(2)
	go func() {
		signals.Wait(ctx)
		assert.For(ctx, "TryWait after Wait").That(signals.TryWait(ctx, time.Millisecond)).Equals(true)
		outfire(ctx)
	}()
	assert.For(ctx, "Out before signal 1").That(out.TryWait(ctx, ExpectBlocking)).Equals(false)
	assert.For(ctx, "Pending list before signal 1").That(signals.Pending()).Equals(2)
	in1fire(ctx)
	assert.For(ctx, "Out between signals").That(out.TryWait(ctx, ExpectBlocking)).Equals(false)
	assert.For(ctx, "Pending list after signal 1").That(signals.Pending()).Equals(1)
	in2fire(ctx)
	assert.For(ctx, "Out after all signals").That(out.TryWait(ctx, ExpectNonBlocking)).Equals(true)
	assert.For(ctx, "Pending list after all signals").That(signals.Pending()).Equals(0)
}
