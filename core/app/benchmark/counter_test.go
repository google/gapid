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

package benchmark_test

import (
	"testing"
	"time"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/assert"
)

func TestCounterCollectionAndIntegerCounter(t *testing.T) {
	ctx := assert.Context(t)

	m := benchmark.NewCounters()

	i1 := m.Integer("int.one")
	i2 := m.Integer("int.two")

	i1.Increment()
	i2.Add(10)
	i1.Add(2)
	i2.Add(20)
	i1.Add(3)
	i2.Add(30)
	assert.With(ctx).That(i1.Get()).Equals(int64(6))
	assert.With(ctx).That(i2.Get()).Equals(int64(60))
	assert.With(ctx).That(m.Integer("int.two").Get()).Equals(int64(60))
	assert.With(ctx).That(m.Integer("int.one").Get()).Equals(int64(6))
	i1.Reset()
	assert.With(ctx).That(i1.Get()).Equals(int64(0))
}

func TestDurationCounter(t *testing.T) {
	ctx := assert.Context(t)

	m := benchmark.NewCounters()

	d := m.Duration("c")
	d.Add(60 * time.Second)
	d.Add(2 * time.Minute)

	assert.With(ctx).That(d.Get()).Equals(3 * time.Minute)
	d.Reset()
	assert.With(ctx).That(d.Get()).Equals(time.Duration(0))
}

func TestCounterMismatchPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Should have resulted in a panic.")
		}
	}()

	m := benchmark.NewCounters()

	m.Duration("d")
	m.Integer("d")
}

func TestIntegerCounterAtomicIncrement(t *testing.T) {
	cnt := benchmark.NewCounters().Integer("_")
	done := make(chan bool)
	for i := 0; i < 16; i++ {
		go func() {
			for i := 0; i < 1024; i++ {
				cnt.Increment()
			}
			done <- true
		}()
	}
	for i := 0; i < 16; i++ {
		<-done
	}
	assert.With(assert.Context(t)).That(cnt.Get()).Equals(int64(16384))
}
