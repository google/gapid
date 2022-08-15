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

package api

import (
	"sort"
)

// SubCmdIdx is a qualified path from a particular index to a given subcommand.
type SubCmdIdx []uint64

// LessThan returns true if s comes before s2.
func (s SubCmdIdx) LessThan(s2 SubCmdIdx) bool {
	for i := range s {
		if i > len(s2)-1 {
			// This case is a bit weird, but
			// {0} > {0, 1}, since {0} represents
			// the ALL commands under 0.
			return true
		}
		if s[i] < s2[i] {
			return true
		}
		if s[i] > s2[i] {
			return false
		}
	}
	return false
}

// LEQ returns true if s comes before s2.
func (s SubCmdIdx) LEQ(s2 SubCmdIdx) bool {
	for i := range s {
		if i > len(s2)-1 {
			// This case is a bit weird, but
			// {0} > {0, 1}, since {0} represents
			// the ALL commands under 0.
			return true
		}
		if s[i] < s2[i] {
			return true
		}
		if s[i] > s2[i] {
			return false
		}
	}
	return true
}

// Equals returns true if both sets of subcommand indices are the same.
func (s SubCmdIdx) Equals(s2 SubCmdIdx) bool {
	if len(s) != len(s2) {
		return false
	}
	for i := range s {
		if s[i] != s2[i] {
			return false
		}
	}
	return true
}

// Decrement returns the subcommand that preceded this subcommand.
// Decrement will decrement its way UP subcommand chains.
// Eg: {0, 1}.Decrement() == {0, 0}
//
//	{1, 0}.Decrement() == {0}
//	{0}.Decrement() == {}
func (s *SubCmdIdx) Decrement() {
	for len(*s) > 0 {
		if (*s)[len(*s)-1] > 0 {
			(*s)[len(*s)-1]--
			return
		}
		*s = (*s)[:len(*s)-1]
	}
}

// Contains returns true if s is one of the parent nodes of s2 or equals to s2.
func (s SubCmdIdx) Contains(s2 SubCmdIdx) bool {
	return len(s2) >= len(s) && len(s) != 0 && s.Equals(s2[:len(s)])
}

func (s SubCmdIdx) InRange(begin SubCmdIdx, end SubCmdIdx) bool {
	if s.LessThan(begin) {
		return false
	}

	if end.LessThan(s) {
		return false
	}

	return true
}

// SortSubCmdIDs sorts the slice of subcommand ids
func SortSubCmdIDs(ids []SubCmdIdx) {
	lessFunc := func(i, j int) bool {
		return ids[i].LessThan(ids[j])
	}
	sort.Slice(ids, lessFunc)
}

// ReverseSubCmdIDs reverses the slice of subcommand ids
func ReverseSubCmdIDs(ids []SubCmdIdx) {
	size := len(ids)
	for i := 0; i < size/2; i++ {
		ids[i], ids[size-1-i] = ids[size-1-i], ids[i]
	}
}
