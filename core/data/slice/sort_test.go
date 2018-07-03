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

package slice_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/log"
)

type S struct{ I int }

func (s S) String() string { return fmt.Sprint(s.I) }
func TestSort(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []struct {
		unsorted interface{}
		expected interface{}
	}{
		{[]int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4, 5}},
		{[]int{5, 4, 3, 2, 1}, []int{1, 2, 3, 4, 5}},
		{[]int8{1, 2, 3, 4, 5}, []int8{1, 2, 3, 4, 5}},
		{[]int8{5, 4, 3, 2, 1}, []int8{1, 2, 3, 4, 5}},
		{[]int16{1, 2, 3, 4, 5}, []int16{1, 2, 3, 4, 5}},
		{[]int16{5, 4, 3, 2, 1}, []int16{1, 2, 3, 4, 5}},
		{[]int32{1, 2, 3, 4, 5}, []int32{1, 2, 3, 4, 5}},
		{[]int32{5, 4, 3, 2, 1}, []int32{1, 2, 3, 4, 5}},
		{[]int64{1, 2, 3, 4, 5}, []int64{1, 2, 3, 4, 5}},
		{[]int64{5, 4, 3, 2, 1}, []int64{1, 2, 3, 4, 5}},

		{[]uint{1, 2, 3, 4, 5}, []uint{1, 2, 3, 4, 5}},
		{[]uint{5, 4, 3, 2, 1}, []uint{1, 2, 3, 4, 5}},
		{[]uint8{1, 2, 3, 4, 5}, []uint8{1, 2, 3, 4, 5}},
		{[]uint8{5, 4, 3, 2, 1}, []uint8{1, 2, 3, 4, 5}},
		{[]uint16{1, 2, 3, 4, 5}, []uint16{1, 2, 3, 4, 5}},
		{[]uint16{5, 4, 3, 2, 1}, []uint16{1, 2, 3, 4, 5}},
		{[]uint32{1, 2, 3, 4, 5}, []uint32{1, 2, 3, 4, 5}},
		{[]uint32{5, 4, 3, 2, 1}, []uint32{1, 2, 3, 4, 5}},
		{[]uint64{1, 2, 3, 4, 5}, []uint64{1, 2, 3, 4, 5}},
		{[]uint64{5, 4, 3, 2, 1}, []uint64{1, 2, 3, 4, 5}},

		{[]float32{5, 4, 3, 2, 1}, []float32{1, 2, 3, 4, 5}},
		{[]float64{1, 2, 3, 4, 5}, []float64{1, 2, 3, 4, 5}},

		{[]bool{false, false, true, true}, []bool{false, false, true, true}},
		{[]bool{false, true, false, true}, []bool{false, false, true, true}},

		{[]rune{'a', 'b', 'c', 'd', 'e'}, []rune{'a', 'b', 'c', 'd', 'e'}},
		{[]rune{'e', 'd', 'c', 'b', 'a'}, []rune{'a', 'b', 'c', 'd', 'e'}},

		{[]string{"a", "b", "c", "d", "e"}, []string{"a", "b", "c", "d", "e"}},
		{[]string{"e", "d", "c", "b", "a"}, []string{"a", "b", "c", "d", "e"}},

		{[]S{{1}, {2}, {10}, {11}}, []S{{1}, {10}, {11}, {2}}},
		{[]S{{11}, {10}, {2}, {1}}, []S{{1}, {10}, {11}, {2}}},
	} {
		{ // Sort()
			s := slice.Clone(test.unsorted)
			slice.Sort(s)
			assert.For(ctx, "Sort(%v)", test.unsorted).
				That(s).DeepEquals(test.expected)
		}
		{ // SortValues()
			unsorted := reflect.ValueOf(test.unsorted)
			s := make([]reflect.Value, unsorted.Len())
			for i := range s {
				s[i] = unsorted.Index(i)
			}
			slice.SortValues(s, unsorted.Type().Elem())
			sorted := slice.New(unsorted.Type(), len(s), len(s))
			for i := range s {
				sorted.Index(i).Set(s[i])
			}
			assert.For(ctx, "SortValues(%v, %v)", test.unsorted, unsorted.Type().Elem()).
				That(sorted.Interface()).DeepEquals(test.expected)
		}
	}
}
