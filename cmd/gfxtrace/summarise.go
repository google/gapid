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
	"context"
	"fmt"
	"reflect"

	"sort"

	"strings"

	"github.com/google/gapid/core/data/compare"
	"github.com/google/gapid/core/log"
	_ "github.com/google/gapid/gapis/gfxapi/all"
)

type (
	Summary       map[reflect.Type]int
	SummaryRecord struct {
		Name  string
		Count int
	}
	SummaryList []SummaryRecord
)

func (s Summary) Compute(l interface{}) Summary {
	v := reflect.ValueOf(l)
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			t := reflect.TypeOf(v.Index(i).Interface())
			s[t] = s[t] + 1
		}
	} else {
		t := v.Type()
		s[t] = s[t] + 1
	}
	return s
}

func (s Summary) List() SummaryList {
	result := SummaryList{}
	for t, count := range s {
		result = append(result, SummaryRecord{Name: t.String(), Count: count})
	}
	sort.Sort(result)
	return result
}

func (s Summary) Format(f fmt.State, c rune) {
	s.List().Format(f, c)
}

//func (l SummaryList) Less(i, j int) bool { return l[i].Count > l[j].Count }
func (l SummaryList) Len() int           { return len(l) }
func (l SummaryList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l SummaryList) Less(i, j int) bool { return strings.Compare(l[i].Name, l[j].Name) < 0 }
func (l SummaryList) Format(f fmt.State, c rune) {
	for _, r := range l {
		fmt.Fprintf(f, "%s : %d\n", r.Name, r.Count)
	}
}

func SummaryDiff(ctx context.Context, a, b SummaryList) {
	for {
		if len(a) == 0 {
			if len(b) == 0 {
				return
			}
			log.W(ctx, "Extra %s : %d", b[0].Name, b[0].Count)
			b = b[1:]
			continue
		}
		if len(b) == 0 {
			log.W(ctx, "Missing %s : %d", a[0].Name, a[0].Count)
			a = a[1:]
			continue
		}
		match := strings.Compare(a[0].Name, b[0].Name)
		switch {
		case match == 0:
			if a[0].Count != b[0].Count {
				log.W(ctx, "Different %s : in %d out %d", a[0].Name, a[0].Count, b[0].Count)
			}
			a = a[1:]
			b = b[1:]
		case match < 0:
			log.W(ctx, "Missing %s : %d", a[0].Name, a[0].Count)
			a = a[1:]
		default:
			log.W(ctx, "Extra %s : %d", b[0].Name, b[0].Count)
			b = b[1:]
		}
	}
}

func AtomDiff(ctx context.Context, in, out interface{}) {
	for i, d := range compare.Diff(in, out, 100) {
		log.I(ctx, "# %v : %v", i, d)
	}
}
