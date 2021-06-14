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
	"sort"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
)

var (
	dependencyGraphBuilderCounter = benchmark.Duration("DependencyGraph.Builder")
)

type NodeStats struct {
	NumFragReads        uint64
	NumFragWrites       uint64
	NumMemReads         uint64
	NumMemWrites        uint64
	NumForwardDepOpens  uint64
	NumForwardDepCloses uint64
	NumForwardDepDrops  uint64
	NumDeps             uint64
	NumFragDeps         uint64
	NumCompleteFragDeps uint64
	NumMemDeps          uint64
	UniqueFragReads     uint64
	UniqueFragWrites    uint64
	UniqueMemReads      uint64
	UniqueMemWrites     uint64
	UniqueDeps          uint64
}

// AccessMode is a bitfield that records read/write accesses
type AccessMode uint

const (
	// For memory accesses, we need to distinguish "PLAIN" memory accesses that
	// are just recording read/write accesses (relevant for e.g. the framegraph)
	// and "DEP" accesses that are more subtely managed for the tracking of
	// dependencies to properly handle e.g. read-after-write by the same command.

	// PLAIN: just plain read/write accesses
	ACCESS_PLAIN_READ AccessMode = 1 << iota
	ACCESS_PLAIN_WRITE
	// DEP: read/write relevant for dependencies
	ACCESS_DEP_READ
	ACCESS_DEP_WRITE
	// Combined values
	ACCESS_READ  AccessMode = ACCESS_DEP_READ | ACCESS_PLAIN_READ
	ACCESS_WRITE AccessMode = ACCESS_DEP_WRITE | ACCESS_PLAIN_WRITE
)

// The data needed to build a dependency graph by iterating through the commands in a trace
type dependencyGraphBuilder struct {
	// graph is the dependency graph being constructed
	// graph *dependencyGraph
	capture *capture.GraphicsCapture

	config DependencyGraphConfig

	fragWatcher    FragWatcher
	memWatcher     MemWatcher
	forwardWatcher ForwardWatcher

	graphBuilder GraphBuilder

	subCmdStack []CmdContext

	Stats struct {
		NumFragReads        uint64
		NumFragWrites       uint64
		NumMemReads         uint64
		NumMemWrites        uint64
		NumForwardDepOpens  uint64
		NumForwardDepCloses uint64
		NumForwardDepDrops  uint64
	}
}

// Build a new dependencyGraphBuilder.
func newDependencyGraphBuilder(ctx context.Context, config DependencyGraphConfig,
	c *capture.GraphicsCapture, initialCmds []api.Cmd, state *api.GlobalState) *dependencyGraphBuilder {
	builder := &dependencyGraphBuilder{}
	builder.capture = c
	builder.config = config
	builder.fragWatcher = NewFragWatcher()
	builder.memWatcher = NewMemWatcher()
	builder.forwardWatcher = NewForwardWatcher()
	builder.graphBuilder = NewGraphBuilder(ctx, config, c, initialCmds, state)
	return builder
}

// BeginCmd is called at the beginning of each API call
func (b *dependencyGraphBuilder) OnBeginCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
	if len(b.subCmdStack) > 0 {
		log.E(ctx, "OnBeginCmd called while processing another command")
		b.subCmdStack = b.subCmdStack[:0]
	}
	cmdCtx := b.graphBuilder.GetCmdContext(ctx, cmdID, cmd)
	b.graphBuilder.OnBeginCmd(ctx, cmdCtx)
	b.fragWatcher.OnBeginCmd(ctx, cmdCtx)
	b.memWatcher.OnBeginCmd(ctx, cmdCtx)
	b.forwardWatcher.OnBeginCmd(ctx, cmdCtx)
	b.subCmdStack = append(b.subCmdStack, cmdCtx)
}

// EndCmd is called at the end of each API call
func (b *dependencyGraphBuilder) OnEndCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
	if len(b.subCmdStack) > 1 {
		log.E(ctx, "OnEndCmd called while still processing subcommands")
	}
	cmdCtx := b.cmdCtx()

	fragAcc := b.fragWatcher.OnEndCmd(ctx, cmdCtx)
	memAcc := b.memWatcher.OnEndCmd(ctx, cmdCtx)
	accesses := b.forwardWatcher.OnEndCmd(ctx, cmdCtx)

	b.graphBuilder.AddDependencies(ctx, fragAcc, memAcc, accesses.nodeAccesses, accesses.isUnopened)

	b.subCmdStack = b.subCmdStack[:0]
}

