// Copyright (C) 2018 Google Inc.
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

package dependencygraph2

import (
	"context"
	"math/bits"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
)

type GraphBuilder interface {
	AddDependencies(map[NodeID][]FragmentAccess,
		map[NodeID][]MemoryAccess,
		map[NodeID][]ForwardAccess)
	AddNodeDependencies(NodeID, []FragmentAccess, []MemoryAccess, []ForwardAccess)
	BuildReverseDependencies()
	GetCmdNodeID(api.CmdID) NodeID
	GetObsNodeIDs(api.CmdID, []api.CmdObservation, bool) []NodeID
	GetCmdContext(api.CmdID, api.Cmd) CmdContext
	GetNodeStats(NodeID) *NodeStats
	GetStats() *GraphBuilderStats
	GetGraph() *dependencyGraph
}

type GraphBuilderStats struct {
	UniqueFragReads     uint64
	UniqueFragWrites    uint64
	UniqueMemReads      uint64
	UniqueMemWrites     uint64
	UniqueDeps          uint64
	NumDeps             uint64
	NumFragDeps         uint64
	NumCompleteFragDeps uint64
	NumMemDeps          uint64
	NumCmdNodes         uint64
	NumObsNodes         uint64
	DepDist             Distribution
}

type graphBuilder struct {
	pendingNodes []NodeID
	nodeStats    []*NodeStats
	graph        *dependencyGraph
	stats        GraphBuilderStats
	isDep        []bool
	depSlice     []NodeID
}

func NewGraphBuilder(ctx context.Context, config DependencyGraphConfig,
	c *capture.Capture, initialCmds []api.Cmd) *graphBuilder {
	return &graphBuilder{
		graph: newDependencyGraph(ctx, config, c, initialCmds, []Node{}),
	}
}

func (b *graphBuilder) AddDependencies(
	fragAcc map[NodeID][]FragmentAccess,
	memAcc map[NodeID][]MemoryAccess,
	forwardAcc map[NodeID][]ForwardAccess) {
	for _, n := range b.pendingNodes {
		b.AddNodeDependencies(n, fragAcc[n], memAcc[n], forwardAcc[n])
	}
	b.pendingNodes = b.pendingNodes[:0]
}
func (b *graphBuilder) AddNodeDependencies(nodeID NodeID,
	fragAccesses []FragmentAccess,
	memAccesses []MemoryAccess,
	forwardAccesses []ForwardAccess) {
	if len(b.isDep) < len(b.graph.nodes) {
		n := uint(len(b.graph.nodes))
		b.isDep = make([]bool, 1<<uint(bits.Len(n-1)))
	}
	isDep := b.isDep
	depSlice := b.depSlice
	stats := b.nodeStats[nodeID]
	for _, a := range fragAccesses {
		if a.Mode&ACCESS_READ != 0 {
			stats.UniqueFragReads++
		}
		if a.Mode&ACCESS_WRITE != 0 {
			stats.UniqueFragWrites++
		}
		stats.NumDeps += uint64(len(a.Deps))
		stats.NumFragDeps += uint64(len(a.Deps))
		if _, ok := a.Fragment.(api.CompleteFragment); ok {
			stats.NumCompleteFragDeps += uint64(len(a.Deps))
		}
		for _, d := range a.Deps {
			if !isDep[d] {
				isDep[d] = true
				depSlice = append(depSlice, d)
			}
		}
	}
	for _, a := range memAccesses {
		if a.Mode&ACCESS_READ != 0 {
			stats.UniqueMemReads++
		}
		if a.Mode&ACCESS_WRITE != 0 {
			stats.UniqueMemWrites++
		}
		stats.NumDeps += uint64(len(a.Deps))
		stats.NumMemDeps += uint64(len(a.Deps))
		for _, d := range a.Deps {
			if !isDep[d] {
				isDep[d] = true
				depSlice = append(depSlice, d)
			}
		}
	}
	openForwardDeps := 0
	for _, a := range forwardAccesses {
		if a.Mode == FORWARD_OPEN {
			stats.NumDeps++
			d := a.Nodes.Close
			if d == NodeNoID || d > nodeID {
				// Forward dep going to later node.
				// Dependency will be added when processing the CLOSE
				openForwardDeps++
			} else {
				// Forward dep actually going to earlier node
				// (possibly from subcommand to parent).
				// Add dependency now, since dependency has already been processed.
				if !isDep[d] {
					isDep[d] = true
					depSlice = append(depSlice, d)
				}
			}
		} else if a.Mode == FORWARD_CLOSE && a.Nodes.Open < a.Nodes.Close {
			// Close is on a later node than open,
			// so dependency hasn't been added yet
			b.graph.addDependency(a.Nodes.Open, a.Nodes.Close)
		}
	}

	newDepSlice := make([]NodeID, len(depSlice), len(depSlice)+openForwardDeps)
	for i, d := range depSlice {
		isDep[d] = false
		newDepSlice[i] = depSlice[i]
	}
	b.depSlice = depSlice[:0]

	b.graph.setDependencies(nodeID, newDepSlice)

	stats.UniqueDeps = uint64(cap(newDepSlice))
	b.stats.DepDist.Add(stats.UniqueDeps)
	b.stats.NumDeps += stats.NumDeps
	b.stats.NumFragDeps += stats.NumFragDeps
	b.stats.NumCompleteFragDeps += stats.NumCompleteFragDeps
	b.stats.NumMemDeps += stats.NumMemDeps
	b.stats.UniqueDeps += stats.UniqueDeps
	b.stats.UniqueFragReads += stats.UniqueFragReads
	b.stats.UniqueFragWrites += stats.UniqueFragWrites
	b.stats.UniqueMemReads += stats.UniqueMemReads
	b.stats.UniqueMemWrites += stats.UniqueMemWrites
}

