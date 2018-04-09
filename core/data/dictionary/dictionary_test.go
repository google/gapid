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
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/log"
)

type keyTy string
type valTy int

type custom struct{ m *map[keyTy]valTy }

func (g custom) Get(k keyTy) valTy            { return (*g.m)[k] }
func (g custom) Add(k keyTy, v valTy)         { (*g.m)[k] = v }
func (g custom) Lookup(k keyTy) (valTy, bool) { v, ok := (*g.m)[k]; return v, ok }
func (g custom) Contains(k keyTy) bool        { _, ok := (*g.m)[k]; return ok }
func (g custom) Remove(k keyTy)               { delete((*g.m), k) }
func (g custom) Len() int                     { return len((*g.m)) }
func (g custom) Keys() []keyTy {
	out := make([]keyTy, 0, len(*g.m))
	for k := range *g.m {
		out = append(out, k)
	}
	slice.Sort(out)
	return out
}

func TestIsSource(t *testing.T) {
	ctx := log.Testing(t)

	assert.For(ctx, "dictionary.IsSource(1)").ThatSlice(dictionary.IsSource(1)).IsNotEmpty()
	assert.For(ctx, "dictionary.IsSource(custom{})").ThatSlice(dictionary.IsSource(custom{})).IsEmpty()
}

func TestDictionary(t *testing.T) {
	ctx := log.Testing(t)

	assert.For(ctx, "dictionary.From(nil)").That(dictionary.From(nil)).Equals(nil)

	apple := keyTy("apple")
	orange := keyTy("orange")
	grape := keyTy("grape")
	kiwi := keyTy("kiwi")
	lemon := keyTy("lemon")
	banana := keyTy("banana")

	one := valTy(1)
	two := valTy(2)
	three := valTy(3)
	four := valTy(4)
	five := valTy(5)

	newMap := func() map[keyTy]valTy {
		return map[keyTy]valTy{
			apple:  one,
			orange: two,
			grape:  three,
			kiwi:   four,
			lemon:  five,
		}
	}
	newCustom := func() custom { m := newMap(); return custom{&m} }

	for _, test := range []struct {
		name   string
		source interface{}
	}{
		{"map", newMap()},
		{"custom", newCustom()},
	} {
		ctx := log.Enter(ctx, test.name)

		d := dictionary.From(test.source)

		if !assert.For(ctx, "d").That(d).IsNotNil() {
			continue
		}

		assert.For(ctx, "d.Get('apple')").That(d.Get(apple)).Equals(one)
		assert.For(ctx, "d.Get('grape')").That(d.Get(grape)).Equals(three)
		assert.For(ctx, "d.KeyTy()").That(d.KeyTy()).Equals(reflect.TypeOf(apple))
		assert.For(ctx, "d.ValTy()").That(d.ValTy()).Equals(reflect.TypeOf(one))
		assert.For(ctx, "d.Len()").That(d.Len()).Equals(5)
		assert.For(ctx, "d.Keys()").ThatSlice(d.Keys()).Equals([]keyTy{
			apple, grape, kiwi, lemon, orange,
		})

		val, ok := d.Lookup(banana)
		assert.For(ctx, "d.Get('banana').val").That(val).Equals(valTy(0))
		assert.For(ctx, "d.Get('banana').ok").That(ok).Equals(false)
		assert.For(ctx, "d.Contains('banana')").That(d.Contains(banana)).Equals(false)

		val, ok = d.Lookup(orange)
		assert.For(ctx, "d.Get('orange').val").That(val).Equals(two)
		assert.For(ctx, "d.Get('orange').ok").That(ok).Equals(true)
		assert.For(ctx, "d.Contains('orange')").That(d.Contains(orange)).Equals(true)

		assert.For(ctx, "len(Entries(d))").That(len(dictionary.Entries(d))).Equals(5)

		for _, e := range dictionary.Entries(d) {
			assert.For(ctx, "Entries(d)[%v]", e.K).That(d.Get(e.K)).Equals(e.V)
		}

		// Below we start mutating the map.
		d.Add(kiwi, valTy(40))
		assert.For(ctx, "d.Add('kiwi')").That(d.Get(kiwi)).Equals(valTy(40))

		d.Remove(kiwi)
		assert.For(ctx, "d.Len() <post-remove>").That(d.Len()).Equals(4)
		assert.For(ctx, "d.Get('kiwi') <post-remove>").That(d.Get(kiwi)).Equals(valTy(0))

		dictionary.Clear(d)
		assert.For(ctx, "d.Len() <post-clear>").That(d.Len()).Equals(0)
	}
}
