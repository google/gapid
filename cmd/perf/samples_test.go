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

package main

import (
	"encoding/json"
	"testing"
	"time"

	"reflect"

	"github.com/google/gapid/core/assert"
)

func TestAnnotatedSampleJSONMarshaling(t *testing.T) {
	ctx := assert.Context(t)

	for _, s := range []struct {
		serialized string
		obj        Sample
	}{
		{`"0s"`, Sample(0)},
		{`"5s"`, Sample(5 * time.Second)},
	} {
		data, err := json.Marshal(s.obj)

		assert.With(ctx).ThatError(err).Succeeded()
		assert.With(ctx).ThatString(string(data)).Equals(s.serialized)

		var deserialized Sample
		err = json.Unmarshal(data, &deserialized)
		assert.With(ctx).ThatError(err).Succeeded()
		assert.With(ctx).That(deserialized).Equals(s.obj)
	}
}

func TestMultisampleMedian(t *testing.T) {
	ctx := assert.Context(t)

	s := &Multisample{}
	for _, x := range []time.Duration{0, 1, 3, 3, 6, 7, 8, 9, 1000} {
		s.Add(x * time.Second)
	}
	assert.With(ctx).That(s.Median()).Equals(6 * time.Second)

	s = &Multisample{}
	for _, x := range []time.Duration{1, 2, 3, 4, 5, 6, 8, 9} {
		s.Add(x * time.Minute)
	}
	assert.With(ctx).That(s.Median()).Equals(4*time.Minute + 30*time.Second)
}

func TestKeyedSamples_IndexedMultisamples(t *testing.T) {
	ctx := assert.Context(t)
	m := NewKeyedSamples()
	m.Add(2, 10*time.Second)
	m.Add(42, 30*time.Second)
	m.Add(1, 20*time.Second)

	expected := IndexedMultisamples{
		IndexedMultisample{Index: 1, Values: &Multisample{Sample(20 * time.Second)}},
		IndexedMultisample{Index: 2, Values: &Multisample{Sample(10 * time.Second)}},
		IndexedMultisample{Index: 42, Values: &Multisample{Sample(30 * time.Second)}},
	}

	assert.With(ctx).That(reflect.DeepEqual(m.IndexedMultisamples(), expected)).Equals(true)
}
