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
	"reflect"
	"sort"
	"testing"
)

func getSortedKeys(input map[int]int) []int {
	sortedKeys := []int{}
	for key := range input {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Ints(sortedKeys)
	return sortedKeys
}

func getSortedKeysForNodes(input map[int]*Node) []int {
	sortedKeys := []int{}
	for key := range input {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Ints(sortedKeys)
	return sortedKeys
}

func areEqualGraphs(t *testing.T, wantedGraph *Graph, obtainedGraph *Graph) bool {

	if wantedGraph.numberOfNodes != obtainedGraph.numberOfNodes {
		t.Errorf("The numbers of nodes are different %v != %v\n", wantedGraph.numberOfNodes, obtainedGraph.numberOfNodes)
	}
	if wantedGraph.numberOfEdges != obtainedGraph.numberOfEdges {
		t.Errorf("The numbers of edges are different %v != %v\n", wantedGraph.numberOfEdges, obtainedGraph.numberOfEdges)
	}

	wantedSortedIdNodes := getSortedKeysForNodes(wantedGraph.nodeIdToNode)
	obtainedSortedIdNodes := getSortedKeysForNodes(obtainedGraph.nodeIdToNode)
	if reflect.DeepEqual(wantedSortedIdNodes, obtainedSortedIdNodes) == false {
		t.Errorf("The nodes ID are different in the graphs\n")
	}

	for _, id := range wantedSortedIdNodes {
		wantedNode := wantedGraph.nodeIdToNode[id]
		obtainedNode := obtainedGraph.nodeIdToNode[id]
		if reflect.DeepEqual(wantedNode.label, obtainedNode.label) == false {
			t.Errorf("The labels from nodes with ID %d are different %v != %v\n", id, wantedNode.label, obtainedNode.label)
		}
		if reflect.DeepEqual(wantedNode.name, obtainedNode.name) == false {
			t.Errorf("The name from nodes with ID %d are different %v != %v\n", id, wantedNode.name, obtainedNode.name)
		}
		wantedInNeighbourIdSorted := getSortedKeys(wantedNode.inNeighbourIdToEdgeId)
		obtainedInNeighbourIdSorted := getSortedKeys(obtainedNode.inNeighbourIdToEdgeId)
		if reflect.DeepEqual(wantedInNeighbourIdSorted, obtainedInNeighbourIdSorted) == false {
			t.Errorf("The in-Neighbours ID are different for nodes with ID %d\n", id)
		}
		wantedOutNeighbourIdSorted := getSortedKeys(wantedNode.outNeighbourIdToEdgeId)
		obtainedOutNeighbourIdSorted := getSortedKeys(obtainedNode.outNeighbourIdToEdgeId)
		if reflect.DeepEqual(wantedOutNeighbourIdSorted, obtainedOutNeighbourIdSorted) == false {
			t.Errorf("The out-Neighbours ID are different for nodes with ID %d\n", id)
		}
	}
	return true
}

func TestGraph1(t *testing.T) {

	wantedGraph := createGraph(0)
	wantedGraph.addNodeByIdAndLabel(0, "A")
	wantedGraph.addNodeByIdAndLabel(1, "B")
	wantedGraph.addNodeByIdAndLabel(2, "C")
	wantedGraph.addNodeByIdAndLabel(3, "D")
	wantedGraph.addNodeByIdAndLabel(4, "E")
	wantedGraph.addNodeByIdAndLabel(5, "F")
	wantedGraph.addNodeByIdAndLabel(6, "G")
	wantedGraph.addNodeByIdAndLabel(7, "H")
	wantedGraph.addNodeByIdAndLabel(8, "I")
	wantedGraph.addNodeByIdAndLabel(9, "J")

	obtainedGraph := createGraph(0)
	obtainedGraph.addNodeByIdAndLabel(0, "A")
	obtainedGraph.addNodeByIdAndLabel(1, "B")
	obtainedGraph.addNodeByIdAndLabel(2, "C")
	obtainedGraph.addNodeByIdAndLabel(3, "D")
	obtainedGraph.addNodeByIdAndLabel(4, "E")
	obtainedGraph.addNodeByIdAndLabel(5, "F")
	obtainedGraph.addNodeByIdAndLabel(6, "G")
	obtainedGraph.addNodeByIdAndLabel(7, "H")
	obtainedGraph.addNodeByIdAndLabel(8, "I")
	obtainedGraph.addNodeByIdAndLabel(9, "J")

	obtainedGraph.addNodeByIdAndLabel(10, "K")
	obtainedGraph.addNodeByIdAndLabel(11, "L")
	obtainedGraph.addNodeByIdAndLabel(12, "M")
	obtainedGraph.removeNodeById(10)
	obtainedGraph.removeNodeById(11)
	obtainedGraph.removeNodeById(12)

	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}

	obtainedGraph.addNodeByIdAndLabel(10, "K")
	obtainedGraph.addNodeByIdAndLabel(11, "L")
	obtainedGraph.addNodeByIdAndLabel(12, "M")
	obtainedGraph.addEdgeBetweenNodesById(10, 1)
	obtainedGraph.addEdgeBetweenNodesById(10, 11)
	obtainedGraph.addEdgeBetweenNodesById(10, 4)
	obtainedGraph.addEdgeBetweenNodesById(11, 12)
	obtainedGraph.addEdgeBetweenNodesById(2, 12)
	obtainedGraph.addEdgeBetweenNodesById(4, 11)
	obtainedGraph.addEdgeBetweenNodesById(10, 12)
	obtainedGraph.removeNodeById(10)
	obtainedGraph.removeNodeById(11)
	obtainedGraph.removeNodeById(12)

	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}

}

