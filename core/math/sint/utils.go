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

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MinOf(a int, b ...int) int {
	v := a
	for _, x := range b {
		if x < v {
			v = x
		}
	}
	return v
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MaxOf(a int, b ...int) int {
	v := a
	for _, x := range b {
		if x > v {
			v = x
		}
	}
	return v
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
