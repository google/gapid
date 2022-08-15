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
	Index    api.SubCmdIdx
	CmdFlags api.CmdFlags
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

type NodeAccesses struct {
	FragmentAccesses []FragmentAccess
	MemoryAccesses   []MemoryAccess
	ForwardAccesses  []ForwardAccess
	ParentNode       NodeID
	InitCmdNodes     []NodeID
}

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

	SaveNodeAccesses bool
}

// NodeID identifies a node in a dependency graph
type NodeID uint32

const NodeNoID = NodeID(math.MaxUint32)

// NodeIDSorter is a structure to use for sorting NodeIDs in the sort package
type NodeIDSorter struct {
	Nodes []NodeID
}

// Len returns the length of the node list
func (s *NodeIDSorter) Len() int {
	return len(s.Nodes)
}

// Less returns trus if the elements at index i are less than j
func (s *NodeIDSorter) Less(i, j int) bool {
	return s.Nodes[i] < s.Nodes[j]
}

// Swap swaps the locations of 2 nodes in the list
func (s *NodeIDSorter) Swap(i, j int) {
	s.Nodes[i], s.Nodes[j] = s.Nodes[j], s.Nodes[i]
}

// DependencyGraph stores the dependencies among api calls and memory observations,
type DependencyGraph interface {

	// NumNodes returns the number of nodes in the graph
	NumNodes() int

	// NumDependencies returns the number of dependencies (edges) in the graph
	NumDependencies() uint64

	// GetNode returns the node data associated with the given NodeID
	GetNode(NodeID) Node

	// GetNodeID returns the NodeID associated with given node data
	GetCmdNodeID(api.CmdID, api.SubCmdIdx) NodeID

	// GetCmdAncestorNodeIDs returns the NodeIDs associated with the ancestors of the
	// given subcommand.
	GetCmdAncestorNodeIDs(api.CmdID, api.SubCmdIdx) []NodeID

	// ForeachCmd iterates over all API calls in the graph.
	// If IncludeInitialCommands is true, this includes the initial commands
	// which reconstruct the initial state.
	// CmdIDs for initial commands are:
	//   CmdID(0).Derived(), CmdID(1).Derived(), ...
	// Whether or not IncludeInitialCommands is true, the CmdIDs for captured
	// commands are: 0, 1, 2, ...
	ForeachCmd(ctx context.Context,
		cb func(context.Context, api.CmdID, api.Cmd) error) error

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
	Capture() *capture.GraphicsCapture

	// GetUnopenedForwardDependencies returns the commands that have dependencies that
	// are not part of the capture.
	GetUnopenedForwardDependencies() []api.CmdID

	// GetCommand returns the command identified by the given CmdID
	GetCommand(api.CmdID) api.Cmd

	// NumInitialCommands returns the number of initial commands
	// (the commands needed to reconstruct the initial state before the
	// first command in the capture)
	NumInitialCommands() int

	GetNodeAccesses(NodeID) NodeAccesses

	// Config returns the config used to create this graph
	Config() DependencyGraphConfig
}

type obsNodeIDs struct {
	readNodeIDStart  NodeID
	writeNodeIDStart NodeID
}

type dependencyGraph struct {
	capture                     *capture.GraphicsCapture
	cmdNodeIDs                  *api.SubCmdIdxTrie
	initialCommands             []api.Cmd
	nodes                       []Node
	numDependencies             uint64
	dependenciesFrom            [][]NodeID
	dependenciesTo              [][]NodeID
	nodeAccesses                []NodeAccesses
	unopenedForwardDependencies []api.CmdID
	stateRefs                   map[api.RefID]RefFrag

	config DependencyGraphConfig
}

