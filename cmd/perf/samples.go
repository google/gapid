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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/gapid/core/app/benchmark"
)

// Sample is a time.Duration with more readable JSON serialization.
type Sample time.Duration

// Multisample represents a collection of AnnotatedSamples gathered through
// multiple runs of a
type Multisample []Sample

// KeyedSamples maps atom indices to Multisamples.
type KeyedSamples map[string]*Multisample

// IndexedMultisample is a Multisample together with an atom index.
type IndexedMultisample struct {
	Index  int64
	Values *Multisample
}

// IndexedMultisamples is a slice of IndexedMultisample, sortable by
// index and usable for plotting or complexity analysis.
type IndexedMultisamples []IndexedMultisample

// Analyse analyses the samples and returns the algorithmic cost.
func (s IndexedMultisamples) Analyse(accessor func(IndexedMultisample) (int, time.Duration)) benchmark.Fit {
	samples := benchmark.Samples{}
	for _, m := range s {
		samples.AddOrUpdate(accessor(m))
	}
	return samples.Analyse()
}

// NewKeyedSamples instantiates a new KeyedSamples.
func NewKeyedSamples() KeyedSamples {
	return make(map[string]*Multisample)
}

// Add records a new duration associated with the given index.
func (m KeyedSamples) Add(index int64, t time.Duration) {
	key := fmt.Sprintf("%010d", index) // Easiest way to have samples show up in sorted order in the json.
	ms, found := m[key]
	if !found {
		ms = new(Multisample)
		m[key] = ms
	}
	ms.Add(t)
}

// IndexedMultisamples returns a sortable array of indexed multisamples.
func (m KeyedSamples) IndexedMultisamples() IndexedMultisamples {
	result := make(IndexedMultisamples, len(m))
	i := 0
	for key, values := range m {
		index, _ := strconv.ParseInt(key, 10, 64)
		result[i] = IndexedMultisample{Index: index, Values: values}
		i++
	}
	sort.Sort(result)
	return result
}

func (s IndexedMultisamples) Len() int { return len(s) }
func (s IndexedMultisamples) Less(i, j int) bool {
	return s[i].Index < s[j].Index
}
func (s IndexedMultisamples) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (a *Sample) Set(t time.Duration) {
	*a = Sample(t)
}

func (a Sample) Duration() time.Duration {
	return time.Duration(a)
}

func (a *Sample) UnmarshalJSON(b []byte) error {
	var durationString string
	if err := json.Unmarshal(b, &durationString); err != nil {
		return err
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return err
	}
	a.Set(duration)
	return nil
}

func (a Sample) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Duration().String())
}

func (s *Multisample) AddWithData(t time.Duration, data string) {
	*s = append(*s, Sample(t))
}

func (s *Multisample) Len() int {
	return len(*s)
}

func (s *Multisample) Clear() {
	*s = make([]Sample, 0)
}

func (s *Multisample) Add(t time.Duration) {
	*s = append(*s, Sample(t))
}

func (s *Multisample) Average() time.Duration {
	sum := int64(0)
	for _, value := range *s {
		sum += int64(value.Duration())
	}
	return time.Duration(sum / int64(len(*s)))
}

func (s *Multisample) Min() time.Duration {
	min := time.Duration(1<<63 - 1)
	for _, value := range *s {
		if value.Duration() < min {
			min = value.Duration()
		}
	}
	return min
}

func (s *Multisample) Max() time.Duration {
	max := time.Duration(-1 << 63)
	for _, value := range *s {
		if value.Duration() > max {
			max = value.Duration()
		}
	}
	return max
}
func (s *Multisample) Median() time.Duration {
	arr := make([]int64, len(*s))
	for i := range arr {
		arr[i] = int64((*s)[i].Duration())
	}

	sort.Sort(int64Slice(arr))
	if len(arr)%2 != 0 {
		return time.Duration(arr[len(arr)/2])
	} else {
		return time.Duration((arr[len(arr)/2-1] + arr[len(arr)/2]) / 2)
	}
}

type int64Slice []int64

func (p int64Slice) Len() int           { return len(p) }
func (p int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
