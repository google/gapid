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
	"encoding/json"
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
	i2.AddInt64(10)
	i1.AddInt64(2)
	i2.AddInt64(20)
	i1.AddInt64(3)
	i2.AddInt64(30)
	assert.With(ctx).That(i1.GetInt64()).Equals(int64(6))
	assert.With(ctx).That(i1.Get()).Equals(int64(6))
	assert.With(ctx).That(i2.GetInt64()).Equals(int64(60))
	assert.With(ctx).That(m.Integer("int.two").GetInt64()).Equals(int64(60))
	assert.With(ctx).That(m.Integer("int.one").GetInt64()).Equals(int64(6))
	i1.Reset()
	assert.With(ctx).That(i1.GetInt64()).Equals(int64(0))
}

func TestDurationCounter(t *testing.T) {
	ctx := assert.Context(t)

	m := benchmark.NewCounters()

	d := m.Duration("c")
	d.AddDuration(60 * time.Second)
	d.AddDuration(2 * time.Minute)

	assert.With(ctx).That(d.GetDuration()).Equals(3 * time.Minute)
	assert.With(ctx).That(d.Get()).Equals(3 * time.Minute)
	d.Reset()
	assert.With(ctx).That(d.GetDuration()).Equals(time.Duration(0))
}

func TestStringHolder(t *testing.T) {
	ctx := assert.Context(t)

	m := benchmark.NewCounters()

	text := "hello"
	c := m.String("f")
	c.SetString(text)
	assert.With(ctx).ThatString(c.GetString()).Equals(text)
	assert.With(ctx).That(c.Get()).Equals(text)
}

func TestLazyCounter(t *testing.T) {
	ctx := assert.Context(t)

	m := benchmark.NewCounters()

	called := 0
	m.LazyWithFunction("l", func() benchmark.Counter {
		called = called + 1
		return benchmark.StringHolderOf("hello")
	})
	assert.With(ctx).ThatInteger(called).Equals(0)
	assert.With(ctx).That(m.Lazy("l").Get()).Equals("hello")
	assert.With(ctx).ThatInteger(called).Equals(1)
	assert.With(ctx).That(m.Lazy("l").Get()).Equals("hello")
	assert.With(ctx).ThatInteger(called).Equals(2)

	called = 0
	m.LazyWithCaching("m", func() benchmark.Counter {
		called = called + 1
		return benchmark.StringHolderOf("hello")
	})
	assert.With(ctx).ThatInteger(called).Equals(0)
	assert.With(ctx).That(m.Lazy("m").Get()).Equals("hello")
	assert.With(ctx).ThatInteger(called).Equals(1)
	assert.With(ctx).That(m.Lazy("m").Get()).Equals("hello")
	assert.With(ctx).ThatInteger(called).Equals(1)
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

func TestJson(t *testing.T) {
	ctx := assert.Context(t)

	initial := benchmark.NewCounters()
	initial.Integer("myIntCounter").SetInt64(12345)
	initial.Duration("myDurationCounter").SetDuration(38 * time.Minute)
	initial.Integer("someOtherCounter").SetInt64(-42)
	initial.Lazy("theLazy").SetFunction(func() benchmark.Counter {
		return benchmark.IntegerCounterOf(128)
	})

	marshaled, err := json.Marshal(initial)
	assert.With(ctx).ThatError(err).Succeeded()
	assert.With(ctx).ThatString(string(marshaled)).Equals(
		`{"myDurationCounter":{"Type":"duration","Value":"38m0s"},` +
			`"myIntCounter":{"Type":"int","Value":12345},` +
			`"someOtherCounter":{"Type":"int","Value":-42},` +
			`"theLazy":{"Type":"int","Value":128}}`)

	unmarshaled := benchmark.NewCounters()
	assert.With(ctx).ThatError(json.Unmarshal(marshaled, unmarshaled)).Succeeded()

	marshaledAgain, err := json.Marshal(unmarshaled)
	assert.With(ctx).ThatError(err).Succeeded()

	assert.With(ctx).ThatString(string(marshaledAgain)).Equals(string(marshaled))
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
	assert.With(assert.Context(t)).That(cnt.GetInt64()).Equals(int64(16384))
}
