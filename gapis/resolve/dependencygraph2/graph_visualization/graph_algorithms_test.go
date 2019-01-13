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
	"testing"
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
