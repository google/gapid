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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

func TestFootprintAddAndGetBehaviour(t *testing.T) {
	ctx := log.Testing(t)
	ft := NewEmptyFootprint(ctx)
	behaviours := []*Behaviour{
		NewBehaviour(api.SubCmdIdx{0}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{1}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{2}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{3}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 3}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 4}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 5, 6, 7}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 5, 6, 8}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 6}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{5}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 7}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4, 1, 2, 8}, &dummyMachine{}),
		NewBehaviour(api.SubCmdIdx{4}, &dummyMachine{}), // overwrites the previous one
		NewBehaviour(api.SubCmdIdx{6}, &dummyMachine{}),
	}
	for _, b := range behaviours {
		ft.AddBehaviour(ctx, b)
	}
	for bi, b := range behaviours {
		i := ft.GetBehaviourIndex(ctx, b.BelongTo)
		if bi == 4 {
			assert.To(t).For("Behaviour Index should be %v", 13).That(
				i).Equals(uint64(13))
		} else {
			assert.To(t).For("Behaviour Index should be %v", bi).That(
				i).Equals(uint64(bi))
		}
	}
}
