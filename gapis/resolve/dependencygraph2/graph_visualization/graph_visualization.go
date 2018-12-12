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
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
)

const (
	VK_BEGIN_COMMAND_BUFFER    = "vkBeginCommandBuffer"
	VK_CMD_BEGIN_RENDER_PASS   = "vkCmdBeginRenderPass"
	VK_CMD_NEXT_SUBPASS        = "vkCmdNextSubpass"
	VK_COMMAND_BUFFER          = "vkCommandBuffer"
	VK_RENDER_PASS             = "vkRenderPass"
	VK_SUBPASS                 = "vkSubpass"
	VK_END_COMMAND_BUFFER      = "vkEndCommandBuffer"
	VK_CMD_END_RENDER_PASS     = "vkCmdEndRenderPass"
	VK_CMD_DRAW_INDEXED        = "vkCmdDrawIndexed"
	VK_CMD_DRAW                = "vkCmdDraw"
	COMMAND_BUFFER             = "commandBuffer"
	MAXIMUM_LEVEL_IN_HIERARCHY = 10
)

var (
	beginCommands = map[string]int{
		VK_BEGIN_COMMAND_BUFFER:  0,
		VK_CMD_BEGIN_RENDER_PASS: 1,
		VK_CMD_NEXT_SUBPASS:      2,
	}
	listOfBeginCommands = []string{
		VK_BEGIN_COMMAND_BUFFER,
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_NEXT_SUBPASS,
	}
	listOfCommandNames = []string{
		VK_COMMAND_BUFFER,
		VK_RENDER_PASS,
		VK_SUBPASS,
	}
	endCommands = map[string]int{
		VK_END_COMMAND_BUFFER:  0,
		VK_CMD_END_RENDER_PASS: 1,
		VK_CMD_NEXT_SUBPASS:    2,
	}
	commandsInsideRenderScope = map[string]struct{}{
		VK_CMD_DRAW_INDEXED: struct{}{},
		VK_CMD_NEXT_SUBPASS: struct{}{},
		VK_CMD_DRAW:         struct{}{},
	}
)

type Hierarchy struct {
	levelId      [MAXIMUM_LEVEL_IN_HIERARCHY]int
	currentId    int
	currentLevel int
}

func getCommandBuffer(command api.Cmd) string {
	parameters := command.CmdParams()
	for _, parameter := range parameters {
		if parameter.Name == COMMAND_BUFFER {
			commandBuffer := parameter.Name + fmt.Sprintf("%d", parameter.Get()) + "/"
			return commandBuffer
		}
	}
	return ""
}

func getCommandLabel(currentHierarchy *Hierarchy, command api.Cmd) string {
	commandName := command.CmdName()
	isEndCommand := false
	if currentLevel, ok := beginCommands[commandName]; ok && currentLevel <= currentHierarchy.currentLevel {
		currentHierarchy.levelId[currentLevel] = currentHierarchy.currentId
		currentHierarchy.currentId++
		currentHierarchy.currentLevel = currentLevel + 1
	} else {
		if currentLevel, ok := endCommands[commandName]; ok && currentLevel <= currentHierarchy.currentLevel {
			currentHierarchy.currentLevel = currentLevel + 1
			isEndCommand = true
		}
	}

	label := "\""
	for i := 0; i < currentHierarchy.currentLevel; i++ {
		if i == 0 {
			label += getCommandBuffer(command)
		} else {
			label += fmt.Sprintf("%s%d/", listOfCommandNames[i], currentHierarchy.levelId[i])
		}
	}
	if isEndCommand {
		currentHierarchy.currentLevel--
	} else {
		if _, ok := beginCommands[commandName]; ok {
			if commandName == VK_CMD_BEGIN_RENDER_PASS {
				currentHierarchy.levelId[currentHierarchy.currentLevel] = currentHierarchy.currentId
				currentHierarchy.currentId++
				currentHierarchy.currentLevel++
			}
		}
	}
	return label
}

func getSubCommandLabel(cmdNode dependencygraph2.CmdNode) string {
	label := ""
	for i := 1; i < len(cmdNode.Index); i++ {
		label += fmt.Sprintf("/%d", cmdNode.Index[i])
	}
	return label
}