// newDependencyGraph constructs a new dependency graph
func newDependencyGraph(ctx context.Context, config DependencyGraphConfig,
	c *capture.GraphicsCapture, initialCmds []api.Cmd, nodes []Node) *dependencyGraph {
	g := &dependencyGraph{
		capture:          c,
		cmdNodeIDs:       new(api.SubCmdIdxTrie),
		initialCommands:  initialCmds,
		config:           config,
		nodes:            nodes,
		dependenciesFrom: make([][]NodeID, len(nodes)),
	}
	for i, n := range nodes {
		if c, ok := n.(CmdNode); ok {
			g.cmdNodeIDs.SetValue(c.Index, (NodeID)(i))
		}
	}
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

// GetCmdNodeID returns the NodeID associated with a given (sub)command
func (g *dependencyGraph) GetCmdNodeID(cmdID api.CmdID, idx api.SubCmdIdx) NodeID {
	fullIdx := append(api.SubCmdIdx{(uint64)(cmdID)}, idx...)
	x := g.cmdNodeIDs.Value(fullIdx)
	if x != nil {
		return x.(NodeID)
	}
	return NodeNoID
}

// GetCmdAncestorNodeIDs returns the NodeIDs associated with the ancestors of the
// given subcommand.
func (g *dependencyGraph) GetCmdAncestorNodeIDs(cmdID api.CmdID, idx api.SubCmdIdx) []NodeID {
	fullIdx := append(api.SubCmdIdx{(uint64)(cmdID)}, idx...)
	values := g.cmdNodeIDs.Values(fullIdx)
	nodeIDs := make([]NodeID, len(values))
	for i, v := range values {
		if v != nil {
			nodeIDs[i] = v.(NodeID)
		} else {
			nodeIDs[i] = NodeNoID
		}
	}
	return nodeIDs
}

// ForeachCmd iterates over all API calls in the graph.
// If IncludeInitialCommands is true, this includes the initial commands
// which reconstruct the initial state.
// CmdIDs for initial commands are:
//
//	CmdID(0).Derived(), CmdID(1).Derived(), ...
//
// Whether or not IncludeInitialCommands is true, the CmdIDs for captured
// commands are: 0, 1, 2, ...
func (g *dependencyGraph) ForeachCmd(ctx context.Context, cb func(context.Context, api.CmdID, api.Cmd) error) error {
	if g.config.IncludeInitialCommands {
		cbDerived := func(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) error {
			return cb(ctx, cmdID.Derived(), cmd)
		}
		if err := api.ForeachCmd(ctx, g.initialCommands, true, cbDerived); err != nil {
			return err
		}
	}
	return api.ForeachCmd(ctx, g.capture.Commands, true, cb)
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
func (g *dependencyGraph) Capture() *capture.GraphicsCapture {
	return g.capture
}

func (g *dependencyGraph) GetUnopenedForwardDependencies() []api.CmdID {
	return g.unopenedForwardDependencies
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

func (g *dependencyGraph) GetNodeAccesses(nodeID NodeID) NodeAccesses {
	if nodeID < NodeID(len(g.nodeAccesses)) {
		return g.nodeAccesses[nodeID]
	} else {
		return NodeAccesses{
			ParentNode: NodeNoID,
		}
	}
}

// Config returns the config used to create this graph
func (g *dependencyGraph) Config() DependencyGraphConfig {
	return g.config
}

func (g *dependencyGraph) addNode(node Node) NodeID {
	nodeID := (NodeID)(len(g.nodes))
	g.nodes = append(g.nodes, node)
	g.dependenciesFrom = append(g.dependenciesFrom, []NodeID{})
	if g.config.SaveNodeAccesses {
		g.nodeAccesses = append(g.nodeAccesses, NodeAccesses{})
	}
	if cmdNode, ok := node.(CmdNode); ok {
		g.cmdNodeIDs.SetValue(cmdNode.Index, nodeID)
	}
	return nodeID
}

func (g *dependencyGraph) setNodeAccesses(nodeID NodeID, acc NodeAccesses) {
	if nodeID < NodeID(len(g.nodeAccesses)) {
		g.nodeAccesses[nodeID] = acc
	}
}

func (g *dependencyGraph) setDependencies(src NodeID, targets []NodeID) {
	g.numDependencies -= (uint64)(len(g.dependenciesFrom[src]))
	g.numDependencies += (uint64)(len(targets))
	g.dependenciesFrom[src] = targets
}

func (g *dependencyGraph) addUnopenedForwardDependency(id api.CmdID) {
	g.unopenedForwardDependencies = append(g.unopenedForwardDependencies, id)
}

func (g *dependencyGraph) setStateRefs(stateRefs map[api.RefID]RefFrag) {
	if g.config.SaveNodeAccesses {
		g.stateRefs = stateRefs
	}
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
