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

package u32

// Min returns the minimum value of a and b.
func Min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum value of a and b.
func Max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}
