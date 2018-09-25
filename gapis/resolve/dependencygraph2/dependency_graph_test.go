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
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay/builder"
)

type TestCmd struct{}

func (TestCmd) API() api.API                                                       { return nil }
func (TestCmd) Caller() api.CmdID                                                  { return 0 }
func (TestCmd) SetCaller(api.CmdID)                                                {}
func (TestCmd) Thread() uint64                                                     { return 0 }
func (TestCmd) SetThread(uint64)                                                   {}
func (TestCmd) CmdName() string                                                    { return "TestCmd" }
func (TestCmd) CmdFlags(context.Context, api.CmdID, *api.GlobalState) api.CmdFlags { return 0 }
func (TestCmd) Extras() *api.CmdExtras                                             { return &api.CmdExtras{} }
func (TestCmd) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder, api.StateWatcher) error {
	return nil
}
func (TestCmd) Clone(arena.Arena) api.Cmd { return TestCmd{} }
func (TestCmd) Alive() bool               { return false }
func (TestCmd) CmdParams() api.Properties { return api.Properties{} }
func (TestCmd) CmdResult() *api.Property  { return nil }

type TestRef struct {
	refID api.RefID
}

func (ref TestRef) RefID() api.RefID {
	return ref.refID
}

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

type FIELD_B_C struct{}

func (FIELD_B_C) ClassName() string {
	return "B"
}
func (FIELD_B_C) FieldName() string {
	return "C"
}

func TestBuilder(t *testing.T) {
	ctx := context.Background()
	// cmds := make([]api.Cmd, 6)
	c := &capture.Capture{
		Name: "test",
		Header: &capture.Header{
			ABI: device.LinuxX86_64,
		},
		Commands:     []api.Cmd{TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}},
		InitialState: &capture.InitialState{},
	}
	b := newDependencyGraphBuilder(ctx, DependencyGraphConfig{}, c, []api.Cmd{})
	eg := newDependencyGraph(ctx, DependencyGraphConfig{}, c, []api.Cmd{})
	// root := api.RefID(1)
	// b.AddRefRoot("R", root)
	getNodeID := func(cmdID uint64) NodeID {
		return b.graph.GetNodeID(CmdNode{api.SubCmdIdx{cmdID}})
	}
	refA, refB, refC := newTestRef(), newTestRef(), newTestRef()
	b.OnBeginCmd(ctx, 0, TestCmd{})
	b.OnSet(ctx, refA, api.FieldFragment{FIELD_A_B{}}, api.NilReference{}, refB)
	b.OnEndCmd(ctx, 0, TestCmd{})
	b.OnBeginCmd(ctx, 1, TestCmd{})
	b.OnSet(ctx, refB, api.FieldFragment{FIELD_B_C{}}, api.NilReference{}, refC)
	b.OnEndCmd(ctx, 1, TestCmd{})

	b.OnBeginCmd(ctx, 2, TestCmd{})
	b.OnGet(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB)
	b.OnGet(ctx, refB, api.FieldFragment{FIELD_B_C{}}, refC)
	b.OnEndCmd(ctx, 2, TestCmd{})
	eg.setDependencies(getNodeID(2), []NodeID{getNodeID(0), getNodeID(1)})

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading struct should depend on writes to fields after last write to struct (0)").That(
		b.graph).DeepEquals(eg)

	b.OnBeginCmd(ctx, 3, TestCmd{})
	b.OnGet(ctx, refB, api.FieldFragment{FIELD_B_C{}}, refC)
	b.OnEndCmd(ctx, 3, TestCmd{})
	eg.setDependencies(getNodeID(3), []NodeID{getNodeID(1)})

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should not depend on write to struct before last write to field").That(
		b.graph).DeepEquals(eg)

	b.OnBeginCmd(ctx, 4, TestCmd{})
	b.OnSet(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB, refB)
	b.OnEndCmd(ctx, 4, TestCmd{})

	b.OnBeginCmd(ctx, 5, TestCmd{})
	b.OnGet(ctx, refA, api.FieldFragment{FIELD_A_B{}}, refB)
	b.OnEndCmd(ctx, 5, TestCmd{})
	eg.setDependencies(getNodeID(5), []NodeID{getNodeID(4)})

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should only depend on write to struct after list write to field").That(
		b.graph).DeepEquals(eg)
}
