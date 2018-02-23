// Copyright (C) 2018 Google Inc.
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

package data_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/log"
)

func TestDedupe(t *testing.T) {
	ctx := log.Testing(t)

	B := func(s string) []byte { return ([]byte)(s) }
	slices := [][]byte{
		B("cat says meow"),
		B("says"),
		B("the cat says meow. the dog says woof. "),
		B("fish says blub"),
	}
	expected := B("the cat says meow. the dog says woof. fish says blub")
	deduped, indices := data.Dedupe(slices)
	if assert.For(ctx, "got").ThatString(string(deduped)).Equals(string(expected)) {
		for i, slice := range slices {
			g := string(deduped[indices[i] : indices[i]+len(slice)])
			e := string(slice)
			assert.For(ctx, "%v", i).ThatString(g).Equals(e)
		}
	}
}
