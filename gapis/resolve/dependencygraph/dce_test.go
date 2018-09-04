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

package dependencygraph_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

type dummyDefUseVar struct {
	b *dependencygraph.Behavior
}

func (v *dummyDefUseVar) GetDefBehavior() *dependencygraph.Behavior {
	return v.b
}

func (v *dummyDefUseVar) SetDefBehavior(b *dependencygraph.Behavior) {
	v.b = b
}

func TestDCE(t *testing.T) {
	ctx := log.Testing(t)
	ft := dependencygraph.NewEmptyFootprint(ctx)
	nodes := map[int]*dummyDefUseVar{}

	behave := func(fci api.SubCmdIdx,
		reads, writes []int) {
		b := dependencygraph.NewBehavior(fci)
		for _, r := range reads {
			if rv, ok := nodes[r]; ok {
				b.Read(rv)
			}
		}
		for _, w := range writes {
			if _, ok := nodes[w]; !ok {
				nodes[w] = &dummyDefUseVar{}
			}
			b.Write(nodes[w])
		}
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
	behave([]uint64{0}, []int{}, []int{1, 2, 3})
	behave([]uint64{1}, []int{}, []int{2, 3})
	behave([]uint64{2}, []int{}, []int{4})
	behave([]uint64{3, 0, 0, 0}, []int{2}, []int{5})
	behave([]uint64{3, 0, 0, 1}, []int{3}, []int{6})
	behave([]uint64{3, 0, 0, 2}, []int{4}, []int{7})
	behave([]uint64{3, 0, 0, 3}, []int{5, 6, 7}, []int{8})
	behave([]uint64{3, 0, 0, 4}, []int{8}, []int{9})
	behave([]uint64{3, 0, 1, 0}, []int{8, 9}, []int{10})
	behave([]uint64{4}, []int{10}, []int{})

	dce := dependencygraph.NewDCE(ctx, ft)
	expectedLiveness := func(aliveCommands *dependencygraph.CommandIndicesSet, fci api.SubCmdIdx, expected bool) {
		assert.For(ctx, "Liveness of command with full command index: %v should be %v",
			fci, expected).That(aliveCommands.Contains(fci)).Equals(expected)
	}

	// Case: Request: 4, Dead: 0
	dce.Request(ctx, []uint64{4})
	_, alived := dce.BackPropagate(ctx)
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

	dce = dependencygraph.NewDCE(ctx, ft)

	// Case: Request: 3-0-0-1, Dead: 3-0-0-0, 2, 0, and all after 3-0-0-1
	dce.Request(ctx, []uint64{3, 0, 0, 1})
	_, alived = dce.BackPropagate(ctx)
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
