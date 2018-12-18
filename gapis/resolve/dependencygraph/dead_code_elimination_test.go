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
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

func TestLivenessTree(t *testing.T) {
	ctx := log.Testing(t)
	ctx = database.Put(ctx, database.NewInMemory(ctx))

	//
	//          root
	//         /    \
	//     child1  child2
	//      /  \
	// childA  childB
	//
	root := dependencygraph.StateAddress(1)
	child1 := dependencygraph.StateAddress(2)
	child2 := dependencygraph.StateAddress(3)
	childA := dependencygraph.StateAddress(4)
	childB := dependencygraph.StateAddress(5)
	tree := dependencygraph.NewLivenessTree(map[dependencygraph.StateAddress]dependencygraph.StateAddress{
		dependencygraph.NullStateAddress: dependencygraph.NullStateAddress,
		root:                             dependencygraph.NullStateAddress,
		child1:                           root,
		child2:                           root,
		childA:                           child1,
		childB:                           child1,
	})

	tree.MarkLive(child1)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(true)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(true)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(false)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(true)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(true)

	tree.MarkDead(root)
	tree.MarkLive(child1)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(true)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(true)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(false)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(true)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(true)

	tree.MarkLive(root)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(true)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(true)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(true)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(true)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(true)

	tree.MarkDead(child1)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(true)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(false)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(true)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(false)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(false)

	tree.MarkDead(root)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(false)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(false)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(false)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(false)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(false)

	tree.MarkLive(childA)
	assert.For(ctx, "IsLive(root)").That(tree.IsLive(root)).Equals(true)
	assert.For(ctx, "IsLive(child1)").That(tree.IsLive(child1)).Equals(true)
	assert.For(ctx, "IsLive(child2)").That(tree.IsLive(child2)).Equals(false)
	assert.For(ctx, "IsLive(childA)").That(tree.IsLive(childA)).Equals(true)
	assert.For(ctx, "IsLive(childB)").That(tree.IsLive(childB)).Equals(false)
}
