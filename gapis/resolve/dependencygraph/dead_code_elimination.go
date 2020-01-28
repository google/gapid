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

package dependencygraph

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/config"
)

const (
	// Logs all the commands that were dropped.
	// Recommended to be used with a carefully considered `gapit screenshot`.
	debugDCE = false
)

var (
	deadCodeEliminationCounter         = benchmark.Duration("deadCodeElimination")
	deadCodeEliminationCmdDeadCounter  = benchmark.Integer("deadCodeElimination.cmd.dead")
	deadCodeEliminationCmdLiveCounter  = benchmark.Integer("deadCodeElimination.cmd.live")
	deadCodeEliminationDataDeadCounter = benchmark.Integer("deadCodeElimination.data.dead")
	deadCodeEliminationDataLiveCounter = benchmark.Integer("deadCodeElimination.data.live")
)

// DeadCodeElimination is an implementation of Transformer that outputs live
// commands. That is, all commands which do not affect the requested output are
// omitted. It is named after the standard compiler optimization.
// (state is like memory and commands are instructions which read/write it).
// Construct with NewDeadCodeElimination, do not build directly.
type DeadCodeElimination struct {
	KeepAllAlive bool
	depGraph     *DependencyGraph
	requests     api.CmdIDSet
	lastRequest  api.CmdID
}

// NewDeadCodeElimination constructs and returns a new DeadCodeElimination
// transform.
//
// The transform generates commands from the given depGraph, it does not take
// inputs.
func NewDeadCodeElimination(ctx context.Context, depGraph *DependencyGraph) *DeadCodeElimination {
	return &DeadCodeElimination{
		depGraph: depGraph,
		requests: make(api.CmdIDSet),
	}
}

// Request ensures that we keep alive all commands needed to render framebuffer
// at the given point.
func (t *DeadCodeElimination) Request(id api.CmdID) {
	if id.IsReal() {
		t.requests.Add(id)
		if id > t.lastRequest {
			t.lastRequest = id
		}
	}
}

func (t *DeadCodeElimination) Transform(ctx context.Context, id api.CmdID, c api.Cmd, out transform.Writer) error {
	panic(fmt.Errorf("This transform does not accept input commands"))
}

func (t *DeadCodeElimination) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *DeadCodeElimination) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *DeadCodeElimination) BuffersCommands() bool                              { return false }

func (t *DeadCodeElimination) Flush(ctx context.Context, out transform.Writer) error {
	ctx = status.Start(ctx, "DCE Flush")
	defer status.Finish(ctx)

	if t.KeepAllAlive {
		err := api.ForeachCmd(ctx, t.depGraph.Commands, true, func(ctx context.Context, index api.CmdID, cmd api.Cmd) error {
			return out.MutateAndWrite(ctx, t.depGraph.GetCmdID(int(index)), cmd)
		})
		return err
	}
	t0 := deadCodeEliminationCounter.Start()
	isLive := t.propagateLiveness(ctx)
	deadCodeEliminationCounter.Stop(t0)
	return api.ForeachCmd(ctx, t.depGraph.Commands[:len(isLive)], true, func(ctx context.Context, index api.CmdID, cmd api.Cmd) error {
		id := t.depGraph.GetCmdID(int(index))
		if isLive[index] {
			return out.MutateAndWrite(ctx, id, cmd)
		} else if debugDCE {
			log.I(ctx, "Dropped %v %v", id, cmd)
		}
		return nil
	})
}

// See https://en.wikipedia.org/wiki/Live_variable_analysis
func (t *DeadCodeElimination) propagateLiveness(ctx context.Context) []bool {
	isLive := make([]bool, t.depGraph.NumInitialCommands+int(t.lastRequest)+1)
	state := NewLivenessTree(t.depGraph.GetHierarchyStateMap())
	for i := len(isLive) - 1; i >= 0; i-- {
		b := t.depGraph.Behaviours[i]
		isLive[i] = b.KeepAlive
		// Always ignore commands that abort.
		if b.Aborted {
			continue
		}
		// If this is requested ID, mark all root state as live.
		id := t.depGraph.GetCmdID(i)
		if t.requests.Contains(id) {
			isLive[i] = true
			for root := range t.depGraph.Roots {
				state.MarkLive(root)
			}
		}
		// If any output state is live then this command is live as well.
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
		if config.DebugDeadCodeElimination && t.requests.Contains(id) {
			log.I(ctx, "DCE: Requested cmd %v: %v", id, t.depGraph.Commands[i])
			t.depGraph.Print(ctx, &b)
		}
	}

	{
		// Collect and report statistics
		num, numDead, numLive := len(isLive), 0, 0
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
				deadMem += mem
			} else {
				numLive++
				liveMem += mem
			}
		}
		deadCodeEliminationCmdDeadCounter.Add(int64(numDead))
		deadCodeEliminationCmdLiveCounter.Add(int64(numLive))
		deadCodeEliminationDataDeadCounter.Add(int64(deadMem))
		deadCodeEliminationDataLiveCounter.Add(int64(liveMem))
		log.D(ctx, "DCE: dead: %v%% %v cmds %v MB, live: %v%% %v cmds %v MB",
			100*numDead/num, numDead, deadMem/1024/1024,
			100*numLive/num, numLive, liveMem/1024/1024)
	}
	return isLive
}

// LivenessTree assigns boolean value to each state (live or dead).
// Think of each node as memory range, with children being sub-ranges.
type LivenessTree struct {
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

// NewLivenessTree creates a new tree.
// The parent map defines parent for each node,
// and it must be continuous with no gaps.
func NewLivenessTree(parents map[StateAddress]StateAddress) LivenessTree {
	nodes := make([]livenessNode, len(parents))
	for address, parent := range parents {
		if parent != NullStateAddress {
			nodes[address].parent = &nodes[parent]
		}
	}
	return LivenessTree{nodes: nodes, time: 1}
}

// IsLive returns true if the state, or any of its descendants, are live.
func (l *LivenessTree) IsLive(address StateAddress) bool {
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
func (l *LivenessTree) MarkDead(address StateAddress) {
	node := &l.nodes[address]
	node.live = false
	node.anyLive = false
	node.timestamp = l.time
	l.time++
}

// MarkLive makes the given state, and all of its descendants, live.
func (l *LivenessTree) MarkLive(address StateAddress) {
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
