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
	"fmt"
	"math"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
)

// Node represents a node in the dependency graph, and holds data about the
// associated command or memory observation.
type Node interface {
	dependencyNode()
}

// CmdNode is a dependency node corresponding to an API call
type CmdNode struct {
	Index api.SubCmdIdx
}

func (CmdNode) dependencyNode() {}

// ObsNode is a dependency node corresponding to a memory observation
type ObsNode struct {
	CmdObservation api.CmdObservation
	CmdID          api.CmdID
	IsWrite        bool
	Index          int
}

func (ObsNode) dependencyNode() {}

// Information about what sort of data to store in a dependency graph
type DependencyGraphConfig struct {
	// MergeSubCmdNodes indicates whether the graph should have one node per
	// command (true), or a separate node for each subcommand (false)
	MergeSubCmdNodes bool

	// IncludeInitialCommands indicates whether nodes should be created for
	// the initial (state rebuild) commands
	IncludeInitialCommands bool

	// ReverseDependencies indicates whether reverse edges should be created
	ReverseDependencies bool
}

// NodeID identifies a node in a dependency graph
type NodeID uint32

const NodeNoID = NodeID(math.MaxUint32)

// DependencyGraph stores the dependencies among api calls and memory observations,
type DependencyGraph interface {

	// NumNodes returns the number of nodes in the graph
	NumNodes() int

	// NumDependencies returns the number of dependencies (edges) in the graph
	NumDependencies() uint64

	// GetNode returns the node data associated with the given NodeID
	GetNode(NodeID) Node

	// GetNodeID returns the NodeID associated with given node data
	GetNodeID(Node) NodeID

	// ForeachCmd iterates over all API calls in the graph
	ForeachCmd(ctx context.Context, cb func(context.Context, api.CmdID, api.Cmd) error) error

	// ForeachNode iterates over all nodes in the graph in chronological order.
	// I.e., the following order:
	//   * For each initial command
	//     * Read observation nodes for this command
	//     * command node
	//     * Write observation nodes for this command
	//   * For each (non-initial) command
	//     * Read observation nodes for this command
	//     * command node
	//     * Write observation nodes for this command
	ForeachNode(cb func(NodeID, Node) error) error

	// ForeachDependency iterates over all pairs (src, tgt), where src depends on tgt
	ForeachDependency(cb func(NodeID, NodeID) error) error

	// ForeachDependencyFrom iterates over all the nodes tgt, where src depends on tgt
	ForeachDependencyFrom(src NodeID, cb func(NodeID) error) error

	// ForeachDependencyTo iterates over all the nodes src, where src depends on tgt.
	// If Config().ReverseDependencies is false, this will return an error.
	ForeachDependencyTo(tgt NodeID, cb func(NodeID) error) error

	// Capture returns the capture whose dependencies are stored in this graph
	Capture() *capture.Capture

	// GetCommand returns the command identified by the given CmdID
	GetCommand(api.CmdID) api.Cmd

	// NumInitialCommands returns the number of initial commands
	// (the commands needed to reconstruct the initial state before the
	// first command in the capture)
	NumInitialCommands() int

	// Config returns the config used to create this graph
	Config() DependencyGraphConfig
}

type obsNodeIDs struct {
	readNodeIDStart  NodeID
	writeNodeIDStart NodeID
}

type dependencyGraph struct {
	capture           *capture.Capture
	cmdNodeIDs        *api.SubCmdIdxTrie
	initMemoryNodeIDs obsNodeIDs
	initCmdObsNodeIDs []obsNodeIDs
	cmdObsNodeIDs     []obsNodeIDs
	initialCommands   []api.Cmd
	nodes             []Node
	numDependencies   uint64
	dependenciesFrom  [][]NodeID
	dependenciesTo    [][]NodeID

	config DependencyGraphConfig
}

// newDependencyGraph constructs a new dependency graph
func newDependencyGraph(ctx context.Context, config DependencyGraphConfig,
	c *capture.Capture, initialCmds []api.Cmd) *dependencyGraph {
	g := &dependencyGraph{
		capture:         c,
		cmdNodeIDs:      new(api.SubCmdIdxTrie),
		initialCommands: initialCmds,
		config:          config,
	}
	numCmds := len(initialCmds) + len(c.Commands)
	numObservations := countObservations(initialCmds) + countObservations(c.Commands)
	numNodes := numCmds + numObservations
	g.nodes = make([]Node, 0, numNodes)
	g.initCmdObsNodeIDs = make([]obsNodeIDs, len(initialCmds))
	for i, cmd := range initialCmds {
		g.initCmdObsNodeIDs[i] = g.addCmdNodes(api.CmdID(i).Derived(), cmd.Extras().Observations())
	}
	g.cmdObsNodeIDs = make([]obsNodeIDs, len(c.Commands))
	for i, cmd := range c.Commands {
		g.cmdObsNodeIDs[i] = g.addCmdNodes(api.CmdID(i), cmd.Extras().Observations())
	}
	g.dependenciesFrom = make([][]NodeID, numNodes)
	return g
}