func (b *graphBuilder) GetCmdNodeID(cmdID api.CmdID) NodeID {
	nodeID := b.graph.GetCmdNodeID(cmdID, api.SubCmdIdx{})
	if nodeID != NodeNoID {
		return nodeID
	}
	fullIdx := api.SubCmdIdx{(uint64)(cmdID)}
	node := CmdNode{fullIdx}
	return b.addNode(node)
}

func (b *graphBuilder) GetObsNodeIDs(cmdID api.CmdID, obs []api.CmdObservation, isWrite bool) []NodeID {
	nodeIDs := make([]NodeID, len(obs))
	for i, o := range obs {
		nodeIDs[i] = b.addNode(ObsNode{
			CmdObservation: o,
			CmdID:          cmdID,
			IsWrite:        isWrite,
			Index:          i,
		})
	}
	return nodeIDs
}

func (b *graphBuilder) GetCmdContext(cmdID api.CmdID, cmd api.Cmd) CmdContext {
	nodeID := b.GetCmdNodeID(cmdID)
	stats := b.nodeStats[nodeID]
	return CmdContext{cmdID, cmd, nodeID, stats}
}

func (b *graphBuilder) GetNodeStats(nodeID NodeID) *NodeStats {
	return b.nodeStats[nodeID]
}

func (b *graphBuilder) GetStats() *GraphBuilderStats {
	return &b.stats
}

func (b *graphBuilder) GetGraph() *dependencyGraph {
	return b.graph
}

func (b *graphBuilder) BuildReverseDependencies() {
	b.graph.buildDependenciesTo()
}

func (b *graphBuilder) addNode(node Node) NodeID {
	if _, ok := node.(CmdNode); ok {
		b.stats.NumCmdNodes++
	}
	if _, ok := node.(ObsNode); ok {
		b.stats.NumObsNodes++
	}
	nodeID := b.graph.addNode(node)
	b.pendingNodes = append(b.pendingNodes, nodeID)
	newNodeStats := append(b.nodeStats, make([]*NodeStats, int(nodeID)+1-len(b.nodeStats))...)
	for i := len(b.nodeStats); i < len(newNodeStats); i++ {
		newNodeStats[i] = &NodeStats{}
	}
	b.nodeStats = newNodeStats
	return nodeID
}
