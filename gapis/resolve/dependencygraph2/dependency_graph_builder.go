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
	"reflect"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
)

var (
	dependencyGraphBuilderCounter = benchmark.Duration("DependencyGraph.Builder")
)

// The data needed to build a dependency graph by iterating through the commands in a trace
type dependencyGraphBuilder struct {
	// graph is the dependency graph being constructed
	graph *dependencyGraph

	// currentCmdID is the CmdID of the command currently being processed
	currentCmdID api.CmdID

	// currentCmd is the command currently being processed
	currentCmd api.Cmd

	// currentNodeID is the NodeID of the dependency graph node corresponding to the currentCmdID
	currentNodeID NodeID

	// currentDependencyExists[i] is true if currentNodeID depends on node i
	currentDependencyExists []bool

	// currentDependencies is a slice containing all of the nodes depended on by currentNodeID
	currentDependencies []NodeID

	// currentSubCmdIdx stores the SubCmdIdx of the latest effect
	// currentSubCmdIdx api.SubCmdIdx

	// currentCreateCount is a count of how many CreateResource effects are associated with currentNodeID
	currentCreateCount int

	// fragmentWrites stores all non-overwritten writes to the state, excluding memory writes.
	// fragmentWrites[ref][frag] gives the latest node to write to the fragment
	// frag in the object corresponding to ref
	fragmentWrites map[api.RefID]map[api.Fragment]NodeID

	// memoryWrites stores all non-overwritten writes to memory ranges
	memoryWrites map[memory.PoolID]*memoryWriteList

	// openForwardDependencies tracks pending forward dependencies, where the
	// first node has been processed, but the second node has not yet been
	// processed. The keys are the unique identifier for the forward
	// dependency.
	openForwardDependencies map[interface{}]NodeID

	Stats struct {
		NumCmdNodes     uint64
		NumObsNodes     uint64
		NumReads        uint64
		NumWrites       uint64
		NumMemoryReads  uint64
		NumMemoryWrites uint64
	}
}

// Build a new dependencyGraphBuilder.
func newDependencyGraphBuilder(ctx context.Context, config DependencyGraphConfig,
	c *capture.Capture, initialCmds []api.Cmd) *dependencyGraphBuilder {
	graph := newDependencyGraph(ctx, config, c, initialCmds)
	builder := &dependencyGraphBuilder{
		graph: graph,
		currentDependencyExists: make([]bool, graph.NumNodes()),
		currentCmdID:            api.CmdNoID,
		currentNodeID:           NodeNoID,
		fragmentWrites:          make(map[api.RefID]map[api.Fragment]NodeID),
		memoryWrites:            make(map[memory.PoolID]*memoryWriteList),
		openForwardDependencies: make(map[interface{}]NodeID),
	}
	return builder
}

// BeginCmd is called at the beginning of each API call
func (b *dependencyGraphBuilder) OnBeginCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
	debug(ctx, "OnBeginCmd [%d] %v", cmdID, cmd)
	b.currentCmdID = cmdID
	b.currentCmd = cmd
	b.currentNodeID = b.graph.GetNodeID(CmdNode{api.SubCmdIdx{uint64(cmdID)}})
}

// EndCmd is called at the end of each API call
func (b *dependencyGraphBuilder) OnEndCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
	debug(ctx, "OnEndCmd [%d] %v", cmdID, cmd)
	if len(b.currentDependencies) > 0 {
		sortedDeps := make([]NodeID, len(b.currentDependencies), len(b.currentDependencies)+b.currentCreateCount)
		copy(sortedDeps, b.currentDependencies)
		b.currentDependencies = b.currentDependencies[:0]
		for _, tgt := range sortedDeps {
			b.currentDependencyExists[tgt] = false
		}
		b.graph.setDependencies(b.currentNodeID, sortedDeps)
		b.currentCmdID = api.CmdNoID
		b.currentCmd = nil
		b.currentCreateCount = 0
		b.currentNodeID = NodeNoID
	}
}

