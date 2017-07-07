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

package transform

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

var (
	deadCodeEliminationCounter         = benchmark.GlobalCounters.Duration("deadCodeElimination")
	deadCodeEliminationCmdDeadCounter  = benchmark.GlobalCounters.Integer("deadCodeElimination.cmd.dead")
	deadCodeEliminationCmdLiveCounter  = benchmark.GlobalCounters.Integer("deadCodeElimination.cmd.live")
	deadCodeEliminationDrawDeadCounter = benchmark.GlobalCounters.Integer("deadCodeElimination.draw.dead")
	deadCodeEliminationDrawLiveCounter = benchmark.GlobalCounters.Integer("deadCodeElimination.draw.live")
	deadCodeEliminationDataDeadCounter = benchmark.GlobalCounters.Integer("deadCodeElimination.data.dead")
	deadCodeEliminationDataLiveCounter = benchmark.GlobalCounters.Integer("deadCodeElimination.data.live")
)

// DeadCodeElimination is an implementation of Transformer that outputs live
// commands. That is, all commands which do not affect the requested output are
// omitted. It is named after the standard compiler optimization.
// (state is like memory and commands are instructions which read/write it).
// Construct with NewDeadCodeElimination, do not build directly.
type DeadCodeElimination struct {
	depGraph    *dependencygraph.DependencyGraph
	requests    api.CmdIDSet
	lastRequest api.CmdID
}

// NewDeadCodeElimination constructs and returns a new DeadCodeElimination
// transform.
//
// The transform generates commands from the given depGraph, it does not take
// inputs.
func NewDeadCodeElimination(ctx context.Context, depGraph *dependencygraph.DependencyGraph) *DeadCodeElimination {
	return &DeadCodeElimination{
		depGraph: depGraph,
		requests: make(api.CmdIDSet),
	}
}

// Request ensures that we keep alive all commands needed to render framebuffer
// at the given point.
func (t *DeadCodeElimination) Request(id api.CmdID) {
	t.requests.Add(id)
	if id > t.lastRequest {
		t.lastRequest = id
	}
}

func (t *DeadCodeElimination) Transform(ctx context.Context, id api.CmdID, c api.Cmd, out Writer) {
	panic(fmt.Errorf("This transform does not accept input atoms"))
}

func (t *DeadCodeElimination) Flush(ctx context.Context, out Writer) {
	t0 := deadCodeEliminationCounter.Start()
	isLive := t.propagateLiveness(ctx)
	deadCodeEliminationCounter.Stop(t0)
	for i, live := range isLive {
		if live {
			out.MutateAndWrite(ctx, api.CmdID(i), t.depGraph.Commands[i])
		}
	}
}

