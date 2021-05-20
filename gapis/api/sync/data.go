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
	Index api.SubCmdIdx
	// If api.CmdID is api.CmdNoID then the generating command came from before
	// the start of the trace.
	GeneratingCmd api.CmdID
	// If GeneratingCmd is nil, then MidExecutionCommandData contains the data
	// that the API needs in order to reconstruct this command.
	MidExecutionCommandData interface{}
}

// SyncNodeIdx is the identifier for a node in the sync dependency graph.
type SyncNodeIdx uint64

// SyncNode is the interface implemented by types that can be used as vertices
// in the sync dependency graph.
type SyncNode interface {
	isSyncNode()
}

// CmdNode is a node in the sync dependency graph that is a command.
type CmdNode struct {
	Idx api.SubCmdIdx
}

// AbstractNode is a node in the sync dependency graph that doesn't correspond
// to any point in the trace and is just used as a marker.
type AbstractNode struct{}

var (
	_ = SyncNode(CmdNode{})
	_ = SyncNode(AbstractNode{})
)

// Data contains a map of synchronization pairs.
type Data struct {
	// SubcommandReferences contains the information about every subcommand
	// run by a particular command.
	SubcommandReferences map[api.CmdID][]SubcommandReference
	// SubcommandGroups represents the next utilizable subcommand index in
	// every command buffer.
	SubcommandGroups map[api.CmdID][]api.SubCmdIdx
	// Hidden contains all the commands that should be hidden from the regular
	// command tree as they exist as a subcommand of another command.
	Hidden api.CmdIDSet
	// SubCommandMarkerGroups contains all the marker groups in the subcommands,
	// indexed by the immediate parent of the subcommands in the group.
	// e.g.: group: [73, 1, 4, 5~6] should be indexed by [73, 1, 4]
	SubCommandMarkerGroups *subCommandMarkerGroupTrie
	// SyncDependencies contains the commands that must complete
	// (according to their fences or semaphores) before they can be executed.
	SyncDependencies map[SyncNodeIdx][]SyncNodeIdx
	SyncNodes        []SyncNode
	CmdSyncNodes     map[api.CmdID]SyncNodeIdx
	// SubcommandLookup maps a SubCmdIdx to its corresponding SubcommandReference.
	SubcommandLookup *api.SubCmdIdxTrie
	// SubcommandNames maps a SubCmdIdx to its corresponding string typed name.
	// The names are especially useful for the virtual SubCmdRoot nodes, which are
	// created to organize psubmits, command buffers, etc.
	SubcommandNames   *api.SubCmdIdxTrie
	SubmissionIndices map[api.CmdSubmissionKey][]api.SubCmdIdx
}

type subCommandMarkerGroupTrie struct {
	api.SubCmdIdxTrie
}

// NewMarkerGroup creates a new CmdIDGroup for the marker group in the marker
// group trie with the specified name and parent SubCmdIdx, and returns a
// pointer to the created CmdIDGroup.
func (t *subCommandMarkerGroupTrie) NewMarkerGroup(parent api.SubCmdIdx, name string, start, end uint64, experimentalCmds []api.SubCmdIdx) *api.CmdIDGroup {
	l := []*api.CmdIDGroup{}
	if o, ok := t.Value(parent).([]*api.CmdIDGroup); ok {
		l = o
	}
	group := &api.CmdIDGroup{Name: name}
	group.Range.Start = api.CmdID(start)
	group.Range.End = api.CmdID(end)
	group.ExperimentableCmds = experimentalCmds
	l = append(l, group)
	t.SetValue(parent, l)
	return group
}

// NewData creates a new clean Data object
func NewData() *Data {
	return &Data{
		SubcommandReferences:   map[api.CmdID][]SubcommandReference{},
		SubcommandGroups:       map[api.CmdID][]api.SubCmdIdx{},
		Hidden:                 api.CmdIDSet{},
		SubCommandMarkerGroups: &subCommandMarkerGroupTrie{},
		SyncDependencies:       map[SyncNodeIdx][]SyncNodeIdx{},
		SyncNodes:              []SyncNode{},
		CmdSyncNodes:           map[api.CmdID]SyncNodeIdx{},
		SubcommandLookup:       new(api.SubCmdIdxTrie),
		SubcommandNames:        new(api.SubCmdIdxTrie),
		SubmissionIndices:      map[api.CmdSubmissionKey][]api.SubCmdIdx{},
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

// SortedKeys returns the keys of 'e' in sorted order
func (e ExecutionRanges) SortedKeys() SynchronizationIndices {
	v := make(SynchronizationIndices, 0, len(e.Ranges))
	for k := range e.Ranges {
		v = append(v, k)
	}
	sort.Sort(v)
	return v
}

func (CmdNode) isSyncNode() {}

func (AbstractNode) isSyncNode() {}
