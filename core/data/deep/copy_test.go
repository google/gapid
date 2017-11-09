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

package deep_test

import (
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data"
	"github.com/google/gapid/core/data/deep"
	"github.com/google/gapid/core/log"
)

var _ data.Assignable = &assignable{}

type assignable struct{ hidden int }

func (a *assignable) Assign(o interface{}) bool {
	if o, ok := o.(assignable); ok {
		*a = assignable{o.hidden}
		return true
	}
	return false
}

type nonAssignable struct{ hidden int }

func TestCopy(t *testing.T) {
	ctx := log.Testing(t)
	type StructB struct {
		F float32
		P *StructB
	}
	type StructA struct {
		I int
		B bool
		T string
		P *StructB
		R *StructB
		M map[int]StructB
		S []bool
		G interface{}
		A assignable
		N nonAssignable
	}

	cyclic := &StructB{F: 10}
	cyclic.P = cyclic

	for i, test := range []struct {
		dst, src, expect interface{}
	}{
		{&StructA{}, StructA{}, StructA{}},
		{&StructA{}, StructA{I: 10}, StructA{I: 10}},
		{&StructA{I: 20}, StructA{I: 10}, StructA{I: 10}},
		{&StructA{}, StructA{I: 10, B: true, T: "meow"}, StructA{I: 10, B: true, T: "meow"}},
		{&StructA{}, StructA{A: assignable{5}}, StructA{A: assignable{5}}},
		{&StructA{}, StructA{N: nonAssignable{5}}, StructA{N: nonAssignable{0}}},
		{
			&StructA{},
			StructA{
				I: 10, B: true, T: "meow", P: &StructB{F: 123.456},
			},
			StructA{
				I: 10, B: true, T: "meow", P: &StructB{F: 123.456},
			},
		}, {
			&StructA{},
			StructA{
				I: 10, B: true, T: "meow", P: cyclic,
			},
			StructA{
				I: 10, B: true, T: "meow", P: cyclic,
			},
		}, {
			&StructA{},
			StructA{
				I: 10, B: true, T: "meow", P: cyclic, R: cyclic,
			},
			StructA{
				I: 10, B: true, T: "meow", P: cyclic, R: cyclic,
			},
		}, {
			&StructA{},
			struct{ G string }{"purr"},
			StructA{G: "purr"},
		}, {
			&StructA{},
			struct{ P interface{} }{cyclic},
			StructA{P: cyclic},
		},
	} {
		ctx := log.V{"src": test.src}.Bind(ctx)
		err := deep.Copy(test.dst, test.src)
		if assert.For(ctx, "err").ThatError(err).Succeeded() {
			got := reflect.ValueOf(test.dst).Elem().Interface()
			assert.For(ctx, "Test %v", i).That(got).DeepEquals(test.expect)
		}
	}
}
