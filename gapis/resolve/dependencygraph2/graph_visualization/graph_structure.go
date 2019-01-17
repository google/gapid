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
	"fmt"
	"github.com/google/gapid/gapis/api"
	"sort"
)

type node struct {
	inNeighbourIdToEdgeId  map[int]int
	outNeighbourIdToEdgeId map[int]int
	id                     int
	commandTypeId          int
	label                  *api.Label
	attributes             string
	isEndOfFrame           bool
	subCommandNodes        []*node
	color                  string
}

type edge struct {
	source *node
	sink   *node
	id     int
	label  string
}

type graph struct {
	nodeIdToNode map[int]*node
	edgeIdToEdge map[int]*edge
	maxNodeId    int
	maxEdgeId    int
}

func (g *graph) getNumberOfNodes() int {
	return len(g.nodeIdToNode)
}

func (g *graph) getNumberOfEdges() int {
	return len(g.edgeIdToEdge)
}

func createGraph(numberOfNodes int) *graph {
	newGraph := &graph{nodeIdToNode: map[int]*node{}, edgeIdToEdge: map[int]*edge{}}
	for i := 0; i < numberOfNodes; i++ {
		newNode := &node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{}, id: newGraph.maxNodeId + 1}
		newGraph.nodeIdToNode[newNode.id] = newNode
		newGraph.maxNodeId++
	}
	return newGraph
}

func (g *graph) addNode(newNode *node) error {
	if _, ok := g.nodeIdToNode[newNode.id]; ok {
		return fmt.Errorf("Trying to add an existing node with id %d", newNode.id)
	}

	g.nodeIdToNode[newNode.id] = newNode
	if newNode.id > g.maxNodeId {
		g.maxNodeId = newNode.id
	}
	return nil
}

func getNewNode(id int, label *api.Label) *node {
	newNode := &node{
		inNeighbourIdToEdgeId:  map[int]int{},
		outNeighbourIdToEdgeId: map[int]int{},
		id:                     id,
		label:                  label,
	}
	return newNode
}

func (currentNode *node) addSubCommandNode(subCommandNode *node) {
	currentNode.subCommandNodes = append(currentNode.subCommandNodes, subCommandNode)
}

func (g *graph) addEdge(newEdge *edge) {
	source, sink := newEdge.source, newEdge.sink
	if _, ok := source.outNeighbourIdToEdgeId[sink.id]; ok {
		return
	}

	g.edgeIdToEdge[newEdge.id] = newEdge
	source.outNeighbourIdToEdgeId[sink.id] = newEdge.id
	sink.inNeighbourIdToEdgeId[source.id] = newEdge.id
	if newEdge.id > g.maxEdgeId {
		g.maxEdgeId = newEdge.id
	}
}

func (g *graph) addEdgeBetweenNodes(source, sink *node) {
	id := g.maxEdgeId + 1
	newEdge := &edge{source: source, sink: sink, id: id}
	g.addEdge(newEdge)
}

func (g *graph) addEdgeBetweenNodesById(idSource, idSink int) error {
	source, ok := g.nodeIdToNode[idSource]
	if !ok {
		return fmt.Errorf("Adding edge from non-existent node with id %d\n", idSource)
	}
	sink, ok := g.nodeIdToNode[idSink]
	if !ok {
		return fmt.Errorf("Adding edge to non-existent node with id %d\n", idSink)
	}
	id := g.maxEdgeId + 1
	newEdge := &edge{source: source, sink: sink, id: id}
	g.addEdge(newEdge)
	return nil
}

func (g *graph) removeEdgeById(id int) {
	currentEdge, ok := g.edgeIdToEdge[id]
	if !ok {
		return
	}

	source, sink := currentEdge.source, currentEdge.sink
	delete(source.outNeighbourIdToEdgeId, sink.id)
	delete(sink.inNeighbourIdToEdgeId, source.id)
	delete(g.edgeIdToEdge, id)
}

func (g *graph) removeNodeById(id int) {
	currentNode, ok := g.nodeIdToNode[id]
	if !ok {
		return
	}

	in, out := currentNode.inNeighbourIdToEdgeId, currentNode.outNeighbourIdToEdgeId
	for _, edgeId := range in {
		g.removeEdgeById(edgeId)
	}
	for _, edgeId := range out {
		g.removeEdgeById(edgeId)
	}
	delete(g.nodeIdToNode, id)
}

func (g *graph) removeNodesWithZeroDegree() {
	for id, currentNode := range g.nodeIdToNode {
		if (len(currentNode.inNeighbourIdToEdgeId) + len(currentNode.outNeighbourIdToEdgeId)) == 0 {
			g.removeNodeById(id)
		}
	}
}

func (g *graph) addEdgesBetweenInNeighboursAndOutNeighbours(idNode int) {
	currentNode, ok := g.nodeIdToNode[idNode]
	if !ok {
		return
	}
	for idSource := range currentNode.inNeighbourIdToEdgeId {
		for idSink := range currentNode.outNeighbourIdToEdgeId {
			g.addEdgeBetweenNodesById(idSource, idSink)
		}
	}
}

func (g *graph) removeNodePreservingEdges(idNode int) {
	g.addEdgesBetweenInNeighboursAndOutNeighbours(idNode)
	g.removeNodeById(idNode)
}

type nodeSorter []*node

func (input nodeSorter) Len() int {
	return len(input)
}
func (input nodeSorter) Swap(i, j int) {
	input[i], input[j] = input[j], input[i]
}
func (input nodeSorter) Less(i, j int) bool {
	return input[i].id < input[j].id
}

func (g *graph) getSortedNodes() []*node {
	nodes := []*node{}
	for _, currentNode := range g.nodeIdToNode {
		nodes = append(nodes, currentNode)
	}
	sort.Sort(nodeSorter(nodes))
	return nodes
}

func (g *graph) getSortedNeighbours(neighbourIdToEdgeId map[int]int) []*node {
	neighbours := []*node{}
	for neighbourId := range neighbourIdToEdgeId {
		neighbours = append(neighbours, g.nodeIdToNode[neighbourId])
	}
	sort.Sort(nodeSorter(neighbours))
	return neighbours
}
