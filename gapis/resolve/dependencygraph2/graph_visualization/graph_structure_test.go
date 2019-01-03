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

func getSortedKeysForNodes(input map[int]*node) []int {
	sortedKeys := []int{}
	for key := range input {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Ints(sortedKeys)
	return sortedKeys
}

func addNodeByIdAndLabel(g *graph, id int, label string) {
	newNode := getNewNode(id, label)
	g.addNode(newNode)
}

func addNodeByIdAndLabelAndNameAndAttributes(g *graph, id int, label, name, attributes string) {
	newNode := getNewNode(id, label)
	newNode.name = name
	newNode.attributes = attributes
	g.addNode(newNode)
}

func areEqualGraphs(t *testing.T, wantedGraph *graph, obtainedGraph *graph) bool {

	if wantedGraph.getNumberOfNodes() != obtainedGraph.getNumberOfNodes() {
		t.Errorf("The numbers of nodes are different %v != %v\n", wantedGraph.getNumberOfNodes(), obtainedGraph.getNumberOfNodes())
		return false
	}
	if wantedGraph.getNumberOfEdges() != obtainedGraph.getNumberOfEdges() {
		t.Errorf("The numbers of edges are different %v != %v\n", wantedGraph.getNumberOfEdges(), obtainedGraph.getNumberOfEdges())
		return false
	}

	wantedSortedIdNodes := getSortedKeysForNodes(wantedGraph.nodeIdToNode)
	obtainedSortedIdNodes := getSortedKeysForNodes(obtainedGraph.nodeIdToNode)
	if !reflect.DeepEqual(wantedSortedIdNodes, obtainedSortedIdNodes) {
		t.Errorf("The nodes ID are different in the graphs\n")
		return false
	}

	for _, id := range wantedSortedIdNodes {
		wantedNode := wantedGraph.nodeIdToNode[id]
		obtainedNode := obtainedGraph.nodeIdToNode[id]
		if wantedNode.label != obtainedNode.label {
			t.Errorf("The labels from nodes with ID %d are different %v != %v\n", id, wantedNode.label, obtainedNode.label)
			return false
		}
		if wantedNode.name != obtainedNode.name {
			t.Errorf("The names from nodes with ID %d are different %v != %v\n", id, wantedNode.name, obtainedNode.name)
			return false
		}
		wantedInNeighbourIdSorted := getSortedKeys(wantedNode.inNeighbourIdToEdgeId)
		obtainedInNeighbourIdSorted := getSortedKeys(obtainedNode.inNeighbourIdToEdgeId)
		if !reflect.DeepEqual(wantedInNeighbourIdSorted, obtainedInNeighbourIdSorted) {
			t.Errorf("The in-Neighbours ID are different for nodes with ID %d\n", id)
			return false
		}
		wantedOutNeighbourIdSorted := getSortedKeys(wantedNode.outNeighbourIdToEdgeId)
		obtainedOutNeighbourIdSorted := getSortedKeys(obtainedNode.outNeighbourIdToEdgeId)
		if !reflect.DeepEqual(wantedOutNeighbourIdSorted, obtainedOutNeighbourIdSorted) {
			t.Errorf("The out-Neighbours ID are different for nodes with ID %d\n", id)
			return false
		}
	}
	return true
}

func TestGraph1(t *testing.T) {

	wantedGraph := createGraph(0)
	addNodeByIdAndLabel(wantedGraph, 0, "A")
	addNodeByIdAndLabel(wantedGraph, 1, "B")
	addNodeByIdAndLabel(wantedGraph, 2, "C")
	addNodeByIdAndLabel(wantedGraph, 3, "D")
	addNodeByIdAndLabel(wantedGraph, 4, "E")
	addNodeByIdAndLabel(wantedGraph, 5, "F")
	addNodeByIdAndLabel(wantedGraph, 6, "G")
	addNodeByIdAndLabel(wantedGraph, 7, "H")
	addNodeByIdAndLabel(wantedGraph, 8, "I")
	addNodeByIdAndLabel(wantedGraph, 9, "J")

	obtainedGraph := createGraph(0)
	addNodeByIdAndLabel(obtainedGraph, 0, "A")
	addNodeByIdAndLabel(obtainedGraph, 1, "B")
	addNodeByIdAndLabel(obtainedGraph, 2, "C")
	addNodeByIdAndLabel(obtainedGraph, 3, "D")
	addNodeByIdAndLabel(obtainedGraph, 4, "E")
	addNodeByIdAndLabel(obtainedGraph, 5, "F")
	addNodeByIdAndLabel(obtainedGraph, 6, "G")
	addNodeByIdAndLabel(obtainedGraph, 7, "H")
	addNodeByIdAndLabel(obtainedGraph, 8, "I")
	addNodeByIdAndLabel(obtainedGraph, 9, "J")

	addNodeByIdAndLabel(obtainedGraph, 10, "K")
	addNodeByIdAndLabel(obtainedGraph, 11, "L")
	addNodeByIdAndLabel(obtainedGraph, 12, "M")
	obtainedGraph.removeNodeById(10)
	obtainedGraph.removeNodeById(11)
	obtainedGraph.removeNodeById(12)

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	addNodeByIdAndLabel(obtainedGraph, 10, "K")
	addNodeByIdAndLabel(obtainedGraph, 11, "L")
	addNodeByIdAndLabel(obtainedGraph, 12, "M")
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

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

}

func TestGraph2(t *testing.T) {

	wantedGraph := createGraph(0)
	obtainedGraph := createGraph(0)
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 0, "A", "vkCommandBuffer0", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 1, "A", "vkCommandBuffer1", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 2, "A", "vkCommandBuffer2", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 3, "A", "vkCommandBuffer3", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 4, "A", "vkCommandBuffer4", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 5, "A", "vkCommandBuffer5", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 6, "A", "vkCommandBuffer6", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 7, "A", "vkCommandBuffer7", "")
	obtainedGraph.removeNodesWithZeroDegree()

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 0, "A", "vkCommandBuffer0", "")
	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 2, "B", "vkCommandBuffer2", "")
	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 6, "C", "vkCommandBuffer6", "")
	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 3, "D", "vkCommandBuffer3", "")
	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 4, "E", "vkCommandBuffer4", "")
	addNodeByIdAndLabelAndNameAndAttributes(wantedGraph, 5, "F", "vkCommandBuffer5", "")
	wantedGraph.addEdgeBetweenNodesById(0, 3)
	wantedGraph.addEdgeBetweenNodesById(0, 4)
	wantedGraph.addEdgeBetweenNodesById(0, 5)
	wantedGraph.addEdgeBetweenNodesById(2, 3)
	wantedGraph.addEdgeBetweenNodesById(2, 4)
	wantedGraph.addEdgeBetweenNodesById(2, 5)
	wantedGraph.addEdgeBetweenNodesById(6, 3)
	wantedGraph.addEdgeBetweenNodesById(6, 4)
	wantedGraph.addEdgeBetweenNodesById(6, 5)

	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 0, "A", "vkCommandBuffer0", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 2, "B", "vkCommandBuffer2", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 6, "C", "vkCommandBuffer6", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 3, "D", "vkCommandBuffer3", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 4, "E", "vkCommandBuffer4", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 5, "F", "vkCommandBuffer5", "")
	addNodeByIdAndLabelAndNameAndAttributes(obtainedGraph, 1, "G", "vkCommandBuffer1", "")
	obtainedGraph.addEdgeBetweenNodesById(0, 1)
	obtainedGraph.addEdgeBetweenNodesById(2, 1)
	obtainedGraph.addEdgeBetweenNodesById(6, 1)
	obtainedGraph.addEdgeBetweenNodesById(1, 3)
	obtainedGraph.addEdgeBetweenNodesById(1, 4)
	obtainedGraph.addEdgeBetweenNodesById(1, 5)
	obtainedGraph.removeNodePreservingEdges(1)

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}
}

func TestGraph3(t *testing.T) {

	wantedGraph := createGraph(123456)
	obtainedGraph := createGraph(123456)
	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	wantedGraph.removeNodesWithZeroDegree()
	obtainedGraph.removeNodesWithZeroDegree()
	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	addNodeByIdAndLabel(wantedGraph, 123456, "")
	addNodeByIdAndLabel(wantedGraph, 10, "")
	wantedGraph.addEdgeBetweenNodesById(123456, 10)

	addNodeByIdAndLabel(obtainedGraph, 123456, "")
	addNodeByIdAndLabel(obtainedGraph, 10, "")
	obtainedGraph.addEdgeBetweenNodesById(10, 123456)
	obtainedGraph.removeEdgeById(1)
	obtainedGraph.addEdgeBetweenNodesById(123456, 10)

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}
}

func TestGetIdInStronglyConnectedComponents(t *testing.T) {

	currentGraph := createGraph(0)
	numberOfNodes := 12
	for id := 0; id < numberOfNodes; id++ {
		addNodeByIdAndLabel(currentGraph, id, "")
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