// NumNodes returns the number of nodes in the graph
func (g *dependencyGraph) NumNodes() int {
	return len(g.nodes)
}

// NumDependencies returns the number of dependencies (edges) in the graph
func (g *dependencyGraph) NumDependencies() uint64 {
	return g.numDependencies
}

// GetNode returns the node data associated with the given NodeID
func (g *dependencyGraph) GetNode(nodeID NodeID) Node {
	if nodeID >= NodeID(len(g.nodes)) {
		return nil
	}
	return g.nodes[nodeID]
}

// GetNodeID returns the NodeID associated with given node data
func (g *dependencyGraph) GetNodeID(node Node) NodeID {
	if cmdNode, ok := node.(CmdNode); ok {
		index := cmdNode.Index
		if len(index) == 0 {
			return NodeNoID
		}
		if !g.config.IncludeInitialCommands && !api.CmdID(index[0]).IsReal() {
			return NodeNoID
		}
		if g.config.MergeSubCmdNodes {
			index = index[:1]
		}
		if val := g.cmdNodeIDs.Value(index); val != nil {
			return val.(NodeID)
		}
	} else if obsNode, ok := node.(ObsNode); ok {
		cmdID := obsNode.CmdID
		var obsNodeIDs obsNodeIDs
		if cmdID == api.CmdNoID {
			obsNodeIDs = g.initMemoryNodeIDs
		} else if cmdID.IsReal() {
			obsNodeIDs = g.cmdObsNodeIDs[cmdID]
		} else {
			cmdID = cmdID.Real()
			obsNodeIDs = g.initCmdObsNodeIDs[cmdID]
		}
		index := obsNode.Index
		if obsNode.IsWrite {
			return obsNodeIDs.writeNodeIDStart + NodeID(index)
		} else {
			return obsNodeIDs.readNodeIDStart + NodeID(index)
		}
	}
	return NodeNoID
}

