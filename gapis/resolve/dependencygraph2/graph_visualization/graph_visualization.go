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
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

const (
	EMPTY = "Empty"
)

func createGraphFromDependencyGraph(ctx context.Context, dependencyGraph dependencygraph2.DependencyGraph) (*graph, error) {

	currentGraph := createGraph(0)
	idToBuilder := map[api.ID]api.GraphVisualizationBuilder{}
	commandNameAndIdToNodeId := map[string]int{}

	err := dependencyGraph.ForeachNode(
		func(nodeId dependencygraph2.NodeID, dependencyNode dependencygraph2.Node) error {
			if cmdNode, ok := dependencyNode.(dependencygraph2.CmdNode); ok {
				if len(cmdNode.Index) == 0 {
					return nil
				}
				cmdId := cmdNode.Index[0]
				command := dependencyGraph.GetCommand(api.CmdID(cmdId))
				if !api.CmdID(cmdId).IsReal() {
					return nil
				}

				if graphVisualizationAPI, ok := command.API().(api.GraphVisualizationAPI); ok {
					if _, ok := idToBuilder[command.API().ID()]; !ok {
						idToBuilder[command.API().ID()] = graphVisualizationAPI.GetGraphVisualizationBuilder()
					}
					builder := idToBuilder[command.API().ID()]
					commandNameAndId := fmt.Sprintf("%s_%d", command.CmdName(), cmdId)
					isSubCommand, parentNodeId := false, 0

					label := &api.Label{}
					if len(cmdNode.Index) == 1 {
						label = builder.GetCommandLabel(command, cmdId)
						commandNameAndIdToNodeId[commandNameAndId] = int(nodeId)
					} else if len(cmdNode.Index) > 1 {
						dependencyNodeAccesses := dependencyGraph.GetNodeAccesses(nodeId)
						subCommandName := EMPTY
						if len(dependencyNodeAccesses.InitCmdNodes) > 0 {
							if id, ok := commandNameAndIdToNodeId[commandNameAndId]; ok {
								isSubCommand = true
								parentNodeId = id
							}

							subCmdDependencyNode := dependencyGraph.GetNode(dependencyNodeAccesses.InitCmdNodes[0])
							if subCmdNode, ok := subCmdDependencyNode.(dependencygraph2.CmdNode); ok {
								if len(subCmdNode.Index) == 0 {
									return nil
								}
								subCmdId := subCmdNode.Index[0]
								subCmd := dependencyGraph.GetCommand(api.CmdID(subCmdId))
								subCommandName = subCmd.CmdName()
							}
						}
						label = builder.GetSubCommandLabel(cmdNode.Index, command.CmdName(), cmdId, subCommandName)
					}

					attributes := ""
					for _, parameter := range command.CmdParams() {
						attributes += parameter.Name + " "
					}

					newNode := getNewNode(int(nodeId), label)
					newNode.attributes = attributes
					newNode.isEndOfFrame = cmdNode.CmdFlags.IsEndOfFrame()
					if isSubCommand {
						parentNode := currentGraph.nodeIdToNode[parentNodeId]
						parentNode.addSubCommandNode(newNode)
					}
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

func GetGraphVisualizationFromCapture(ctx context.Context, p *path.Capture, format service.GraphFormat) ([]byte, error) {
	config := dependencygraph2.DependencyGraphConfig{
		SaveNodeAccesses:       true,
		IncludeInitialCommands: true,
	}
	dependencyGraph, err := dependencygraph2.GetDependencyGraph(ctx, p, config)
	if err != nil {
		return []byte{}, err
	}

	currentGraph, err := createGraphFromDependencyGraph(ctx, dependencyGraph)
	if err != nil {
		return []byte{}, err
	}
	currentGraph.assignColorToNodes()
	currentGraph.joinNodesByFrame()
	currentGraph.joinNodesThatDoNotBelongToAnyFrame()

	currentChunkConfig := chunkConfig{
		maximumNumberOfNodesByLevel:   5,
		minimumNumberOfNodesToBeChunk: 2,
	}
	currentGraph.makeChunks(currentChunkConfig)

	output := []byte{}
	if format == service.GraphFormat_PBTXT {
		output = currentGraph.getGraphInPbtxtFormat()
	} else if format == service.GraphFormat_DOT {
		output = currentGraph.getGraphInDotFormat()
	}

	return output, err
}
