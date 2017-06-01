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

package sint

func Abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func Clamp(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// Min returns the minimum value of a and b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MinOf returns the minimum value of all the arguments.
func MinOf(a int, b ...int) int {
	v := a
	for _, x := range b {
		if x < v {
			v = x
		}
	}
	return v
}

// Max returns the maximum value of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MaxOf returns the maximum value of all the arguments.
func MaxOf(a int, b ...int) int {
	v := a
	for _, x := range b {
		if x > v {
			v = x
		}
	}
	return v
}

// Log10 returns the log10(i), rounded down to a whole integer.
// i must be positive.
func Log10(i int) int {
	if i < 0 {
		panic("i must be positive")
	}
	o := 0
	for {
		i /= 10
		if i == 0 {
			return o
		}
		o++
	}
}

func Byte(i int) byte {
	if i < 0 {
		return 0
	}
	if i > 255 {
		return 255
	}
	return byte(i)
}

func AlignUp(v int, alignment int) int {
	return alignment * ((v + (alignment - 1)) / alignment)
}
