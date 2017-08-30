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

package sync

import (
	"sort"

	"github.com/google/gapid/gapis/api"
)

// SynchronizationIndices is a list of command identifiers, defining the
// location of a one side synchronization dependency.
type SynchronizationIndices []api.CmdID

// ExecutionRanges contains the information about a blocked command.
type ExecutionRanges struct {
	// LastIndex is the final subcommand that exists within this command.
	LastIndex api.SubCmdIdx
	// Ranges defines which future command will unblock the command in question, and
	// which subcommand is the last that will be run at that point.
	Ranges map[api.CmdID]api.SubCmdIdx
}

// SubcommandReference contains a subcommand index as well as an api.CmdID that
// references the command that generated this subcommand.
type SubcommandReference struct {
	Index         api.SubCmdIdx
	GeneratingCmd api.CmdID
	// IsCalledGroup is true if the reference is to a nested call, otherwise
	// the reference belongs to a command-list.
	IsCallerGroup bool
}

// Data contains a map of synchronization pairs.
type Data struct {
	// CommandRanges contains commands that will be blocked from completion,
	// and what subcommands will be made available by future commands.
	CommandRanges map[api.CmdID]ExecutionRanges
	// SubcommandReferences contains the information about every subcommand
	// run by a particular command.
	SubcommandReferences map[api.CmdID][]SubcommandReference
	// SubcommandGroups represents the last Subcommand in every command buffer.
	SubcommandGroups map[api.CmdID][]api.SubCmdIdx
	// Hidden contains all the commands that should be hidden from the regular
	// command tree as they exist as a subcommand of another command.
	Hidden api.CmdIDSet
}

// NewData creates a new clean Data object
func NewData() *Data {
	return &Data{
		CommandRanges:        map[api.CmdID]ExecutionRanges{},
		SubcommandReferences: map[api.CmdID][]SubcommandReference{},
		SubcommandGroups:     map[api.CmdID][]api.SubCmdIdx{},
		Hidden:               api.CmdIDSet{},
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
func (s Data) SortedKeys() SynchronizationIndices {
	v := make(SynchronizationIndices, 0, len(s.CommandRanges))
	for k := range s.CommandRanges {
		v = append(v, k)
	}
	sort.Sort(v)
	return v
}

// SortedKeys returns the keys of 'e' in sorted order
func (e ExecutionRanges) SortedKeys() SynchronizationIndices {
	v := make(SynchronizationIndices, 0, len(e.Ranges))
	for k := range e.Ranges {
		v = append(v, k)
	}
	sort.Sort(v)
	return v
}
