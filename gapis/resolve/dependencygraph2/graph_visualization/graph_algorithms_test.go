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
	"github.com/google/gapid/gapis/api"
	"reflect"
	"testing"
)

const (
	VK_BEGIN_COMMAND_BUFFER   = "vkBeginCommandBuffer"
	VK_CMD_BEGIN_RENDER_PASS  = "vkCmdBeginRenderPass"
	VK_CMD_NEXT_SUBPASS       = "vkCmdNextSubpass"
	VK_COMMAND_BUFFER         = "vkCommandBuffer"
	VK_RENDER_PASS            = "vkRenderPass"
	VK_SUBPASS                = "vkSubpass"
	VK_END_COMMAND_BUFFER     = "vkEndCommandBuffer"
	VK_CMD_END_RENDER_PASS    = "vkCmdEndRenderPass"
	COMMAND_BUFFER            = "commandBuffer"
	VK_CMD_DEBUG_MARKER_BEGIN = "vkCmdDebugMarkerBeginEXT"
	VK_CMD_DEBUG_MARKER_END   = "vkCmdDebugMarkerEndEXT"
	VK_CMD_DEBUG_MARKER       = "vkCmdDebugMarker"
)

func TestGetIdInStronglyConnectedComponents(t *testing.T) {

	currentGraph := createGraph(0)
	numberOfNodes := 12
	for id := 0; id < numberOfNodes; id++ {
		addNodeByIdAndName(currentGraph, id, "")
	}
	currentGraph.addEdgeBetweenNodesById(0, 1)
	currentGraph.addEdgeBetweenNodesById(1, 2)
	currentGraph.addEdgeBetweenNodesById(2, 3)
	currentGraph.addEdgeBetweenNodesById(3, 0)
	currentGraph.addEdgeBetweenNodesById(3, 4)
	currentGraph.addEdgeBetweenNodesById(3, 5)
	currentGraph.addEdgeBetweenNodesById(3, 6)
	currentGraph.addEdgeBetweenNodesById(4, 9)
	currentGraph.addEdgeBetweenNodesById(4, 11)
	currentGraph.addEdgeBetweenNodesById(11, 10)
	currentGraph.addEdgeBetweenNodesById(11, 8)
	currentGraph.addEdgeBetweenNodesById(6, 10)
	currentGraph.addEdgeBetweenNodesById(6, 7)
	currentGraph.addEdgeBetweenNodesById(7, 6)
	currentGraph.addEdgeBetweenNodesById(10, 8)
	currentGraph.addEdgeBetweenNodesById(8, 10)
	currentGraph.addEdgeBetweenNodesById(7, 8)
	currentGraph.addEdgeBetweenNodesById(8, 7)

	idInStronglyConnectedComponents := currentGraph.getIdInStronglyConnectedComponents()
	wantedStronglyConnectedComponentes := [][]int{
		[]int{0, 1, 2, 3},
		[]int{6, 7, 8, 10},
		[]int{4},
		[]int{5},
		[]int{9},
		[]int{11},
	}
	for _, currentScc := range wantedStronglyConnectedComponentes {
		idInSccForNodes := map[int]bool{}
		for _, idNode := range currentScc {
			idInScc := idInStronglyConnectedComponents[idNode]
			idInSccForNodes[idInScc] = true
		}
		if len(idInSccForNodes) != 1 {
			t.Errorf("There are nodes belonging to different SCC %v", currentScc)
		}
	}

}

