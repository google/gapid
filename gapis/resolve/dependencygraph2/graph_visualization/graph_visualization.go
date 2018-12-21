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

package graph_visualization

import (
	"context"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
)

func GetGraphVisualizationFromCapture(ctx context.Context, p *capture.Capture) ([]byte, error) {

	config := dependencygraph2.DependencyGraphConfig{}
	dependencyGraph, err := dependencygraph2.BuildDependencyGraph(ctx, config, p, []api.Cmd{}, interval.U64RangeList{})
	_ = dependencyGraph

	return []byte("OutputFile"), err
}
