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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

type dummyDefUseVar uint64

func (dummyDefUseVar) DefUseVariable() {}

type dummyMachine struct {
	undefined map[dummyDefUseVar]struct{}
}

func (m *dummyMachine) IsAlive(behaviorIndex uint64,
	ft *dependencygraph.Footprint) bool {
	bh := ft.Behaviors[behaviorIndex]
	for _, w := range bh.Writes {
		if u, ok := w.(dummyDefUseVar); ok {
			_, contains := m.undefined[u]
			if contains {
				return true
			}
		}
	}
	return false
}

func (m *dummyMachine) RecordBehaviorEffects(behaviorIndex uint64,
	ft *dependencygraph.Footprint) []uint64 {
	alive := []uint64{behaviorIndex}
	bh := ft.Behaviors[behaviorIndex]
	for _, w := range bh.Writes {
		if u, ok := w.(dummyDefUseVar); ok {
			delete(m.undefined, u)
		}
	}
	for _, r := range bh.Reads {
		if u, ok := r.(dummyDefUseVar); ok {
			m.undefined[u] = struct{}{}
		}
	}
	return alive
}

func (m *dummyMachine) Clear() {
	m.undefined = map[dummyDefUseVar]struct{}{}
}

func (m *dummyMachine) FramebufferRequest(uint64, *dependencygraph.Footprint) {}

func TestDCE(t *testing.T) {
	ctx := log.Testing(t)
	ft := dependencygraph.NewEmptyFootprint(ctx)
	machine := &dummyMachine{undefined: map[dummyDefUseVar]struct{}{}}

	behave := func(fci api.SubCmdIdx,
		reads, writes []dummyDefUseVar) {
		b := dependencygraph.NewBehavior(fci, machine)
		for _, r := range reads {
			b.Read(r)
		}
		for _, w := range writes {
			b.Write(w)
		}
		b.Machine = machine
		ft.AddBehavior(ctx, b)
	}

	// FullCommandIndex: Behavior Reads, Writes
	// 0: R[], W[1, 2, 3]
	// 1: R[], W[2, 3]
	// 2: R[], W[4]
	// 3-0-0-0: R[2], W[5]
	// 3-0-0-1: R[3], W[6]
	// 3-0-0-2: R[4], W[7]
	// 3-0-0-3: R[5, 6, 7], W[8]
	// 3-0-0-4: R[8], W[9]
	// 3-0-1-0: R[8, 9], W[10]
	// 4: R[10], W[]
	behave([]uint64{0}, []dummyDefUseVar{}, []dummyDefUseVar{1, 2, 3})
	behave([]uint64{1}, []dummyDefUseVar{}, []dummyDefUseVar{2, 3})
	behave([]uint64{2}, []dummyDefUseVar{}, []dummyDefUseVar{4})
	behave([]uint64{3, 0, 0, 0}, []dummyDefUseVar{2}, []dummyDefUseVar{5})
	behave([]uint64{3, 0, 0, 1}, []dummyDefUseVar{3}, []dummyDefUseVar{6})
	behave([]uint64{3, 0, 0, 2}, []dummyDefUseVar{4}, []dummyDefUseVar{7})
	behave([]uint64{3, 0, 0, 3}, []dummyDefUseVar{5, 6, 7}, []dummyDefUseVar{8})
	behave([]uint64{3, 0, 0, 4}, []dummyDefUseVar{8}, []dummyDefUseVar{9})
	behave([]uint64{3, 0, 1, 0}, []dummyDefUseVar{8, 9}, []dummyDefUseVar{10})
	behave([]uint64{4}, []dummyDefUseVar{10}, []dummyDefUseVar{})

	dce := NewDCE(ctx, ft)
	expectedLiveness := func(alivedCommands *commandIndicesSet, fci api.SubCmdIdx, expected bool) {
		assert.To(t).For("Liveness of command with full command index: %v should be %v",
			fci, expected).That(alivedCommands.contains(fci)).Equals(expected)
	}

	// Case: Request: 4, Dead: 0
	dce.Request(ctx, []uint64{4})
	_, alived := dce.backPropagate(ctx)
	expectedLiveness(alived, []uint64{0}, false)
	expectedLiveness(alived, []uint64{1}, true)
	expectedLiveness(alived, []uint64{2}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 0}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 1}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 2}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 3}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 4}, true)
	expectedLiveness(alived, []uint64{3, 0, 1, 0}, true)
	expectedLiveness(alived, []uint64{4}, true)

	dce.requests = newCommandIndicesSet()
	dce.endBehaviorIndex = uint64(0)
	dce.requests.count = uint64(0)

	// Case: Request: 3-0-0-1, Dead: 3-0-0-0, 2, 0, and all after 3-0-0-1
	dce.Request(ctx, []uint64{3, 0, 0, 1})
	_, alived = dce.backPropagate(ctx)
	expectedLiveness(alived, []uint64{0}, false)
	expectedLiveness(alived, []uint64{1}, true)
	expectedLiveness(alived, []uint64{2}, false)
	expectedLiveness(alived, []uint64{3, 0, 0, 0}, false)
	expectedLiveness(alived, []uint64{3, 0, 0, 1}, true)
	expectedLiveness(alived, []uint64{3, 0, 0, 2}, false)
	expectedLiveness(alived, []uint64{3, 0, 0, 3}, false)
	expectedLiveness(alived, []uint64{3, 0, 0, 4}, false)
	expectedLiveness(alived, []uint64{3, 0, 1, 0}, false)
	expectedLiveness(alived, []uint64{4}, false)
}