func (b *dependencyGraphBuilder) OnGet(ctx context.Context, owner api.Reference, frag api.Fragment, valueRef api.Reference) {
	debug(ctx, "  OnGet (%T %d)%v : %d", owner, owner.RefID(), frag, valueRef.RefID())
	readEffect := ReadFragmentEffect{b.currentNodeID, frag}
	writes, ok := b.fragmentWrites[owner.RefID()]
	if !ok {
		return
	}
	if _, ok := frag.(api.CompleteFragment); ok {
		for writeFrag, writeNode := range writes {
			writeEffect := WriteFragmentEffect{writeNode, writeFrag}
			b.addDependency(writeEffect, readEffect)
		}
	} else if writeNode, ok := writes[frag]; ok {
		writeEffect := WriteFragmentEffect{writeNode, frag}
		b.addDependency(writeEffect, readEffect)
	}
}

func (b *dependencyGraphBuilder) OnSet(ctx context.Context, owner api.Reference, frag api.Fragment, oldValueRef api.Reference, newValueRef api.Reference) {
	debug(ctx, "  OnSet (%T %d)%v : %d → %d", owner, owner.RefID(), frag, oldValueRef.RefID(), newValueRef.RefID())
	if _, ok := frag.(api.CompleteFragment); ok {
		b.fragmentWrites[owner.RefID()] = map[api.Fragment]NodeID{frag: b.currentNodeID}
	} else if writes, ok := b.fragmentWrites[owner.RefID()]; ok {
		writes[frag] = b.currentNodeID
	} else {
		b.fragmentWrites[owner.RefID()] = map[api.Fragment]NodeID{frag: b.currentNodeID}
	}
}

// OnWriteSlice is called when writing to a slice
func (b *dependencyGraphBuilder) OnWriteSlice(ctx context.Context, slice memory.Slice) {
	debug(ctx, "  OnWriteSlice: %v", slice)
	b.addMemoryWrite(WriteMemEffect{b.currentNodeID, slice})
}

// OnReadSlice is called when reading from a slice
func (b *dependencyGraphBuilder) OnReadSlice(ctx context.Context, slice memory.Slice) {
	debug(ctx, "  OnReadSlice: %v", slice)
	b.addMemoryRead(ReadMemEffect{b.currentNodeID, slice})
}

// observationSlice constructs a Slice from a CmdObservation
func observationSlice(obs api.CmdObservation) memory.Slice {
	return memory.NewSlice(obs.Range.Base, obs.Range.Base, obs.Range.Size, obs.Range.Size, obs.Pool, reflect.TypeOf(memory.Char(0)))
}

// OnWriteObs is called when a memory write observation becomes visible
func (b *dependencyGraphBuilder) OnWriteObs(ctx context.Context, obs []api.CmdObservation) {
	for i, o := range obs {
		b.addObs(i, o, true)
	}
}

// OnReadObs is called when a memory read observation becomes visible
func (b *dependencyGraphBuilder) OnReadObs(ctx context.Context, obs []api.CmdObservation) {
	for i, o := range obs {
		b.addObs(i, o, false)
	}
}

// OpenForwardDependency is called to begin a forward dependency.
// See `StateWatcher.OpenForwardDependency` for an explanation of forward dependencies.
func (b *dependencyGraphBuilder) OpenForwardDependency(ctx context.Context, dependencyID interface{}) {
	debug(ctx, "  OpenForwardDependency: %v", dependencyID)
	if _, ok := b.openForwardDependencies[dependencyID]; ok {
		log.I(ctx, "OpenForwardDependency: Forward dependency opened multiple times before being closed. DependencyID: %v, close node: %v", dependencyID, b.currentNodeID)
	} else {
		b.openForwardDependencies[dependencyID] = b.currentNodeID
		b.currentCreateCount++
	}
}

// CloseForwardDependency is called to end a forward dependency.
// See `StateWatcher.OpenForwardDependency` for an explanation of forward dependencies.
func (b *dependencyGraphBuilder) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {
	debug(ctx, "  CloseForwardDependency: %v", dependencyID)
	if openNode, ok := b.openForwardDependencies[dependencyID]; ok {
		delete(b.openForwardDependencies, dependencyID)
		openEffect := OpenForwardDependencyEffect{openNode, dependencyID}
		closeEffect := CloseForwardDependencyEffect{b.currentNodeID, dependencyID}
		b.addDependency(openEffect, closeEffect)
	} else {
		log.I(ctx, "CloseForwardDependency: Forward dependency closed before being opened. DependencyID: %v, close node: %v", dependencyID, b.currentNodeID)
	}
}

