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
	"sort"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/path"
)

type TestCmd struct{}

func (TestCmd) API() api.API           { return nil }
func (TestCmd) Thread() uint64         { return 0 }
func (TestCmd) SetThread(uint64)       {}
func (TestCmd) CmdName() string        { return "TestCmd" }
func (TestCmd) CmdFlags() api.CmdFlags { return 0 }
func (TestCmd) Extras() *api.CmdExtras { return &api.CmdExtras{} }
func (TestCmd) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder, api.StateWatcher) error {
	return nil
}
func (TestCmd) Clone(arena.Arena) api.Cmd     { return TestCmd{} }
func (TestCmd) Alive() bool                   { return false }
func (TestCmd) CmdParams() api.Properties     { return api.Properties{} }
func (TestCmd) CmdResult() *api.Property      { return nil }
func (TestCmd) Terminated() bool              { return true }
func (TestCmd) SetTerminated(terminated bool) {}

type TestRef struct {
	refID api.RefID
}

var _ api.Reference = TestRef{}

func (ref TestRef) RefID() api.RefID {
	return ref.refID
}

func (TestRef) NewFragmentMap() api.FragmentMap {
	return api.NewSparseFragmentMap()
}

// Ensure that TestRef implements api.State so that fragments of TestRef
// are considered part of the global state--otherwise these fragments would be
// considered local variables and ignored by the dependency graph builder.
var _ api.State = TestRef{}

func (TestRef) API() api.API {
	return nil
}

func (r TestRef) Clone(arena.Arena) api.State { return TestRef{r.refID} }

func (TestRef) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return nil, nil
}

func (TestRef) SetupInitialState(ctx context.Context, state *api.GlobalState) {}

func newTestRef() TestRef {
	return TestRef{api.NewRefID()}
}

type FIELD_A_B struct{}

func (FIELD_A_B) ClassName() string {
	return "A"
}
func (FIELD_A_B) FieldName() string {
	return "B"
}
func (FIELD_A_B) FieldIndex() int {
	return 0
}

type FIELD_B_C struct{}

func (FIELD_B_C) ClassName() string {
	return "B"
}
func (FIELD_B_C) FieldName() string {
	return "C"
}
func (FIELD_B_C) FieldIndex() int {
	return 0
}

func TestBuilder(t *testing.T) {
	ctx := log.Testing(t)
	// cmds := make([]api.Cmd, 6)
	header := &capture.Header{ABI: device.LinuxX86_64}
	cmds := []api.Cmd{TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}}
	c, err := capture.NewGraphicsCapture(ctx, arena.New(), "test", header, &capture.InitialState{}, cmds)
	if !assert.For(ctx, "capture.NewGraphicsCapture").ThatError(err).Succeeded() {
		return
	}
	b := newDependencyGraphBuilder(ctx, DependencyGraphConfig{}, c, []api.Cmd{}, c.NewState(ctx))
	eg := newDependencyGraph(ctx, DependencyGraphConfig{}, c, []api.Cmd{}, []Node{})
	getNodeID := func(cmdID uint64) NodeID {
		return b.graphBuilder.GetCmdNodeID(api.CmdID(cmdID), api.SubCmdIdx{})
	}
	addNode := func(cmdID uint64) NodeID {
		return eg.addNode(CmdNode{api.SubCmdIdx{uint64(cmdID)}, 0})
	}
	refA, refB, refC := newTestRef(), newTestRef(), newTestRef()
	b.OnBeginCmd(ctx, 0, TestCmd{})
	b.OnWriteFrag(ctx, refA, api.FieldFragment{FIELD_A_B{}}, api.NilReference{}, refB, true)
	b.OnEndCmd(ctx, 0, TestCmd{})
	addNode(0)

	b.OnBeginCmd(ctx, 1, TestCmd{})
	b.OnWriteFrag(ctx, refB, api.FieldFragment{FIELD_B_C{}}, api.NilReference{}, refC, true)
	b.OnEndCmd(ctx, 1, TestCmd{})
	addNode(1)

	b.OnBeginCmd(ctx, 2, TestCmd{})
	b.OnReadFrag(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB, true)
	b.OnReadFrag(ctx, refB, api.FieldFragment{FIELD_B_C{}}, refC, true)
	b.OnEndCmd(ctx, 2, TestCmd{})
	addNode(2)
	eg.setDependencies(getNodeID(2), []NodeID{getNodeID(0), getNodeID(1)})

	// Sort the dependencies here. They are returned by iterating over
	// a map. Their order has no functional effect, just makes the
	// test deterministic.
	for i := range b.graphBuilder.GetGraph().dependenciesFrom {
		sort.Sort(&NodeIDSorter{b.graphBuilder.GetGraph().dependenciesFrom[i]})
	}

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading struct should depend on writes to fields after last write to struct (0)").That(
		b.graphBuilder.GetGraph().dependenciesFrom).DeepEquals(eg.dependenciesFrom)

	b.OnBeginCmd(ctx, 3, TestCmd{})
	b.OnReadFrag(ctx, refB, api.FieldFragment{FIELD_B_C{}}, refC, true)
	b.OnEndCmd(ctx, 3, TestCmd{})
	addNode(3)
	eg.setDependencies(getNodeID(3), []NodeID{getNodeID(1)})

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should not depend on write to struct before last write to field").That(
		b.graphBuilder.GetGraph()).DeepEquals(eg)

	b.OnBeginCmd(ctx, 4, TestCmd{})
	b.OnWriteFrag(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB, refB, true)
	b.OnEndCmd(ctx, 4, TestCmd{})
	addNode(4)

	b.OnBeginCmd(ctx, 5, TestCmd{})
	b.OnReadFrag(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB, true)
	b.OnEndCmd(ctx, 5, TestCmd{})
	addNode(5)
	eg.setDependencies(getNodeID(5), []NodeID{getNodeID(4)})

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should only depend on write to struct after list write to field").That(
		b.graphBuilder.GetGraph()).DeepEquals(eg)
}
