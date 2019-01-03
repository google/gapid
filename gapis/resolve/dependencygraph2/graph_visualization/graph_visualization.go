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
	"fmt"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/service/path"
)

func createGraphFromDependencyGraph(dependencyGraph dependencygraph2.DependencyGraph) (*graph, error) {

	currentGraph := createGraph(0)
	currentHierarchy := &api.Hierarchy{}

	err := dependencyGraph.ForeachNode(
		func(nodeId dependencygraph2.NodeID, dependencyNode dependencygraph2.Node) error {
			if cmdNode, ok := dependencyNode.(dependencygraph2.CmdNode); ok {
				cmdNodeId := cmdNode.Index[0]
				command := dependencyGraph.GetCommand(api.CmdID(cmdNodeId))
				commandName := command.CmdName()

				if graphVisualizationAPI, ok := command.API().(api.GraphVisualizationAPI); ok {
					label := graphVisualizationAPI.GetCommandLabel(currentHierarchy, command)
					label += fmt.Sprintf("%s%d", commandName, cmdNodeId)
					label += graphVisualizationAPI.GetSubCommandLabel(cmdNode.Index)

					attributes := ""
					for _, parameter := range command.CmdParams() {
						attributes += parameter.Name + " "
					}

					newNode := getNewNode(int(nodeId), label)
					newNode.name = commandName
					newNode.attributes = attributes
					currentGraph.addNode(newNode)
				}
			}
			return nil
		})

	if err != nil {
		return currentGraph, err
	}

	err = dependencyGraph.ForeachDependency(
		func(idSource, idSink dependencygraph2.NodeID) error {
			currentGraph.addEdgeBetweenNodesById(int(idSource), int(idSink))
			return nil
		})

	return currentGraph, err
}

func GetGraphVisualizationFromCapture(ctx context.Context, p *path.Capture) ([]byte, error) {
	config := dependencygraph2.DependencyGraphConfig{
		SaveNodeAccesses:       true,
		IncludeInitialCommands: true,
	}
	dependencyGraph, err := dependencygraph2.GetDependencyGraph(ctx, p, config)
	if err != nil {
		return []byte{}, err
	}

	currentGraph, err := createGraphFromDependencyGraph(dependencyGraph)
	currentGraph.removeNodesWithZeroDegree()
	output := currentGraph.getGraphInPbtxtFormat()
	return []byte(output), err
}