func (b *dependencyGraphBuilder) addMemoryWrite(e WriteMemEffect) {
	span := interval.U64Span{
		Start: e.Slice.Base(),
		End:   e.Slice.Base() + e.Slice.Size(),
	}
	if writes, ok := b.memoryWrites[e.Slice.Pool()]; ok {
		i := interval.Replace(writes, span)
		(*writes)[i].effect = e
	} else {
		b.memoryWrites[e.Slice.Pool()] = &memoryWriteList{memoryWrite{e, span}}
	}
}

func (b *dependencyGraphBuilder) addMemoryRead(e ReadMemEffect) {
	span := interval.U64Span{
		Start: e.Slice.Base(),
		End:   e.Slice.Base() + e.Slice.Size(),
	}
	if writes, ok := b.memoryWrites[e.Slice.Pool()]; ok {
		i, c := interval.Intersect(writes, span)
		for _, w := range (*writes)[i : i+c] {
			b.addDependency(w.effect, e)
		}
	}
}

func (b *dependencyGraphBuilder) addObs(index int, obs api.CmdObservation, isWrite bool) {
	obsNode := ObsNode{
		CmdObservation: obs,
		CmdID:          b.currentCmdID,
		IsWrite:        isWrite,
		Index:          index,
	}
	b.addMemoryWrite(WriteMemEffect{b.graph.GetNodeID(obsNode), observationSlice(obs)})
}

// LogStats logs some interesting stats about the graph construction
func (b *dependencyGraphBuilder) LogStats(ctx context.Context) {
	log.I(ctx, "nodes: %v, cmds: %v, obs: %v, edges: %v",
		b.graph.NumNodes(), b.Stats.NumCmdNodes, b.Stats.NumObsNodes, b.graph.NumDependencies())
	log.I(ctx, "reads: %v, writes: %v, memReads: %v, memWrites: %v",
		b.Stats.NumReads, b.Stats.NumWrites, b.Stats.NumMemoryReads, b.Stats.NumMemoryWrites)
	t := dependencyGraphBuilderCounter.Get()
	if t != 0 {
		log.I(ctx, "time: %v", dependencyGraphBuilderCounter.Get())
	}
}

func (b *dependencyGraphBuilder) addDependency(write WriteEffect, read ReadEffect) {
	writeID := write.GetNodeID()
	readID := read.GetNodeID()
	if _, ok := write.(ReverseEffect); ok {
		b.graph.addDependency(writeID, readID)
		return
	}

	if readID != b.currentNodeID {
		panic(fmt.Errorf("Read from node %v; expected %v", readID, b.currentNodeID))
	}
	if !b.currentDependencyExists[writeID] {
		b.currentDependencies = append(b.currentDependencies, writeID)
		b.currentDependencyExists[writeID] = true
	}
}

func BuildDependencyGraph(ctx context.Context, config DependencyGraphConfig,
	c *capture.Capture, initialCmds []api.Cmd, initialRanges interval.U64RangeList) (DependencyGraph, error) {
	builder := newDependencyGraphBuilder(ctx, config, c, initialCmds)
	var state *api.GlobalState
	if config.IncludeInitialCommands {
		state = c.NewUninitializedState(ctx, initialRanges)
	} else {
		state = c.NewState(ctx)
	}
	err := builder.graph.ForeachCmd(ctx, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		return cmd.Mutate(ctx, id, state, nil, builder)
	})
	if err != nil {
		return nil, err
	}
	if config.ReverseDependencies {
		builder.graph.buildDependenciesTo()
	}
	return builder.graph, nil
}

func debug(ctx context.Context, fmt string, args ...interface{}) {
	if config.DebugDependencyGraph {
		log.D(ctx, fmt, args...)
	}
}
