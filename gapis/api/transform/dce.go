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
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

var (
	dCECounter         = benchmark.Duration("DCE")
	dCECmdDeadCounter  = benchmark.Integer("DCE.cmd.dead")
	dCECmdLiveCounter  = benchmark.Integer("DCE.cmd.live")
	dCEDrawDeadCounter = benchmark.Integer("DCE.draw.dead")
	dCEDrawLiveCounter = benchmark.Integer("DCE.draw.live")
	dCEDataDeadCounter = benchmark.Integer("DCE.data.dead")
	dCEDataLiveCounter = benchmark.Integer("DCE.data.live")
)

type commandIndicesSet struct {
	api.SubCmdIdxTrie
	count uint64
}

func newCommandIndicesSet() *commandIndicesSet {
	return &commandIndicesSet{api.SubCmdIdxTrie{}, uint64(0)}
}

func (s *commandIndicesSet) insert(fci api.SubCmdIdx) {
	if v := s.Value(fci); v != nil {
		s.count++
	}
	s.SetValue(fci, true)
}

func (s *commandIndicesSet) contains(fci api.SubCmdIdx) bool {
	v := s.Value(fci)
	if v != nil {
		if bv, ok := v.(bool); ok {
			return bv
		}
		return false
	}
	return false
}

// DCE contains an execution footprint built from a list of commands, and a
// list of requested command indices. It drives the back-propagation to drop
// commands which are not contributing to the final state at the requested
// commands.
type DCE struct {
	footprint        *dependencygraph.Footprint
	endBehaviorIndex uint64
	endCmdIndex      api.CmdID
	requests         *commandIndicesSet
	requestCount     uint64
}

// NewDCE constructs a new DCE instance and returns a pointer to the created
// DCE instance.
func NewDCE(ctx context.Context, footprint *dependencygraph.Footprint) *DCE {
	return &DCE{
		footprint: footprint,
		requests:  newCommandIndicesSet(),
	}
}

// Request added a requsted command or subcommand, represented by its full
// command index, to the DCE.
func (t *DCE) Request(ctx context.Context, fci api.SubCmdIdx) {
	t.requestCount++
	t.requests.insert(fci)
	bi := t.footprint.BehaviorIndex(ctx, fci)
	if bi > t.endBehaviorIndex {
		t.endBehaviorIndex = bi
	}
	if api.CmdID(fci[0]) > t.endCmdIndex {
		t.endCmdIndex = api.CmdID(fci[0])
	}
}

// Transform is to comform the interface of Transformer, but does not accept
// any input.
func (t *DCE) Transform(ctx context.Context, id api.CmdID, c api.Cmd,
	out Writer) {
	panic(fmt.Errorf("This transform does not accept input atoms"))
}

