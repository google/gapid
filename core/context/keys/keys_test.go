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

package keys_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/context/keys"
)

func TestNoKeys(t *testing.T) {
	ctx := context.Background()
	list := keys.Get(ctx)
	if len(list) != 0 {
		t.Errorf("Background context had non zero sized key list")
	}
}

func TestKeys(t *testing.T) {
	ctx := context.Background()
	keyList := []interface{}{"A", "B"}
	initial := []interface{}{"a", "b"}
	override := []interface{}{"c", nil}
	for i, k := range keyList {
		ctx = keys.WithValue(ctx, k, initial[i])
	}
	for i, k := range keyList {
		if ctx.Value(k) != initial[i] {
			t.Errorf("Context did has %v for %v, expected %v", ctx.Value(k), k, initial[i])
		}
	}
	for i, k := range keyList {
		if override[i] != nil {
			ctx = keys.WithValue(ctx, k, override[i])
		}
	}
	for i, k := range keyList {
		v := override[i]
		if v == nil {
			v = initial[i]
		}
		if ctx.Value(k) != v {
			t.Errorf("Context did has %v for %v, expected %v", ctx.Value(k), k, v)
		}
	}
	list := keys.Get(ctx)
	if len(list) != len(keyList) {
		t.Errorf("Key list was incorrect, got %v expected %v", list, keyList)
	}
	for i, k := range keyList {
		if list[i] != k {
			t.Errorf("Key list was incorrect, got %v expected %v", list, keyList)
		}
	}
}

func TestManyKeys(t *testing.T) {
	ctx := context.Background()
	const max = 100
	for i := 0; i < max; i++ {
		ctx = keys.WithValue(ctx, i, fmt.Sprint(i))
	}
	list := keys.Get(ctx)
	if len(list) != max {
		t.Errorf("Key list was the wrong length, got %d expected %d", len(list), max)
	}
}

func TestChain(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		add    string
		expect string
	}{
		{"A", "A"},
		{"B", "A->B"},
		{"C", "A->B->C"},
	} {
		ctx = keys.Chain(ctx, "Test", test.add)
		result := fmt.Sprint(ctx.Value("Test"))
		if result != test.expect {
			t.Errorf("Key chain incorrect, got %v expected %v", result, test.expect)
		}
	}
}

func TestClone(t *testing.T) {
	a := context.Background()
	b := context.Background()
	a = keys.WithValue(a, "a", "A")
	b = keys.WithValue(b, "b", "B")

	got := fmt.Sprintf("%v", keys.Get(a))
	expect := "[a]"
	if got != expect {
		t.Errorf("Initial source incorrect, got %v expected %v", got, expect)
	}
	got = fmt.Sprintf("%v", keys.Get(b))
	expect = "[b]"
	if got != expect {
		t.Errorf("Initial dest incorrect, got %v expected %v", got, expect)
	}
	c := keys.Clone(b, a)
	got = fmt.Sprintf("%v", keys.Get(c))
	expect = "[a b]"
	if got != expect {
		t.Errorf("Initial dest incorrect, got %v expected %v", got, expect)
	}
}
