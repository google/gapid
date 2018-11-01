package graph_visualization

import (
	"context"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
)

func GetGraphVisualizationFileFromCapture(ctx context.Context, p *capture.Capture) (string, error) {

	log.I(ctx, "Working on GetGraphDotFileFromCapture")
	config := dependencygraph2.DependencyGraphConfig{}
	dependencyGraph, err := dependencygraph2.BuildDependencyGraph(ctx, config, p, []api.Cmd{}, interval.U64RangeList{})
	_ = dependencyGraph

	return "OutputFile", err
}
