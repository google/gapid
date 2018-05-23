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

package sint

import (
	"math"
	"sort"
)

// Histogram represents a series of integer values, for the purpose of
// computing statistics about them.
type Histogram []int

// HistogramStats stores the result of computing statistics on a Histogram
type HistogramStats struct {
	// Average is the mean of all the values in the Histogram.
	Average float64
	// Stddev is the population standard deviation of the values in the
	// Histogram.
	Stddev float64
	// Median is the median value of the values in the Histogram.
	Median int
}

// Add adds `count` to the Histogram at position `at`, zero-extending the
// Histogram if necessary.
func (h *Histogram) Add(at, count int) {
	if at < 0 {
		return
	}
	if at >= cap(*h) { // at exceeds slice capacity, reallocate.
		newCap := Max(at*2, 32)
		n := make(Histogram, at+1, newCap)
		copy(n, *h)
		*h = n
	} else if at >= len(*h) { // at exceeds slice length, reslice.
		*h = (*h)[:at+1]
	}
	(*h)[at] += count
}

// Stats computes average, standard deviation, and median statistics on the
// values in the Histogram.  See HistogramStats for more details.
func (h *Histogram) Stats() HistogramStats {
	if len(*h) == 0 {
		return HistogramStats{}
	}
	total, total2 := 0, 0
	for _, v := range *h {
		total += v
		total2 += v * v
	}

	n := float64(len(*h))
	avg, avg2 := float64(total)/n, float64(total2)/n

	sorted := append([]int{}, (*h)...)
	sort.Ints(sorted)

	return HistogramStats{
		Average: avg,
		Stddev:  math.Sqrt(avg2 - avg*avg),
		Median:  sorted[len(sorted)/2],
	}
}
