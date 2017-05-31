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
package atom_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/test"
)

func TestToServiceToAtom(t *testing.T) {
	ctx := log.Testing(t)
	for n, a := range map[string]atom.Atom{"P": test.P, "Q": test.Q} {
		s, err := atom.ToService(a)
		if !assert.For(ctx, "ToService(%v)", n).ThatError(err).Succeeded() {
			continue
		}
		g, err := atom.ToAtom(s)
		if !assert.For(ctx, "ToAtom(%v)", n).ThatError(err).Succeeded() {
			continue
		}
		assert.For(ctx, "ToService(%v) -> ToAtom", n).That(g).DeepEquals(a)
	}
}
