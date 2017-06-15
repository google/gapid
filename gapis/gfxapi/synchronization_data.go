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

package gfxapi

import "sort"

// An index, typically an atom ID that defines the location of one side
// synchronization dependency.
type SynchronizationIndex uint64
type SynchronizationIndices []SynchronizationIndex

// The index of a subcommand within a command
type SubcommandIndex []uint64

// ExecutionRanges contains the information about a blocked command.
// LastIndex is the final subcommand that exists within this command.
// Ranges defines which future command will unblock the command in question, and
// which subcommandis the last that will be run at that point.
type ExecutionRanges struct {
	LastIndex SubcommandIndex
	Ranges    map[SynchronizationIndex]SubcommandIndex
}

// SynchronizationData contains a map of synchronization pairs.
// The SynchronizationIndex is the command that will be blocked from
// completion, and what subcommands will be made available by future commands.
type SynchronizationData struct {
	CommandRanges map[SynchronizationIndex]ExecutionRanges
}

// NewSynchronizationData creates a new clean SynchronizationData object
func NewSynchronizationData() *SynchronizationData {
	s := new(SynchronizationData)
	s.CommandRanges = make(map[SynchronizationIndex]ExecutionRanges)
	return s
}

// LessThan returns true if s comes before s2.
func (s SubcommandIndex) LessThan(s2 SubcommandIndex) bool {
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

// Decrement returns the subcommand that preceded this subcommand.
// Decrement will decrement its way UP subcommand chains.
// Eg: {0, 1}.Decrement() == {0, 0}
//     {1, 0}.Decrement() == {0}
//     {0}.Decrement() == {}
func (s *SubcommandIndex) Decrement() {
	for len(*s) > 0 {
		if (*s)[len(*s)-1] > 0 {
			(*s)[len(*s)-1]--
			return
		}
		*s = (*s)[:len(*s)-1]
	}
}

// Len returns the length of subcommand indices
func (s SynchronizationIndices) Len() int {
	return len(s)
}

// Swap swaps the 2 subcommands in the given slice
func (s SynchronizationIndices) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns true if s[i] < s[j]
func (s SynchronizationIndices) Less(i, j int) bool {
	return s[i] < s[j]
}

// SortedKeys returns the keys of 's' in sorted order
func (s SynchronizationData) SortedKeys() SynchronizationIndices {
	v := make(SynchronizationIndices, 0, len(s.CommandRanges))
	for k, _ := range s.CommandRanges {
		v = append(v, k)
	}
	sort.Sort(v)
	return v
}

// SortedKeys returns the keys of 'e' in sorted order
func (e ExecutionRanges) SortedKeys() SynchronizationIndices {
	v := make(SynchronizationIndices, 0, len(e.Ranges))
	for k, _ := range e.Ranges {
		v = append(v, k)
	}
	sort.Sort(v)
	return v
}
