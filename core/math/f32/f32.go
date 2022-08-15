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

package f32

import "math"

// MinOf returns the minimum value of all the arguments.
func MinOf(a float32, b ...float32) float32 {
	v := a
	for _, x := range b {
		if x < v {
			v = x
		}
	}
	return v
}

// MaxOf returns the maximum value of all the arguments.
func MaxOf(a float32, b ...float32) float32 {
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
//
//	Round(-0.9) = -1
//	Round(-0.1) = 0
//	Round(0.0) = 0
//	Round(0.1) = 0
//	Round(0.9) = 1
func Round(v float32) int {
	if v < 0 {
		return int(math.Ceil(float64(v) - 0.5))
	}
	return int(math.Floor(float64(v) + 0.5))
}

// Sqrt returns the square root of v.
func Sqrt(v float32) float32 {
	return float32(math.Sqrt(float64(v)))
}

// Abs returns the absolute of v.
func Abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
