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

func TestFootprintAddAndGetBehavior(t *testing.T) {
	ctx := log.Testing(t)
	ft := dependencygraph.NewEmptyFootprint(ctx)
	behaviors := []*dependencygraph.Behavior{
		dependencygraph.NewBehavior(api.SubCmdIdx{0}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{1}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{2}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{3}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 3}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 4}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 5, 6, 7}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 5, 6, 8}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 6}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{5}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 7}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4, 1, 2, 8}, &dummyMachine{}),
		dependencygraph.NewBehavior(api.SubCmdIdx{4}, &dummyMachine{}), // overwrites the previous one
		dependencygraph.NewBehavior(api.SubCmdIdx{6}, &dummyMachine{}),
	}
	for _, b := range behaviors {
		ft.AddBehavior(ctx, b)
	}
	for bi, b := range behaviors {
		i := ft.BehaviorIndex(ctx, b.Owner)
		if bi == 4 {
			assert.For(ctx, "Behavior Index should be %v", 13).That(
				i).Equals(uint64(13))
		} else {
			assert.For(ctx, "Behavior Index should be %v", bi).That(
				i).Equals(uint64(bi))
		}
	}
}
