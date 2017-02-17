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

package f64

import "math"

// Minf returns the minimum value of all the arguments.
func Minf(a float64, b ...float64) float64 {
	v := a
	for _, x := range b {
		if x < v {
			v = x
		}
	}
	return v
}

// Maxf returns the maximum value of all the arguments.
func Maxf(a float64, b ...float64) float64 {
	v := a
	for _, x := range b {
		if x > v {
			v = x
		}
	}
	return v
}

// Round rounds v to the nearest integer.
// Examples:
//   Round(-0.9) = -1
//   Round(-0.1) = 0
//   Round(0.0) = 0
//   Round(0.1) = 0
//   Round(0.9) = 1
func Round(v float64) int {
	if v < 0 {
		return int(math.Ceil(v - 0.5))
	}
	return int(math.Floor(v + 0.5))
}