func getSplittedLabelByChar(label *string, splitChar byte) []string {
	splitLabel := []string{}
	lastPosition := 0
	for i := 0; i <= len(*label); i++ {
		if i == len(*label) || (*label)[i] == splitChar {
			splitLabel = append(splitLabel, (*label)[lastPosition:i])
			lastPosition = i + 1
		}
	}
	return splitLabel
}
func getMaxCommonPrefixBetweenSplitLabels(splitLabel1 *[]string, splitLabel2 *[]string) int {
	size := len(*splitLabel1)
	if len(*splitLabel2) < size {
		size = len(*splitLabel2)
	}
	for i := 0; i < size; i++ {
		if (*splitLabel1)[i] != (*splitLabel2)[i] {
			return i
		}
	}
	return size
}

func getMaxCommonPrefixBetweenLabels(label1, label2 string) int {
	splitLabel1 := getSplittedLabelByChar(&label1, '/')
	splitLabel2 := getSplittedLabelByChar(&label2, '/')
	return getMaxCommonPrefixBetweenSplitLabels(&splitLabel1, &splitLabel2)
}

func createGraphFromDependencyGraph(dependencyGraph dependencygraph2.DependencyGraph) (*Graph, error) {

	numberOfNodes := dependencyGraph.NumNodes()
	graph := createGraph(0)
	currentHierarchy := &Hierarchy{}
	previousNode := &Node{}
	for i := 0; i < numberOfNodes; i++ {
		dependencyNode := dependencyGraph.GetNode(dependencygraph2.NodeID(i))
		if cmdNode, ok := dependencyNode.(dependencygraph2.CmdNode); ok {
			cmdNodeId := cmdNode.Index[0]
			command := dependencyGraph.GetCommand(api.CmdID(cmdNodeId))
			commandName := command.CmdName()
			label := getCommandLabel(currentHierarchy, command)
			label += fmt.Sprintf("%s%d", commandName, cmdNodeId)
			label += getSubCommandLabel(cmdNode)
			label += "\""
			attributes := fmt.Sprintf("\"%v\"", command.CmdParams())

			graph.addNodeByIdAndLabelAndNameAndAttributes(i, label, commandName, attributes)

			node := graph.nodeIdToNode[i]
			if _, ok1 := commandsInsideRenderScope[previousNode.name]; ok1 {
				if _, ok2 := commandsInsideRenderScope[node.name]; ok2 {
					if getMaxCommonPrefixBetweenLabels(previousNode.label, node.label) >= 2 {
						graph.addEdgeBetweenNodes(node, previousNode)
					}
				}
			}
			if _, ok := commandsInsideRenderScope[node.name]; ok {
				previousNode = node
			}
		}
	}

	addDependencyToGraph := func(source, sink dependencygraph2.NodeID) error {
		idSource, idSink := int(source), int(sink)
		if sourceNode, ok1 := graph.nodeIdToNode[idSource]; ok1 {
			if sinkNode, ok2 := graph.nodeIdToNode[idSink]; ok2 {
				_, ok1 = commandsInsideRenderScope[sourceNode.name]
				_, ok2 = commandsInsideRenderScope[sinkNode.name]
				if ok1 == false || ok2 == false {
					graph.addEdgeBetweenNodesById(idSource, idSink)
				}
			}
		}
		return nil
	}

	err := dependencyGraph.ForeachDependency(addDependencyToGraph)
	return graph, err
}

func GetGraphVisualizationFromCapture(ctx context.Context, p *capture.Capture) ([]byte, error) {
	config := dependencygraph2.DependencyGraphConfig{}
	dependencyGraph, err := dependencygraph2.BuildDependencyGraph(ctx, config, p, []api.Cmd{}, interval.U64RangeList{})
	if err != nil {
		return []byte{}, err
	}

	graph, err := createGraphFromDependencyGraph(dependencyGraph)
	graph.removeNodesWithZeroDegree()
	output := graph.getGraphInPbtxtFormat()
	return []byte(output), err
}
