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
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/state_path"
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
func (TestCmd) Mutate(context.Context, api.CmdID, *api.GlobalState, *builder.Builder) error {
	return nil
}

func TestBuilder(t *testing.T) {
	ctx := context.Background()
	// cmds := make([]api.Cmd, 6)
	c := &capture.Capture{
		Name: "test",
		Header: &capture.Header{
			Abi: device.LinuxX86_64,
		},
		Commands:     []api.Cmd{TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}, TestCmd{}},
		InitialState: &capture.InitialState{},
	}
	b := NewDependencyGraphBuilder(ctx, c, DependencyGraphConfig{}).(*dependencyGraphBuilder)
	int := NewDependencyGraphInterceptor(ctx, c, b)
	root := api.RefID(1)
	// b.AddRefRoot("R", root)
	eg := newDependencyGraph(ctx, c, DependencyGraphConfig{})
	getNodeID := func(cmdID uint64) NodeID {
		return eg.GetNodeID(CmdNode{api.SubCmdIdx{cmdID}})
	}
	aPath := state_path.RootPath(root).Field("A")
	abPath := state_path.RootPath(root).Field("A").Field("B")
	// aVar := FieldVar{RootVar{"T", root}, "A"}
	// abVar := FieldVar{FieldVar{RootVar{"T", root}, "A"}, "B"}
	int.BeginCmd(ctx, 0, TestCmd{})
	int.WritePath(ctx, aPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 0, TestCmd{})
	int.BeginCmd(ctx, 1, TestCmd{})
	int.WritePath(ctx, abPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 1, TestCmd{})

	int.BeginCmd(ctx, 2, TestCmd{})
	int.ReadPath(ctx, aPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 2, TestCmd{})
	eg.addDependency(getNodeID(2), getNodeID(0))
	eg.addDependency(getNodeID(2), getNodeID(1))

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading struct should depend on writes to fields after last write to struct").That(
		b.graph.dependenciesFrom).DeepEquals(eg.dependenciesFrom)

	int.BeginCmd(ctx, 3, TestCmd{})
	int.ReadPath(ctx, abPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 3, TestCmd{})
	eg.addDependency(getNodeID(3), getNodeID(1))

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should not depend on write to struct before last write to field").That(
		b.graph).DeepEquals(eg)

	int.BeginCmd(ctx, 4, TestCmd{})
	int.WritePath(ctx, aPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 4, TestCmd{})

	int.BeginCmd(ctx, 5, TestCmd{})
	int.ReadPath(ctx, abPath, api.NilRefID, api.SubCmdIdx{})
	int.EndCmd(ctx, 5, TestCmd{})
	eg.addDependency(getNodeID(5), getNodeID(4))

	// eg.Paths = b.graph.Paths
	assert.To(t).For("Reading field should only depend on write to struct after list write to field").That(
		b.graph).DeepEquals(eg)
}