// ForeachCmd iterates over all API calls in the graph
func (g *dependencyGraph) ForeachCmd(ctx context.Context, cb func(context.Context, api.CmdID, api.Cmd) error) error {
	var cmdID api.CmdID
	var cmd api.Cmd
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic at command %v:%v:\n%v", cmdID, cmd, r))
		}
	}()

	subctx := keys.Clone(context.Background(), ctx)
	for i, cmd := range g.initialCommands {
		cmdID = api.CmdID(i).Derived()
		if err := cb(subctx, cmdID, cmd); err != nil {
			if err != api.Break {
				return err
			}
			return nil
		}
		if err := task.StopReason(ctx); err != nil {
			return err
		}
	}

	for i, cmd := range g.capture.Commands {
		cmdID = api.CmdID(i)
		if err := cb(subctx, cmdID, cmd); err != nil {
			if err != api.Break {
				return err
			}
			return nil
		}
		if err := task.StopReason(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ForeachNode iterates over all nodes in the graph
func (g *dependencyGraph) ForeachNode(cb func(NodeID, Node) error) error {
	for i, node := range g.nodes {
		err := cb(NodeID(i), node)
		if err != nil {
			return err
		}
	}
	return nil
}

// ForeachDependency iterates over all pairs (src, tgt), where src depends on tgt
func (g *dependencyGraph) ForeachDependency(cb func(NodeID, NodeID) error) error {
	for i, depsFrom := range g.dependenciesFrom {
		src := NodeID(i)
		for _, tgt := range depsFrom {
			err := cb(src, tgt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ForeachDependencyFrom iterates over all the nodes tgt, where src depends on tgt
func (g *dependencyGraph) ForeachDependencyFrom(src NodeID, cb func(NodeID) error) error {
	for _, tgt := range g.dependenciesFrom[src] {
		err := cb(tgt)
		if err != nil {
			return err
		}
	}
	return nil
}

// ForeachDependencyTo iterates over all the nodes src, where src depends on tgt.
// If Config().ReverseDependencies is false, this will return an error.
func (g *dependencyGraph) ForeachDependencyTo(tgt NodeID, cb func(NodeID) error) error {
	if !g.Config().ReverseDependencies {
		return fmt.Errorf("ForeachDependencyTo called on dependency graph with reverse dependencies disabled.")
	}
	for _, src := range g.dependenciesTo[tgt] {
		err := cb(src)
		if err != nil {
			return err
		}
	}
	return nil
}

// Capture returns the capture whose dependencies are stored in this graph
func (g *dependencyGraph) Capture() *capture.Capture {
	return g.capture
}

// GetCommand returns the command identified by the given CmdID
func (g *dependencyGraph) GetCommand(cmdID api.CmdID) api.Cmd {
	if cmdID.IsReal() {
		if cmdID >= api.CmdID(len(g.capture.Commands)) {
			return nil
		}
		return g.capture.Commands[cmdID]
	} else {
		cmdID = cmdID.Real()
		if cmdID >= api.CmdID(len(g.initialCommands)) {
			return nil
		}
		return g.initialCommands[cmdID]
	}
}

// InitialCommands returns the initial commands, which
// reconstruct the initial state before the first command in the capture.
func (g *dependencyGraph) NumInitialCommands() int {
	return len(g.initialCommands)
}

// Config returns the config used to create this graph
func (g *dependencyGraph) Config() DependencyGraphConfig {
	return g.config
}

func (g *dependencyGraph) setDependencies(src NodeID, targets []NodeID) {
	g.numDependencies -= (uint64)(len(g.dependenciesFrom[src]))
	g.numDependencies += (uint64)(len(targets))
	g.dependenciesFrom[src] = targets
}

func (g *dependencyGraph) addDependency(src, tgt NodeID) {
	deg := len(g.dependenciesFrom[src])
	if deg > 0 {
		last := g.dependenciesFrom[src][deg-1]
		if last == tgt {
			return
		} else if last > tgt {
			panic(fmt.Errorf("Dependency (%v,%v) added after (%v,%v)", src, tgt, src, g.dependenciesFrom[deg-1]))
		}
	}
	if len(g.dependenciesFrom[src]) == cap(g.dependenciesFrom[src]) {
		// This should not happen.
		// addDependency is only called for forward dependencies, and
		// sufficient capacity should have been allocated for all open
		// forward dependencies.
		// Re-allocating the slice could significantly impact performance.
		// Panic so that we can fix this.
		panic(fmt.Errorf("AddDependency: No remaining capacity (size %v)\n", len(g.dependenciesFrom[src])))
	}
	g.dependenciesFrom[src] = append(g.dependenciesFrom[src], tgt)
	g.numDependencies++
}

func (g *dependencyGraph) buildDependenciesTo() {
	degTo := make([]uint32, len(g.nodes))
	for _, depsFrom := range g.dependenciesFrom {
		for _, tgt := range depsFrom {
			degTo[tgt]++
		}
	}
	g.dependenciesTo = make([][]NodeID, len(g.nodes))
	for tgt := range g.dependenciesTo {
		g.dependenciesTo[tgt] = make([]NodeID, 0, degTo[tgt])
	}
	for src, depsFrom := range g.dependenciesFrom {
		for _, tgt := range depsFrom {
			g.dependenciesTo[tgt] = append(g.dependenciesTo[tgt], NodeID(src))
		}
	}
}

func countObservations(cmds []api.Cmd) int {
	numObservations := 0
	for _, cmd := range cmds {
		observations := cmd.Extras().Observations()
		if observations != nil {
			numObservations += len(observations.Reads)
			numObservations += len(observations.Writes)
		}
	}
	return numObservations
}

func (g *dependencyGraph) addCmdNodes(cmdID api.CmdID, observations *api.CmdObservations) obsNodeIDs {
	obsNodeIDs := obsNodeIDs{readNodeIDStart: NodeID(len(g.nodes))}
	if observations != nil {
		for i, obs := range observations.Reads {
			g.nodes = append(g.nodes, ObsNode{obs, cmdID, false, i})
		}
	}
	if cmdID != api.CmdNoID {
		subCmdIdx := api.SubCmdIdx{uint64(cmdID)}
		g.cmdNodeIDs.SetValue(subCmdIdx, NodeID(len(g.nodes)))
		g.nodes = append(g.nodes, CmdNode{subCmdIdx})
	}
	obsNodeIDs.writeNodeIDStart = NodeID(len(g.nodes))
	if observations != nil {
		for i, obs := range observations.Writes {
			g.nodes = append(g.nodes, ObsNode{obs, cmdID, true, i})
		}
	}
	return obsNodeIDs
}