func (b *dependencyGraphBuilder) OnBeginSubCmd(ctx context.Context, subCmdIdx api.SubCmdIdx, recordIdx api.RecordIdx) {
	if len(b.subCmdStack) == 0 {
		log.E(ctx, "OnBeginSubCmd called while not processing any command")
	}

	cmdCtx := b.cmdCtx()
	if b.config.MergeSubCmdNodes {
		subCmdCtx := cmdCtx
		subCmdCtx.subCmdIdx = subCmdIdx
		b.graphBuilder.OnBeginSubCmd(ctx, cmdCtx, subCmdCtx, recordIdx)
		return
	}

	subCmdCtx := b.graphBuilder.GetSubCmdContext(cmdCtx.cmdID, subCmdIdx)

	b.graphBuilder.OnBeginSubCmd(ctx, cmdCtx, subCmdCtx, recordIdx)
	b.fragWatcher.OnBeginSubCmd(ctx, cmdCtx, subCmdCtx)
	b.memWatcher.OnBeginSubCmd(ctx, cmdCtx, subCmdCtx)
	b.forwardWatcher.OnBeginSubCmd(ctx, cmdCtx, subCmdCtx)

	b.subCmdStack = append(b.subCmdStack, subCmdCtx)
}

func (b *dependencyGraphBuilder) OnEndSubCmd(ctx context.Context) {
	if b.config.MergeSubCmdNodes {
		return
	}
	if len(b.subCmdStack) < 2 {
		log.E(ctx, "OnEndSubCmd called while not processing any subcommand")
	}
	cmdCtx := b.cmdCtx()

	b.fragWatcher.OnEndSubCmd(ctx, cmdCtx)
	b.memWatcher.OnEndSubCmd(ctx, cmdCtx)
	b.forwardWatcher.OnEndSubCmd(ctx, cmdCtx)

	b.subCmdStack = b.subCmdStack[:len(b.subCmdStack)-1]
}

func (b *dependencyGraphBuilder) OnReadFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, valueRef api.RefObject, track bool) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumFragReads++
	b.Stats.NumFragReads++
	b.fragWatcher.OnReadFrag(ctx, cmdCtx, owner, frag, valueRef, track)
}

func (b *dependencyGraphBuilder) OnWriteFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, oldValueRef api.RefObject, newValueRef api.RefObject, track bool) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumFragWrites++
	b.Stats.NumFragWrites++
	b.fragWatcher.OnWriteFrag(ctx, cmdCtx, owner, frag, oldValueRef, newValueRef, track)
}

// OnWriteSlice is called when writing to a slice
func (b *dependencyGraphBuilder) OnWriteSlice(ctx context.Context, slice memory.Slice) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumMemWrites++
	b.Stats.NumMemWrites++
	b.memWatcher.OnWriteSlice(ctx, cmdCtx, slice)
}

// OnReadSlice is called when reading from a slice
func (b *dependencyGraphBuilder) OnReadSlice(ctx context.Context, slice memory.Slice) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumMemReads++
	b.Stats.NumMemReads++
	b.memWatcher.OnReadSlice(ctx, cmdCtx, slice)
}

// OnWriteObs is called when a memory write observation becomes visible
func (b *dependencyGraphBuilder) OnWriteObs(ctx context.Context, obs []api.CmdObservation) {
	cmdCtx := b.cmdCtx()
	b.memWatcher.OnWriteObs(ctx, cmdCtx, obs, b.graphBuilder.GetObsNodeIDs(cmdCtx.cmdID, obs, true))
}

// OnReadObs is called when a memory read observation becomes visible
func (b *dependencyGraphBuilder) OnReadObs(ctx context.Context, obs []api.CmdObservation) {
	cmdCtx := b.cmdCtx()
	b.memWatcher.OnReadObs(ctx, cmdCtx, obs, b.graphBuilder.GetObsNodeIDs(cmdCtx.cmdID, obs, false))
}

// OpenForwardDependency is called to begin a forward dependency.
// See `StateWatcher.OpenForwardDependency` for an explanation of forward dependencies.
func (b *dependencyGraphBuilder) OpenForwardDependency(ctx context.Context, dependencyID interface{}) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumForwardDepOpens++
	b.Stats.NumForwardDepOpens++
	b.forwardWatcher.OpenForwardDependency(ctx, cmdCtx, dependencyID)
}