func TestBfs(t *testing.T) {

	currentGraph := createGraph(0)
	numberOfNodes := 14
	for id := 0; id < numberOfNodes; id++ {
		addNodeByIdAndName(currentGraph, id, "")
	}

	currentGraph.addEdgeBetweenNodesById(0, 1)
	currentGraph.addEdgeBetweenNodesById(0, 5)
	currentGraph.addEdgeBetweenNodesById(1, 3)
	currentGraph.addEdgeBetweenNodesById(1, 2)
	currentGraph.addEdgeBetweenNodesById(5, 6)
	currentGraph.addEdgeBetweenNodesById(3, 6)
	currentGraph.addEdgeBetweenNodesById(2, 4)
	currentGraph.addEdgeBetweenNodesById(4, 6)
	currentGraph.addEdgeBetweenNodesById(7, 8)
	currentGraph.addEdgeBetweenNodesById(7, 11)
	currentGraph.addEdgeBetweenNodesById(11, 8)
	currentGraph.addEdgeBetweenNodesById(9, 12)
	currentGraph.addEdgeBetweenNodesById(12, 13)

	wantedNodeIdToNumberOfComponent := map[int]int{
		0:  1,
		1:  1,
		2:  1,
		3:  1,
		4:  1,
		5:  1,
		6:  1,
		7:  2,
		8:  2,
		9:  3,
		10: 4,
		11: 2,
		12: 3,
		13: 3,
	}

	visited := make([]bool, currentGraph.maxNodeId+1)
	nodes := currentGraph.getSortedNodes()
	for _, currentNode := range nodes {
		if !visited[currentNode.id] {
			visitedNodes := []*node{}
			currentGraph.bfs(currentNode, visited, &visitedNodes)
			numberComponent := map[int]bool{}
			for _, visitedNode := range visitedNodes {
				numberComponent[wantedNodeIdToNumberOfComponent[visitedNode.id]] = true
			}
			if len(numberComponent) != 1 {
				t.Errorf("There are some nodes in different components")
			}
		}
	}

}

func TestMakeChunks(t *testing.T) {
	numberOfNodes := 30
	currentGraph := createGraph(numberOfNodes)
	auxiliarCommands := []string{
		"SetBarrier",
		"DrawIndex",
		"SetScissor",
		"SetIndexForDraw",
	}
	labels := []*api.Label{
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{0}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_BEGIN_COMMAND_BUFFER}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_BEGIN_RENDER_PASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[1]}, LevelsID: []int{1, 1, 1, 3}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 1, 4}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 5}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 6}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 7}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 8}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 9}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 10}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 11}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 12}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 13}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 1, 2, 14}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{15}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{16}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{17}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{18}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{19}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{20}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{21}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 1, 22}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 2, 23}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 3, 24}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 4, 25}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 5, 26}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, auxiliarCommands[0]}, LevelsID: []int{1, 2, 6, 27}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{28}},
		&api.Label{LevelsName: []string{auxiliarCommands[0]}, LevelsID: []int{29}},
	}
	nodes := []*node{}
	for i := 0; i < numberOfNodes; i++ {
		currentNode := currentGraph.nodeIdToNode[i+1]
		currentNode.label = labels[i]
		nodes = append(nodes, currentNode)
	}

	config := chunkConfig{
		maximumNumberOfNodesByLevel:   5,
		minimumNumberOfNodesToBeChunk: 2,
	}
	currentGraph.makeChunks(config)

	wantedLabels := []*api.Label{
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 1, 0}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, VK_BEGIN_COMMAND_BUFFER}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, VK_CMD_BEGIN_RENDER_PASS}, LevelsID: []int{1, 0, 1, 2}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, auxiliarCommands[1]},
			LevelsID: []int{1, 0, 1, 0, 1, 3}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, auxiliarCommands[0]},
			LevelsID: []int{1, 0, 1, 0, 1, 4}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 1, 5}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 1, 6}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 2, 7}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 2, 8}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 3, 9}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 3, 10}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 4, 11}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 4, 12}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 5, 13}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, VK_SUBPASS, SUPER + auxiliarCommands[0],
			SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{1, 0, 1, 0, 2, 0, 5, 14}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 1, 15}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 2, 16}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 2, 17}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 3, 18}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 3, 19}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 4, 20}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 4, 21}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 1, 1, 22}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 1, 2, 23}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 2, 3, 24}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 2, 4, 25}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 3, 5, 26}},
		&api.Label{LevelsName: []string{COMMAND_BUFFER, SUPER + VK_RENDER_PASS, VK_RENDER_PASS, SUPER + VK_SUBPASS, SUPER + VK_SUBPASS, VK_SUBPASS,
			auxiliarCommands[0]}, LevelsID: []int{1, 0, 2, 0, 3, 6, 27}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 5, 28}},
		&api.Label{LevelsName: []string{SUPER + auxiliarCommands[0], SUPER + auxiliarCommands[0], auxiliarCommands[0]}, LevelsID: []int{0, 5, 29}},
	}
	for i := range wantedLabels {
		if !reflect.DeepEqual(wantedLabels[i], nodes[i].label) {
			t.Errorf("The label for the node with id %d is different\n", i)
			t.Errorf("Wanted %v\n", wantedLabels[i])
			t.Errorf("Obtained %v\n", nodes[i].label)
		}
	}

}