func TestGraph2(t *testing.T) {

	wantedGraph := createGraph(0)
	obtainedGraph := createGraph(0)
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(0, "A", "vkCommandBuffer0", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(1, "A", "vkCommandBuffer1", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(2, "A", "vkCommandBuffer2", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(3, "A", "vkCommandBuffer3", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(4, "A", "vkCommandBuffer4", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(5, "A", "vkCommandBuffer5", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(6, "A", "vkCommandBuffer6", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(7, "A", "vkCommandBuffer7", "")
	obtainedGraph.removeNodesWithZeroDegree()

	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}

	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(0, "A", "vkCommandBuffer0", "")
	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(2, "B", "vkCommandBuffer2", "")
	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(6, "C", "vkCommandBuffer6", "")
	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(3, "D", "vkCommandBuffer3", "")
	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(4, "E", "vkCommandBuffer4", "")
	wantedGraph.addNodeByIdAndLabelAndNameAndAttributes(5, "F", "vkCommandBuffer5", "")
	wantedGraph.addEdgeBetweenNodesById(0, 3)
	wantedGraph.addEdgeBetweenNodesById(0, 4)
	wantedGraph.addEdgeBetweenNodesById(0, 5)
	wantedGraph.addEdgeBetweenNodesById(2, 3)
	wantedGraph.addEdgeBetweenNodesById(2, 4)
	wantedGraph.addEdgeBetweenNodesById(2, 5)
	wantedGraph.addEdgeBetweenNodesById(6, 3)
	wantedGraph.addEdgeBetweenNodesById(6, 4)
	wantedGraph.addEdgeBetweenNodesById(6, 5)

	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(0, "A", "vkCommandBuffer0", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(2, "B", "vkCommandBuffer2", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(6, "C", "vkCommandBuffer6", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(3, "D", "vkCommandBuffer3", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(4, "E", "vkCommandBuffer4", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(5, "F", "vkCommandBuffer5", "")
	obtainedGraph.addNodeByIdAndLabelAndNameAndAttributes(1, "G", "vkCommandBuffer1", "")
	obtainedGraph.addEdgeBetweenNodesById(0, 1)
	obtainedGraph.addEdgeBetweenNodesById(2, 1)
	obtainedGraph.addEdgeBetweenNodesById(6, 1)
	obtainedGraph.addEdgeBetweenNodesById(1, 3)
	obtainedGraph.addEdgeBetweenNodesById(1, 4)
	obtainedGraph.addEdgeBetweenNodesById(1, 5)
	obtainedGraph.removeNodePreservingEdges(1)

	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}
}

func TestGraph3(t *testing.T) {

	wantedGraph := createGraph(123456)
	obtainedGraph := createGraph(123455)
	obtainedGraph.addNodeByDefault()
	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}

	wantedGraph.removeNodesWithZeroDegree()
	obtainedGraph.removeNodesWithZeroDegree()
	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}

	wantedGraph.addNodeByIdAndLabel(123456, "")
	wantedGraph.addNodeByIdAndLabel(10, "")
	wantedGraph.addEdgeBetweenNodesById(123456, 10)

	obtainedGraph.addNodeByIdAndLabel(123456, "")
	obtainedGraph.addNodeByIdAndLabel(10, "")
	obtainedGraph.addEdgeBetweenNodesById(10, 123456)
	obtainedGraph.removeEdgeById(1)
	obtainedGraph.addEdgeBetweenNodesById(123456, 10)

	if areEqualGraphs(t, wantedGraph, obtainedGraph) == false {
		t.Errorf("The graphs are different\n")
	}
}

func TestGetIdInStronglyConnectedComponents(t *testing.T) {

	graph := createGraph(0)
	numberOfNodes := 12
	for id := 0; id < numberOfNodes; id++ {
		graph.addNodeByIdAndLabel(id, "")
	}
	graph.addEdgeBetweenNodesById(0, 1)
	graph.addEdgeBetweenNodesById(1, 2)
	graph.addEdgeBetweenNodesById(2, 3)
	graph.addEdgeBetweenNodesById(3, 0)
	graph.addEdgeBetweenNodesById(3, 4)
	graph.addEdgeBetweenNodesById(3, 5)
	graph.addEdgeBetweenNodesById(3, 6)
	graph.addEdgeBetweenNodesById(4, 9)
	graph.addEdgeBetweenNodesById(4, 11)
	graph.addEdgeBetweenNodesById(11, 10)
	graph.addEdgeBetweenNodesById(11, 8)
	graph.addEdgeBetweenNodesById(6, 10)
	graph.addEdgeBetweenNodesById(6, 7)
	graph.addEdgeBetweenNodesById(7, 6)
	graph.addEdgeBetweenNodesById(10, 8)
	graph.addEdgeBetweenNodesById(8, 10)
	graph.addEdgeBetweenNodesById(7, 8)
	graph.addEdgeBetweenNodesById(8, 7)

	idInStronglyConnectedComponents := graph.getIdInStronglyConnectedComponents()
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