// CloseForwardDependency is called to end a forward dependency.
// See `StateWatcher.OpenForwardDependency` for an explanation of forward dependencies.
func (b *dependencyGraphBuilder) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumForwardDepCloses++
	b.Stats.NumForwardDepCloses++
	b.forwardWatcher.CloseForwardDependency(ctx, cmdCtx, dependencyID)
}

// DropForwardDependency is called to abandon a previously opened
// forward dependency, without actually adding the forward dependency.
// See `StateWatcher.OpenForwardDependency` for an explanation of forward dependencies.
func (b *dependencyGraphBuilder) DropForwardDependency(ctx context.Context, dependencyID interface{}) {
	cmdCtx := b.cmdCtx()
	cmdCtx.stats.NumForwardDepDrops++
	b.Stats.NumForwardDepDrops++
	b.forwardWatcher.DropForwardDependency(ctx, cmdCtx, dependencyID)
}

func (b *dependencyGraphBuilder) OnRecordSubCmd(ctx context.Context, recordIdx api.RecordIdx) {
	cmdCtx := b.cmdCtx()
	b.graphBuilder.OnRecordSubCmd(ctx, cmdCtx, recordIdx)
}

// LogStats logs some interesting stats about the graph construction
func (b *dependencyGraphBuilder) LogStats(ctx context.Context, full bool) {
	log.I(ctx, "Dependency Graph Stats:")
	graphStats := b.graphBuilder.GetStats()
	log.I(ctx, "          NumCmdNodes: %-8v  NumObsNodes: %v", graphStats.NumCmdNodes, graphStats.NumObsNodes)
	log.I(ctx, "        Accesses:")
	log.I(ctx, "          NumFragReads: %-8v  UniqueFragReads: %v", b.Stats.NumFragReads, graphStats.UniqueFragReads)
	log.I(ctx, "          NumFragWrites: %-7v  UniqueFragWrites: %v", b.Stats.NumFragWrites, graphStats.UniqueFragWrites)
	log.I(ctx, "          NumMemReads: %-9v  UniqueMemReads: %v", b.Stats.NumMemReads, graphStats.UniqueMemReads)
	log.I(ctx, "          NumMemWrites: %-8v  UniqueMemWrites: %v", b.Stats.NumMemWrites, graphStats.UniqueMemWrites)
	log.I(ctx, "          NumForwardDepOpens: %-4v  NumForwardDepCloses: %-4v  NumForwardDepDrops: %v", b.Stats.NumForwardDepOpens, b.Stats.NumForwardDepCloses, b.Stats.NumForwardDepDrops)
	log.I(ctx, "        Deps:")
	log.I(ctx, "          NumDeps: %-15v  UniqueDeps: %v", graphStats.NumDeps, graphStats.UniqueDeps)
	log.I(ctx, "          NumFragDeps: %-4v  NumCompleteFragDeps: %-4v  NumMemDeps: %v", graphStats.NumFragDeps, graphStats.NumCompleteFragDeps, graphStats.NumMemDeps)

	if full {
		graph := b.graphBuilder.GetGraph()
		nodeIDs := make([]NodeID, len(graph.nodes))
		for i := range nodeIDs {
			nodeIDs[i] = (NodeID)(i)
		}

		sortBy := func(f func(n NodeID) uint64) {
			sort.Slice(nodeIDs, func(i, j int) bool {
				return f(nodeIDs[i]) > f(nodeIDs[j])
			})
		}

		logNode := func(v uint64, n NodeID) {
			var cmdStr string
			if node, ok := graph.nodes[n].(CmdNode); ok {
				if len(node.Index) == 1 {
					cmdID := (api.CmdID)(node.Index[0])
					cmd := graph.GetCommand(cmdID)
					cmdStr = fmt.Sprintf("%v", cmd)
				}
			}
			log.I(ctx, "%-9v  %v  %s", v, graph.nodes[n], cmdStr)
			s := b.graphBuilder.GetNodeStats(n)
			log.I(ctx, "        Accesses:")
			log.I(ctx, "          NumFragReads: %-8v  UniqueFragReads: %v", s.NumFragReads, s.UniqueFragReads)
			log.I(ctx, "          NumFragWrites: %-7v  UniqueFragWrites: %v", s.NumFragWrites, s.UniqueFragWrites)
			log.I(ctx, "          NumMemReads: %-9v  UniqueMemReads: %v", s.NumMemReads, s.UniqueMemReads)
			log.I(ctx, "          NumMemWrites: %-8v  UniqueMemWrites: %v", s.NumMemWrites, s.UniqueMemWrites)
			log.I(ctx, "          NumForwardDepOpens: %-4v  NumForwardDepCloses: %-4v  NumForwardDepDrops: %v", s.NumForwardDepOpens, s.NumForwardDepCloses, s.NumForwardDepDrops)
			log.I(ctx, "        Deps:")
			log.I(ctx, "          NumDeps: %-15v  UniqueDeps: %v", s.NumDeps, s.UniqueDeps)
			log.I(ctx, "          NumFragDeps: %-4v  NumCompleteFragDeps: %-4v  NumMemDeps: %v", s.NumFragDeps, s.NumCompleteFragDeps, s.NumMemDeps)
		}

		logTop := func(c uint, f func(n NodeID) uint64) {
			sortBy(f)
			for _, n := range nodeIDs[:c] {
				logNode(f(n), n)
			}
		}

		log.I(ctx, "Top Nodes by total accesses:")
		totalAccesses := func(n NodeID) uint64 {
			s := b.graphBuilder.GetNodeStats(n)
			return s.NumFragReads +
				s.NumFragWrites +
				s.NumMemReads +
				s.NumMemWrites +
				s.NumForwardDepOpens +
				s.NumForwardDepCloses +
				s.NumForwardDepDrops
		}
		logTop(10, totalAccesses)

		log.I(ctx, "Top Nodes by unique accesses:")
		uniqueAccesses := func(n NodeID) uint64 {
			s := b.graphBuilder.GetNodeStats(n)
			return s.UniqueFragReads +
				s.UniqueFragWrites +
				s.UniqueMemReads +
				s.UniqueMemWrites +
				s.NumForwardDepOpens +
				s.NumForwardDepCloses +
				s.NumForwardDepDrops
		}
		logTop(10, uniqueAccesses)
	}
}

