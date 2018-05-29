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

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
)

func TestBatonYield(t *testing.T) {
	ctx := log.Testing(t)
	baton := task.NewBaton()
	expect := []string{"A", "B", "C", "D"}
	got := []string{}
	signal, done := task.NewSignal()
	go func() {
		got = append(got, "A")
		baton.Yield(nil)
		got = append(got, "C")
		baton.Release(nil)
	}()
	go func() {
		baton.Acquire()
		got = append(got, "B")
		baton.Yield(nil)
		got = append(got, "D")
		done(ctx)
	}()
	assert.For(ctx, "Interlock complete").That(signal.TryWait(ctx, ExpectNonBlocking)).Equals(true)
	assert.For(ctx, "Interlock order").That(got).DeepEquals(expect)
}

func TestBatonRelay(t *testing.T) {
	ctx := log.Testing(t)
	baton := task.NewBaton()
	expect := []string{"A", "B"}
	got := []string{}
	signal, done := task.NewSignal()
	go func() {
		got = append(got, "A")
		baton.Yield(nil)
		got = append(got, "B")
		done(ctx)
	}()
	go baton.Relay()
	assert.For(ctx, "Replay complete").That(signal.TryWait(ctx, ExpectNonBlocking)).Equals(true)
	assert.For(ctx, "Replay order").That(got).DeepEquals(expect)
}

func TestBatonTryRelease(t *testing.T) {
	assert := assert.To(t)
	baton := task.NewBaton()
	go baton.Acquire()
	assert.For("Baton TryRelease").That(baton.TryRelease(nil, ExpectNonBlocking)).Equals(true)
}

func TestBatonTryReleaseBlocks(t *testing.T) {
	assert := assert.To(t)
	baton := task.NewBaton()
	assert.For("Baton TryRelease").That(baton.TryRelease(nil, ExpectNonBlocking)).Equals(false)
}

func TestBatonTryAcquire(t *testing.T) {
	assert := assert.To(t)
	baton := task.NewBaton()
	expect := 1
	go baton.Release(expect)
	got, ok := baton.TryAcquire(ExpectNonBlocking)
	assert.For("Baton TryAcquire").That(ok).Equals(true)
	assert.For("Baton value").That(got).Equals(expect)
}

func TestBatonTryAcquireBlocks(t *testing.T) {
	assert := assert.To(t)
	baton := task.NewBaton()
	_, ok := baton.TryAcquire(ExpectBlocking)
	assert.For("Baton TryAcquire").That(ok).Equals(false)
}
