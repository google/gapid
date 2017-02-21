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

package file

// PathSet is a list of file paths that does not allow duplicates.
type PathSet struct {
	values PathList
}

// AsList returns the contents of the PathSet as a List.
func (s PathSet) AsList() PathList {
	return s.values
}

// Append is analogous to the append function, except it also suppresses duplicates.
// It copies the set and then adds the supplied paths.
func (s PathSet) Append(paths ...Path) PathSet {
	result := PathSet{}
	result.values = make(PathList, len(s.values), len(s.values)+len(paths))
	copy(result.values, s.values)
	for _, path := range paths {
		if !result.Contains(path) {
			result.values = append(result.values, path)
		}
	}
	return result
}

// Union produces the union of this path set with another.
func (s PathSet) Union(other PathSet) PathSet {
	return s.Append(other.values...)
}

// Contains tests to see if the set contains the path.
func (s PathSet) Contains(path Path) bool {
	return s.values.Contains(path)
}

// RootOf returns the first Path that contains the path, or an empty path if not found.
func (s PathSet) RootOf(p Path) Rooted {
	return s.values.RootOf(p)
}

// Matching returns the set of paths that match any pattern.
func (s PathSet) Matching(patterns ...string) PathSet {
	return PathSet{s.values.Matching(patterns...)}
}

// NotMatching returns the set of paths that do not match any pattern.
func (s PathSet) NotMatching(patterns ...string) PathSet {
	return PathSet{s.values.NotMatching(patterns...)}
}
