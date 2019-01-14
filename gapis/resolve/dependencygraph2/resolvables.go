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

	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service/path"
)

func GetDependencyGraph(ctx context.Context, c *path.Capture, config DependencyGraphConfig) (DependencyGraph, error) {
	obj, err := database.Build(ctx, &DependencyGraph2Resolvable{
		Capture:                c,
		IncludeInitialCommands: config.IncludeInitialCommands,
		MergeSubCmdNodes:       config.MergeSubCmdNodes,
		ReverseDependencies:    config.ReverseDependencies,
		SaveNodeAccesses:       config.SaveNodeAccesses,
	})
	if err != nil {
		return nil, err
	}
	return obj.(DependencyGraph), nil
}

func TryGetDependencyGraph(ctx context.Context, c *path.Capture, config DependencyGraphConfig) (DependencyGraph, error) {
	obj, err := database.GetOrBuild(ctx, &DependencyGraph2Resolvable{
		Capture:                c,
		IncludeInitialCommands: config.IncludeInitialCommands,
		MergeSubCmdNodes:       config.MergeSubCmdNodes,
	})
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(DependencyGraph), nil
}

func (r *DependencyGraph2Resolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	initialCmds := []api.Cmd{}
	initialRanges := interval.U64RangeList{}
	if r.IncludeInitialCommands {
		initialCmds, initialRanges, err = initialcmds.InitialCommands(ctx, r.Capture)
		if err != nil {
			return nil, err
		}
	}
	config := DependencyGraphConfig{
		IncludeInitialCommands: r.IncludeInitialCommands,
		MergeSubCmdNodes:       r.MergeSubCmdNodes,
		ReverseDependencies:    r.ReverseDependencies,
		SaveNodeAccesses:       r.SaveNodeAccesses,
	}
	return BuildDependencyGraph(ctx, config, c, initialCmds, initialRanges)
}
