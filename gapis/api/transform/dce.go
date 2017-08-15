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
	dCECounter         = benchmark.GlobalCounters.Duration("DCE")
	dCECmdDeadCounter  = benchmark.GlobalCounters.Integer("DCE.cmd.dead")
	dCECmdLiveCounter  = benchmark.GlobalCounters.Integer("DCE.cmd.live")
	dCEDrawDeadCounter = benchmark.GlobalCounters.Integer("DCE.draw.dead")
	dCEDrawLiveCounter = benchmark.GlobalCounters.Integer("DCE.draw.live")
	dCEDataDeadCounter = benchmark.GlobalCounters.Integer("DCE.data.dead")
	dCEDataLiveCounter = benchmark.GlobalCounters.Integer("DCE.data.live")
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

func (s *commandIndicesSet) contains(
	fci api.SubCmdIdx) bool {
	v := s.Value(fci)
	if v != nil {
		if bv, ok := v.(bool); ok {
			return bv
		}
		return false
	}
	return false
}

type DCE struct {
	footprint         *dependencygraph.Footprint
	endBehaviourIndex uint64
	requests          *commandIndicesSet
}

func NewDCE(ctx context.Context, footprint *dependencygraph.Footprint) *DCE {
	return &DCE{
		footprint: footprint,
		requests:  newCommandIndicesSet(),
	}
}

func (t *DCE) Request(ctx context.Context, fci api.SubCmdIdx) {
	t.requests.insert(fci)
	bi := t.footprint.GetBehaviourIndex(ctx, fci)
	if bi > t.endBehaviourIndex {
		t.endBehaviourIndex = bi
	}
}

func (t *DCE) Transform(ctx context.Context, id api.CmdID, c api.Cmd,
	out Writer) {
	panic(fmt.Errorf("This transform does not accept input atoms"))
}

func (t *DCE) Flush(ctx context.Context, out Writer) {
	t0 := dCECounter.Start()
	livenessBoard, aliveCmds := t.backPropagate(ctx)
	dCECounter.Stop(t0)
	flushedCommands := newCommandIndicesSet()

	numCmd, numDead, numDeadDraws, numLive, numLiveDraws := 0, 0, 0, 0, 0
	deadMem, liveMem := uint64(0), uint64(0)

	for bi := uint64(0); bi <= t.endBehaviourIndex; bi++ {
		bh := t.footprint.Behaviours[bi]
		fci := bh.BelongTo
		if livenessBoard[bi] && len(fci) == 1 && !flushedCommands.contains(fci) {
			flushedCommands.insert(fci)
			aliveCmd := t.footprint.Commands[fci[0]]

			// Logging the DCE result of alive commands
			numCmd += 1
			numLive += 1
			if aliveCmd.CmdFlags().IsDrawCall() {
				numLiveDraws += 1
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
				numCmd += 1
				numDead += 1
				if deadCmd.CmdFlags().IsDrawCall() {
					numDeadDraws += 1
				}
				if e := deadCmd.Extras(); e != nil && e.Observations() != nil {
					for _, r := range e.Observations().Reads {
						deadMem += r.Range.Size
					}
				}
			}
		}
	}
	dCECmdDeadCounter.AddInt64(int64(numDead))
	dCECmdLiveCounter.AddInt64(int64(numLive))
	dCEDrawDeadCounter.AddInt64(int64(numDeadDraws))
	dCEDrawLiveCounter.AddInt64(int64(numLiveDraws))
	dCEDataDeadCounter.AddInt64(int64(deadMem))
	dCEDataLiveCounter.AddInt64(int64(liveMem))
	log.D(ctx, "DCE: dead: %v%% %v cmds %v MB %v draws, live: %v%% %v cmds %v MB %v draws",
		100*numDead/numCmd, numDead, deadMem/1024/1024, numDeadDraws,
		100*numLive/numCmd, numLive, liveMem/1024/1024, numLiveDraws)
}

func (t *DCE) backPropagate(ctx context.Context) (
	[]bool, *commandIndicesSet) {
	livenessBoard := make([]bool, t.endBehaviourIndex+1)
	aliveCommands := newCommandIndicesSet()
	usedMachines := map[dependencygraph.BackPropagationMachine]struct{}{}
	for bi := int64(t.endBehaviourIndex); bi >= 0; bi-- {
		bh := t.footprint.Behaviours[bi]
		fci := bh.BelongTo
		machine := bh.Machine
		usedMachines[machine] = struct{}{}
		if bh.Aborted {
			continue
		}
		if t.requests.contains(fci) ||
			t.requests.contains(api.SubCmdIdx{fci[0]}) ||
			bh.Alive || machine.IsAlive(uint64(bi), t.footprint) {
			alivedBehaviourIndices := machine.RecordBehaviourEffects(uint64(bi), t.footprint)
			// TODO: Theoretically, we should re-back-propagation from the alive
			// behaviours other than |bi|.
			for _, aliveBI := range alivedBehaviourIndices {
				if aliveBI < t.endBehaviourIndex+1 {
					livenessBoard[aliveBI] = true
					aliveCommands.insert(t.footprint.Behaviours[aliveBI].BelongTo)
				}
			}
		}
	}
	for m, _ := range usedMachines {
		m.Clear()
	}
	return livenessBoard, aliveCommands
}