func BuildDependencyGraph(ctx context.Context, config DependencyGraphConfig,
	c *capture.GraphicsCapture, initialCmds []api.Cmd, initialRanges interval.U64RangeList) (DependencyGraph, error) {
	ctx = status.Start(ctx, "BuildDependencyGraph")
	defer status.Finish(ctx)
	var state *api.GlobalState
	if config.IncludeInitialCommands {
		state = c.NewUninitializedState(ctx).ReserveMemory(initialRanges)
	} else {
		state = c.NewState(ctx)
	}
	b := newDependencyGraphBuilder(ctx, config, c, initialCmds, state)
	mutate := func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		return cmd.Mutate(ctx, id, state, nil, b)
	}
	mutateD := func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		return mutate(ctx, id.Derived(), cmd)
	}
	err := api.ForeachCmd(ctx, initialCmds, true, mutateD)
	if err != nil {
		return nil, err
	}
	err = api.ForeachCmd(ctx, c.Commands, true, mutate)
	if err != nil {
		return nil, err
	}

	if config.ReverseDependencies {
		b.graphBuilder.BuildReverseDependencies()
	}

	graph := b.graphBuilder.GetGraph()

	b.LogStats(ctx, false)

	if graph.config.SaveNodeAccesses {
		graph.setStateRefs(b.fragWatcher.GetStateRefs())
	}

	return graph, nil
}

func (b *dependencyGraphBuilder) cmdCtx() CmdContext {
	if len(b.subCmdStack) == 0 {
		return CmdContext{}
	}
	return b.subCmdStack[len(b.subCmdStack)-1]
}

type Distribution struct {
	SmallBins []uint64
	LargeBins map[uint64]uint64
}

func (d Distribution) Add(x uint64) {
	if x < uint64(len(d.SmallBins)) {
		d.SmallBins[x]++
	} else {
		if d.LargeBins == nil {
			d.LargeBins = make(map[uint64]uint64)
		}
		d.LargeBins[x]++
	}
}

type CmdContext struct {
	cmdID        api.CmdID
	cmd          api.Cmd
	subCmdIdx    api.SubCmdIdx
	nodeID       NodeID
	depth        int
	parentNodeID NodeID
	stats        *NodeStats
}
