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

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/assert"
)

func TestConstantTime(t *testing.T) {
	assert := assert.To(t)
	for _, test := range []struct {
		samples benchmark.Samples
		fit     benchmark.Fit
	}{
		{benchmark.Samples{{5, 100}, {7, 100}, {10, 100}}, benchmark.NewLinearFit(100, 0)},
		{benchmark.Samples{{5, 100}, {7, 102}, {10, 105}}, benchmark.NewLinearFit(95, 1)},
	} {
		fit := test.samples.Analyse()
		assert.For("fit").That(fit).Equals(test.fit)
	}
}
