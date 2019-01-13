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

func addNodeByIdAndName(g *graph, id int, name string) {
	label := &api.Label{LevelsName: []string{name}}
	newNode := getNewNode(id, label)
	g.addNode(newNode)
}

func addNodeByIdAndNameAndAttributes(g *graph, id int, name, attributes string) {
	label := &api.Label{LevelsName: []string{name}}
	newNode := getNewNode(id, label)
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
		if !reflect.DeepEqual(wantedNode.label, obtainedNode.label) {
			t.Errorf("The labels from nodes with ID %d are different %v != %v\n", id, wantedNode.label, obtainedNode.label)
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
	addNodeByIdAndName(wantedGraph, 0, "A")
	addNodeByIdAndName(wantedGraph, 1, "B")
	addNodeByIdAndName(wantedGraph, 2, "C")
	addNodeByIdAndName(wantedGraph, 3, "D")
	addNodeByIdAndName(wantedGraph, 4, "E")
	addNodeByIdAndName(wantedGraph, 5, "F")
	addNodeByIdAndName(wantedGraph, 6, "G")
	addNodeByIdAndName(wantedGraph, 7, "H")
	addNodeByIdAndName(wantedGraph, 8, "I")
	addNodeByIdAndName(wantedGraph, 9, "J")

	obtainedGraph := createGraph(0)
	addNodeByIdAndName(obtainedGraph, 0, "A")
	addNodeByIdAndName(obtainedGraph, 1, "B")
	addNodeByIdAndName(obtainedGraph, 2, "C")
	addNodeByIdAndName(obtainedGraph, 3, "D")
	addNodeByIdAndName(obtainedGraph, 4, "E")
	addNodeByIdAndName(obtainedGraph, 5, "F")
	addNodeByIdAndName(obtainedGraph, 6, "G")
	addNodeByIdAndName(obtainedGraph, 7, "H")
	addNodeByIdAndName(obtainedGraph, 8, "I")
	addNodeByIdAndName(obtainedGraph, 9, "J")

	addNodeByIdAndName(obtainedGraph, 10, "K")
	addNodeByIdAndName(obtainedGraph, 11, "L")
	addNodeByIdAndName(obtainedGraph, 12, "M")
	obtainedGraph.removeNodeById(10)
	obtainedGraph.removeNodeById(11)
	obtainedGraph.removeNodeById(12)

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	addNodeByIdAndName(obtainedGraph, 10, "K")
	addNodeByIdAndName(obtainedGraph, 11, "L")
	addNodeByIdAndName(obtainedGraph, 12, "M")
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
	addNodeByIdAndNameAndAttributes(obtainedGraph, 0, "vkCommandBuffer0", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 1, "vkCommandBuffer1", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 2, "vkCommandBuffer2", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 3, "vkCommandBuffer3", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 4, "vkCommandBuffer4", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 5, "vkCommandBuffer5", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 6, "vkCommandBuffer6", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 7, "vkCommandBuffer7", "")
	obtainedGraph.removeNodesWithZeroDegree()

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}

	addNodeByIdAndNameAndAttributes(wantedGraph, 0, "vkCommandBuffer0", "")
	addNodeByIdAndNameAndAttributes(wantedGraph, 2, "vkCommandBuffer2", "")
	addNodeByIdAndNameAndAttributes(wantedGraph, 6, "vkCommandBuffer6", "")
	addNodeByIdAndNameAndAttributes(wantedGraph, 3, "vkCommandBuffer3", "")
	addNodeByIdAndNameAndAttributes(wantedGraph, 4, "vkCommandBuffer4", "")
	addNodeByIdAndNameAndAttributes(wantedGraph, 5, "vkCommandBuffer5", "")
	wantedGraph.addEdgeBetweenNodesById(0, 3)
	wantedGraph.addEdgeBetweenNodesById(0, 4)
	wantedGraph.addEdgeBetweenNodesById(0, 5)
	wantedGraph.addEdgeBetweenNodesById(2, 3)
	wantedGraph.addEdgeBetweenNodesById(2, 4)
	wantedGraph.addEdgeBetweenNodesById(2, 5)
	wantedGraph.addEdgeBetweenNodesById(6, 3)
	wantedGraph.addEdgeBetweenNodesById(6, 4)
	wantedGraph.addEdgeBetweenNodesById(6, 5)

	addNodeByIdAndNameAndAttributes(obtainedGraph, 0, "vkCommandBuffer0", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 2, "vkCommandBuffer2", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 6, "vkCommandBuffer6", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 3, "vkCommandBuffer3", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 4, "vkCommandBuffer4", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 5, "vkCommandBuffer5", "")
	addNodeByIdAndNameAndAttributes(obtainedGraph, 1, "vkCommandBuffer1", "")
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

	addNodeByIdAndName(wantedGraph, 123456, "")
	addNodeByIdAndName(wantedGraph, 10, "")
	wantedGraph.addEdgeBetweenNodesById(123456, 10)

	addNodeByIdAndName(obtainedGraph, 123456, "")
	addNodeByIdAndName(obtainedGraph, 10, "")
	obtainedGraph.addEdgeBetweenNodesById(10, 123456)
	obtainedGraph.removeEdgeById(1)
	obtainedGraph.addEdgeBetweenNodesById(123456, 10)

	if !areEqualGraphs(t, wantedGraph, obtainedGraph) {
		t.Errorf("The graphs are different\n")
	}
}
