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

package dictionary_test

import (
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/log"
)

type provider struct {
	m map[string]int
}

func (p provider) Dictionary() dictionary.I { return dictionary.From(p.m) }

func TestDictionary(t *testing.T) {
	ctx := log.Testing(t)

	assert.For(ctx, "dictionary.From(nil)").That(dictionary.From(nil)).Equals(nil)

	newMap := func() map[string]int {
		return map[string]int{
			"apple":  1,
			"orange": 2,
			"grape":  3,
			"kiwi":   4,
			"lemon":  5,
		}
	}

	for _, test := range []struct {
		name   string
		source interface{}
	}{
		{"map", newMap()},
		{"provider", provider{newMap()}},
	} {
		ctx := log.Enter(ctx, test.name)

		d := dictionary.From(test.source)
		assert.For(ctx, "d.Get('apple')").That(d.Get("apple")).Equals(1)
		assert.For(ctx, "d.Get('grape')").That(d.Get("grape")).Equals(3)
		assert.For(ctx, "d.KeyTy()").That(d.KeyTy()).Equals(reflect.TypeOf(""))
		assert.For(ctx, "d.ValTy()").That(d.ValTy()).Equals(reflect.TypeOf(1))
		assert.For(ctx, "d.Len()").That(d.Len()).Equals(5)
		assert.For(ctx, "d.KeysSorted()").ThatSlice(d.KeysSorted()).Equals([]string{
			"apple", "grape", "kiwi", "lemon", "orange",
		})

		val, ok := d.Lookup("banana")
		assert.For(ctx, "d.Get('banana').val").That(val).Equals(0)
		assert.For(ctx, "d.Get('banana').ok").That(ok).Equals(false)
		assert.For(ctx, "d.Contains('banana')").That(d.Contains("banana")).Equals(false)

		val, ok = d.Lookup("orange")
		assert.For(ctx, "d.Get('orange').val").That(val).Equals(2)
		assert.For(ctx, "d.Get('orange').ok").That(ok).Equals(true)
		assert.For(ctx, "d.Contains('orange')").That(d.Contains("orange")).Equals(true)

		assert.For(ctx, "len(d.Entries())").That(len(d.Entries())).Equals(5)

		for _, e := range d.Entries() {
			assert.For(ctx, "d.Entries()[%v]", e.K).That(d.Get(e.K)).Equals(e.V)
		}

		// Below we start mutating the map.
		d.Set("kiwi", 40)
		assert.For(ctx, "d.Get('kiwi')").That(d.Get("kiwi")).Equals(40)

		d.Delete("kiwi")
		assert.For(ctx, "d.Len() <post-delete>").That(d.Len()).Equals(4)
		assert.For(ctx, "d.Get('kiwi') <post-delete>").That(d.Get("kiwi")).Equals(0)

		d.Clear()
		assert.For(ctx, "d.Len() <post-clear>").That(d.Len()).Equals(0)
	}
}