// See https://en.wikipedia.org/wiki/Live_variable_analysis
func (t *DeadCodeElimination) propagateLiveness(ctx context.Context) []bool {
	isLive := make([]bool, t.lastRequest+1)
	state := newLivenessTree(t.depGraph.GetHierarchyStateMap())
	for i := int(t.lastRequest); i >= 0; i-- {
		b := t.depGraph.Behaviours[i]
		isLive[i] = b.KeepAlive
		// Always ignore commands that abort.
		if b.Aborted {
			continue
		}
		// If this is requested ID, mark all root state as live.
		if t.requests.Contains(api.CmdID(i)) {
			isLive[i] = true
			for root := range t.depGraph.Roots {
				state.MarkLive(root)
			}
		}
		// If any output state is live then this atom is live as well.
		for _, write := range b.Writes {
			if state.IsLive(write) {
				isLive[i] = true
				// We just completely wrote the state, so we do not care about
				// the earlier value of the state - it is dead.
				state.MarkDead(write) // KILL
			}
		}
		// Modification is just combined read and write
		for _, modify := range b.Modifies {
			if state.IsLive(modify) {
				isLive[i] = true
				// We will mark it as live since it is also a read, but we have
				// to do it at the end so that all inputs are marked as live.
			}
		}
		// Mark input state as live so that we get all dependencies.
		if isLive[i] {
			for _, modify := range b.Modifies {
				state.MarkLive(modify) // GEN
			}
			for _, read := range b.Reads {
				state.MarkLive(read) // GEN
			}
		}
		// Debug output
		if config.DebugDeadCodeElimination && t.requests.Contains(api.CmdID(i)) {
			log.I(ctx, "DCE: Requested atom %v: %v", i, t.depGraph.Commands[i])
			t.depGraph.Print(ctx, &b)
		}
	}

	{
		// Collect and report statistics
		num, numDead, numDeadDraws, numLive, numLiveDraws := len(isLive), 0, 0, 0, 0
		deadMem, liveMem := uint64(0), uint64(0)
		for i := 0; i < num; i++ {
			cmd := t.depGraph.Commands[i]
			mem := uint64(0)
			if e := cmd.Extras(); e != nil && e.Observations() != nil {
				for _, r := range e.Observations().Reads {
					mem += r.Range.Size
				}
			}
			if !isLive[i] {
				numDead++
				if cmd.CmdFlags().IsDrawCall() {
					numDeadDraws++
				}
				deadMem += mem
			} else {
				numLive++
				if cmd.CmdFlags().IsDrawCall() {
					numLiveDraws++
				}
				liveMem += mem
			}
		}
		deadCodeEliminationCmdDeadCounter.AddInt64(int64(numDead))
		deadCodeEliminationCmdLiveCounter.AddInt64(int64(numLive))
		deadCodeEliminationDrawDeadCounter.AddInt64(int64(numDeadDraws))
		deadCodeEliminationDrawLiveCounter.AddInt64(int64(numLiveDraws))
		deadCodeEliminationDataDeadCounter.AddInt64(int64(deadMem))
		deadCodeEliminationDataLiveCounter.AddInt64(int64(liveMem))
		log.D(ctx, "DCE: dead: %v%% %v cmds %v MB %v draws, live: %v%% %v cmds %v MB %v draws",
			100*numDead/num, numDead, deadMem/1024/1024, numDeadDraws,
			100*numLive/num, numLive, liveMem/1024/1024, numLiveDraws)
	}
	return isLive
}

// livenessTree assigns boolean value to each state (live or dead).
// Think of each node as memory range, with children being sub-ranges.
type livenessTree struct {
	nodes []livenessNode // indexed by StateAddress
	time  int            // current time used for time-stamps
}

type livenessNode struct {
	// Liveness value for this node.
	live bool
	// Optimization 1 - union of liveness of this node and all its descendants.
	anyLive bool
	// Optimization 2 - time of the last write to the 'live' field.
	// This allows efficient update of all descendants.
	// Children with lower time-stamp are effectively deleted.
	timestamp int
	// Link to the parent node, or nil if there is none.
	parent *livenessNode
}

// newLivenessTree creates a new tree.
// The parent map defines parent for each node,
// and it must be continuous with no gaps.
func newLivenessTree(parents map[dependencygraph.StateAddress]dependencygraph.StateAddress) livenessTree {
	nodes := make([]livenessNode, len(parents))
	for address, parent := range parents {
		if parent != dependencygraph.NullStateAddress {
			nodes[address].parent = &nodes[parent]
		}
	}
	return livenessTree{nodes: nodes, time: 1}
}

// IsLive returns true if the state, or any of its descendants, are live.
func (l *livenessTree) IsLive(address dependencygraph.StateAddress) bool {
	node := &l.nodes[address]
	live := node.anyLive // Check descendants as well.
	for p := node.parent; p != nil; p = p.parent {
		if p.timestamp > node.timestamp {
			node = p
			live = p.live // Ignore other descendants.
		}
	}
	return live
}

// MarkDead makes the given state, and all of its descendants, dead.
func (l *livenessTree) MarkDead(address dependencygraph.StateAddress) {
	node := &l.nodes[address]
	node.live = false
	node.anyLive = false
	node.timestamp = l.time
	l.time++
}

// MarkLive makes the given state, and all of its descendants, live.
func (l *livenessTree) MarkLive(address dependencygraph.StateAddress) {
	node := &l.nodes[address]
	node.live = true
	node.anyLive = true
	node.timestamp = l.time
	l.time++
	if p := node.parent; p != nil {
		p.setAnyLive()
	}
}

// setAnyLive is helper to recursively set 'anyLive' flag on ancestors.
func (node *livenessNode) setAnyLive() {
	if p := node.parent; p != nil {
		p.setAnyLive()
		if node.timestamp < p.timestamp {
			// This node is effectively deleted so we need to create it.
			node.live = p.live
			node.timestamp = p.timestamp
		}
	}
	node.anyLive = true
}
