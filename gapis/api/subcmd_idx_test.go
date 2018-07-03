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

package api_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

func TestSubcommandLessThan(t *testing.T) {
	ctx := log.Testing(t)
	assert.For(ctx, "0 < 1").That(api.SubCmdIdx{0}.LessThan(api.SubCmdIdx{1})).Equals(true)
	assert.For(ctx, "1 < 0").That(api.SubCmdIdx{1}.LessThan(api.SubCmdIdx{0})).Equals(false)
	assert.For(ctx, "0 < 0").That(api.SubCmdIdx{0}.LessThan(api.SubCmdIdx{0})).Equals(false)
	assert.For(ctx, "0.0 < 0.1").That(api.SubCmdIdx{0, 0}.LessThan(api.SubCmdIdx{0, 1})).Equals(true)
	assert.For(ctx, "1.0 < 0.1").That(api.SubCmdIdx{1, 0}.LessThan(api.SubCmdIdx{0, 1})).Equals(false)
	assert.For(ctx, "1.0 < 0.1").That(api.SubCmdIdx{1, 0}.LessThan(api.SubCmdIdx{0, 1})).Equals(false)

	assert.For(ctx, "1.0 < 1").That(api.SubCmdIdx{1, 0}.LessThan(api.SubCmdIdx{1})).Equals(true)
	assert.For(ctx, "1 < 1.0").That(api.SubCmdIdx{1}.LessThan(api.SubCmdIdx{1, 0})).Equals(false)
}

func deceq(s api.SubCmdIdx, s2 api.SubCmdIdx) bool {
	r := s
	r.Decrement()
	return !((r.LessThan(s2)) || s2.LessThan(r))
}

func TestDecrement(t *testing.T) {
	ctx := log.Testing(t)
	assert.For(ctx, "1 deceq 0").That(deceq(api.SubCmdIdx{1}, api.SubCmdIdx{0})).Equals(true)
	assert.For(ctx, "1.1 deceq 1.0").That(deceq(api.SubCmdIdx{1, 1}, api.SubCmdIdx{1, 0})).Equals(true)
	assert.For(ctx, "1.0 deceq 0").That(deceq(api.SubCmdIdx{1, 0}, api.SubCmdIdx{0})).Equals(true)
	assert.For(ctx, "2.3.4 deceq 2.3.3").That(deceq(api.SubCmdIdx{2, 3, 4}, api.SubCmdIdx{2, 3, 3})).Equals(true)
	assert.For(ctx, "0 deceq {}").That(deceq(api.SubCmdIdx{0}, api.SubCmdIdx{})).Equals(true)
	assert.For(ctx, "2.3.0 deceq 2.2").That(deceq(api.SubCmdIdx{2, 3, 0}, api.SubCmdIdx{2, 2})).Equals(true)
}

func TestContains(t *testing.T) {
	ctx := log.Testing(t)
	assert.For(ctx, "{} contains 0").That(api.SubCmdIdx{}.Contains(api.SubCmdIdx{0})).Equals(false)
	assert.For(ctx, "0 contains 0").That(api.SubCmdIdx{0}.Contains(api.SubCmdIdx{0})).Equals(true)
	assert.For(ctx, "0 contains 1").That(api.SubCmdIdx{0}.Contains(api.SubCmdIdx{1})).Equals(false)
	assert.For(ctx, "0 contains 0.1").That(api.SubCmdIdx{0}.Contains(api.SubCmdIdx{0, 1})).Equals(true)
	assert.For(ctx, "0.1 contains 0.1.2.3.4").That(api.SubCmdIdx{0, 1}.Contains(api.SubCmdIdx{0, 1, 2, 3, 4})).Equals(true)
	assert.For(ctx, "0.1 contains 0.1.2.3.4").That(api.SubCmdIdx{0, 2}.Contains(api.SubCmdIdx{0, 1, 2, 3, 4})).Equals(false)
	assert.For(ctx, "1.2.3.4 contains 1").That(api.SubCmdIdx{1, 2, 3, 4}.Contains(api.SubCmdIdx{1})).Equals(false)
	assert.For(ctx, "1.2.3.4 contains 1.2.3").That(api.SubCmdIdx{1, 2, 3, 4}.Contains(api.SubCmdIdx{1, 2, 3})).Equals(false)
	assert.For(ctx, "1.2.3.4 contains 1.2.3.4.5").That(api.SubCmdIdx{1, 2, 3, 4}.Contains(api.SubCmdIdx{1, 2, 3, 4, 5})).Equals(true)
}
