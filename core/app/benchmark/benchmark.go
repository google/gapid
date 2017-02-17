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

package benchmark

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Sample represents a single benchmark sample.
type Sample struct {
	Index int           // Index of the sample
	Time  time.Duration // Duration of the sample.
}

// Samples is a list of samples
type Samples []Sample

// Add appends the sample to the list.
func (s *Samples) Add(index int, duration time.Duration) {
	*s = append(*s, Sample{index, duration})
}

// Add appends the sample to the list. If there already is a sample
// with that index, overwrite the duration for it.
func (s *Samples) AddOrUpdate(index int, duration time.Duration) {
	if len(*s) != 0 && (*s)[len(*s)-1].Index == index {
		(*s)[len(*s)-1].Time = duration
	} else {
		*s = append(*s, Sample{index, duration})
	}
}

// Len is the number of elements in the collection.
func (s Samples) Len() int { return len(s) }

// Less reports whether the element with
// index i should sort before the element with index j.
func (s Samples) Less(i, j int) bool { return s[i].Index < s[j].Index }

// Swap swaps the elements with indexes i and j.
func (s Samples) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Analyse analyses the samples and returns the algorithmic cost.
func (s Samples) Analyse() Fit {
	sort.Sort(s)
	var best Fit
	var bestErr = math.MaxFloat64
	for _, c := range complexities {
		fit, err := c.Fit(s)
		if err < bestErr {
			best = fit
		}
	}
	return best
}

func (s Samples) String() string {
	sort.Sort(s)
	parts := make([]string, len(s))
	for i, s := range s {
		parts[i] = fmt.Sprintf("%d: %v", s.Index, s.Time)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

var complexities = []Complexity{
	linearTime{},
}