// Flush is to comform the interface of Transformer. Flush starts the back
// propagation of the behaviors recorded in the footprint from the last
// requested command (the one with largest SubCmdIdx, not the one added the
// last in the order of time) to get a list of alive commands. Then it sends
// the alive commands to the following transforms to mutate them and write them
// to build instructions for replay.
func (t *DCE) Flush(ctx context.Context, out Writer) {
	if t.endBehaviorIndex >= uint64(len(t.footprint.Behaviors)) {
		log.E(ctx, "DCE: Cannot backpropagate through def-use chain from behavior index: %v, "+
			"with length of behavior list: %v.", t.endBehaviorIndex, len(t.footprint.Behaviors))
		log.W(ctx, "DCE: Fallback to disable DCE.")
		for i := api.CmdID(0); i <= t.endCmdIndex; i++ {
			out.MutateAndWrite(ctx, i, t.footprint.Commands[int(i)])
		}
		return
	}
	t0 := dCECounter.Start()
	livenessBoard, aliveCmds := t.backPropagate(ctx)
	dCECounter.Stop(t0)
	flushedCommands := newCommandIndicesSet()

	numCmd, numDead, numDeadDraws, numLive, numLiveDraws := 0, 0, 0, 0, 0
	deadMem, liveMem := uint64(0), uint64(0)

	// This state is the state after all the commands have been mutated.
	// It is currently only used for getting command flags, which in turn is
	// used for getting statistics.
	// Do not use for expected state!
	s := out.State()

	for bi := uint64(0); bi <= t.endBehaviorIndex; bi++ {
		bh := t.footprint.Behaviors[bi]
		fci := bh.Owner

		if livenessBoard[bi] && len(fci) == 1 && !flushedCommands.contains(fci) {
			flushedCommands.insert(fci)
			aliveCmd := t.footprint.Commands[fci[0]]

			// Logging the DCE result of alive commands
			numCmd++
			numLive++
			if aliveCmd.CmdFlags(ctx, api.CmdNoID, s).IsDrawCall() {
				numLiveDraws++
			}
			if e := aliveCmd.Extras(); e != nil && e.Observations() != nil {
				for _, r := range e.Observations().Reads {
					liveMem += r.Range.Size
				}
			}

			out.MutateAndWrite(ctx, api.CmdID(fci[0]), aliveCmd)
		} else {
			if len(fci) == 1 && !aliveCmds.contains(fci) {
				// logging the DCE result of dead commands
				deadCmd := t.footprint.Commands[fci[0]]
				numCmd++
				numDead++
				if deadCmd.CmdFlags(ctx, api.CmdNoID, s).IsDrawCall() {
					numDeadDraws++
				}
				if e := deadCmd.Extras(); e != nil && e.Observations() != nil {
					for _, r := range e.Observations().Reads {
						deadMem += r.Range.Size
					}
				}
			}
		}
	}
	dCECmdDeadCounter.Add(int64(numDead))
	dCECmdLiveCounter.Add(int64(numLive))
	dCEDrawDeadCounter.Add(int64(numDeadDraws))
	dCEDrawLiveCounter.Add(int64(numLiveDraws))
	dCEDataDeadCounter.Add(int64(deadMem))
	dCEDataLiveCounter.Add(int64(liveMem))
	log.D(ctx, "DCE: dead: %v%% %v cmds %v MB %v draws, live: %v%% %v cmds %v MB %v draws",
		100*numDead/numCmd, numDead, deadMem/1024/1024, numDeadDraws,
		100*numLive/numCmd, numLive, liveMem/1024/1024, numLiveDraws)
}

func (t *DCE) backPropagate(ctx context.Context) (
	[]bool, *commandIndicesSet) {
	livenessBoard := make([]bool, t.endBehaviorIndex+1)
	aliveCommands := newCommandIndicesSet()
	usedMachines := map[dependencygraph.BackPropagationMachine]struct{}{}
	for bi := int64(t.endBehaviorIndex); bi >= 0; bi-- {
		bh := t.footprint.Behaviors[bi]
		fci := bh.Owner
		machine := bh.Machine
		usedMachines[machine] = struct{}{}
		if bh.Aborted {
			continue
		}

		if t.requests.contains(fci) || t.requests.contains(api.SubCmdIdx{fci[0]}) {
			bh.Alive = true
			machine.FramebufferRequest(uint64(bi), t.footprint)
		}

		if bh.Alive || machine.IsAlive(uint64(bi), t.footprint) {
			alivedBehaviorIndices := machine.RecordBehaviorEffects(uint64(bi), t.footprint)
			// TODO: Theoretically, we should re-back-propagation from the alive
			// behaviors other than |bi|.
			for _, aliveBI := range alivedBehaviorIndices {
				if aliveBI < t.endBehaviorIndex+1 {
					livenessBoard[aliveBI] = true
					aliveCommands.insert(t.footprint.Behaviors[aliveBI].Owner)
				}
			}
		}
	}
	for m := range usedMachines {
		m.Clear()
	}
	return livenessBoard, aliveCommands
}
