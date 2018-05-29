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

func TestSignalFire(t *testing.T) {
	ctx := log.Testing(t)
	signal, fire := task.NewSignal()
	assert.For(ctx, "Signal before fire").That(signal.Fired()).Equals(false)
	fire(ctx)
	assert.For(ctx, "Signal after fire").That(signal.Fired()).Equals(true)
}

func TestSignalWait(t *testing.T) {
	ctx := log.Testing(t)
	inSignal, inFire := task.NewSignal()
	outSignal, outFire := task.NewSignal()
	go func() {
		assert.For(ctx, "Out before wait").That(outSignal.Fired()).Equals(false)
		inFire(ctx)
		outSignal.Wait(ctx)
		assert.For(ctx, "Out after wait").That(outSignal.Fired()).Equals(true)
	}()
	inSignal.Wait(ctx)
	outFire(ctx)
}

func TestSignalTryWait(t *testing.T) {
	ctx := log.Testing(t)
	signal, fire := task.NewSignal()
	fire(ctx)
	assert.For(ctx, "sig").That(signal.TryWait(ctx, ExpectBlocking)).Equals(true)
}

func TestSignalTryWaitTimeout(t *testing.T) {
	ctx := log.Testing(t)
	assert := assert.To(t)
	signal, _ := task.NewSignal()
	before := time.Now()
	assert.For("wait").That(signal.TryWait(ctx, ExpectBlocking)).Equals(false)
	assert.For("duration").ThatDuration(time.Since(before)).IsAtMost(ExpectBlocking + ExpectNonBlocking)
}
