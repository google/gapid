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
	"sort"
	"strconv"
)

const (
	NO_VISITED       = 0
	VISITED_AND_USED = -1
)

type node struct {
	inNeighbourIdToEdgeId  map[int]int
	outNeighbourIdToEdgeId map[int]int
	id                     int
	commandTypeId          int
	label                  string
	name                   string
	attributes             string
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

func getNewNode(id int, label string) *node {
	newNode := &node{
		inNeighbourIdToEdgeId:  map[int]int{},
		outNeighbourIdToEdgeId: map[int]int{},
		id:                     id,
		label:                  label,
	}
	return newNode
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

func (g *graph) traverseGraph(currentNode *node, visitTime, minVisitTime, idInStronglyConnectedComponents, visitedNodesId *[]int, currentId, currentTime *int) {
	*visitedNodesId = append(*visitedNodesId, currentNode.id)
	(*visitTime)[currentNode.id] = *currentTime
	(*minVisitTime)[currentNode.id] = *currentTime
	(*currentTime)++

	for neighbourId := range currentNode.outNeighbourIdToEdgeId {
		neighbour := g.nodeIdToNode[neighbourId]
		if (*visitTime)[neighbour.id] == NO_VISITED {
			g.traverseGraph(neighbour, visitTime, minVisitTime, idInStronglyConnectedComponents, visitedNodesId, currentId, currentTime)
		}
		if (*visitTime)[neighbour.id] != VISITED_AND_USED {
			if (*minVisitTime)[neighbour.id] < (*minVisitTime)[currentNode.id] {
				(*minVisitTime)[currentNode.id] = (*minVisitTime)[neighbour.id]
			}
		}
	}

	if (*minVisitTime)[currentNode.id] == (*visitTime)[currentNode.id] {
		for {
			lastNodeId := (*visitedNodesId)[len(*visitedNodesId)-1]
			(*visitTime)[lastNodeId] = VISITED_AND_USED
			*visitedNodesId = (*visitedNodesId)[:len(*visitedNodesId)-1]
			(*idInStronglyConnectedComponents)[lastNodeId] = *currentId
			if lastNodeId == currentNode.id {
				break
			}
		}
		(*currentId)++
	}
}

func (g *graph) getIdInStronglyConnectedComponents() []int {
	currentId := 0
	currentTime := 1
	visitTime := make([]int, g.maxNodeId+1)
	minVisitTime := make([]int, g.maxNodeId+1)
	idInStronglyConnectedComponents := make([]int, g.maxNodeId+1)
	visitedNodesId := make([]int, 0)

	for _, currentNode := range g.nodeIdToNode {
		if visitTime[currentNode.id] == NO_VISITED {
			g.traverseGraph(currentNode, &visitTime, &minVisitTime, &idInStronglyConnectedComponents, &visitedNodesId, &currentId, &currentTime)
		}
	}
	return idInStronglyConnectedComponents
}

func (g *graph) makeStronglyConnectedComponentsByCommandTypeId() {
	newGraph := createGraph(0)
	for _, currentNode := range g.nodeIdToNode {
		newNode := getNewNode(currentNode.commandTypeId, "")
		newGraph.addNode(newNode)
	}

	for _, currentNode := range g.nodeIdToNode {
		for neighbourId := range currentNode.outNeighbourIdToEdgeId {
			neighbour := g.nodeIdToNode[neighbourId]
			newGraph.addEdgeBetweenNodesById(currentNode.commandTypeId, neighbour.commandTypeId)
		}
	}
	idInStronglyConnectedComponents := newGraph.getIdInStronglyConnectedComponents()
	for _, currentNode := range g.nodeIdToNode {
		id := idInStronglyConnectedComponents[currentNode.commandTypeId]
		currentNode.label = currentNode.label + "/" + fmt.Sprintf("SCC%d", id)
	}
}

func (g *graph) getEdgesInDotFormat() string {
	output := ""
	for _, currentEdge := range g.edgeIdToEdge {
		lines := strconv.Itoa(currentEdge.source.id) + " -> " + strconv.Itoa(currentEdge.sink.id) + ";\n"
		output += lines
	}
	return output
}

func (g *graph) getNodesInDotFormat() string {
	output := ""
	for _, currentNode := range g.nodeIdToNode {
		lines := strconv.Itoa(currentNode.id) + "[label=" + currentNode.label + "]" + ";\n"
		output += lines
	}
	return output
}

func (g *graph) getGraphInDotFormat() string {
	output := "digraph g {\n"
	output += g.getNodesInDotFormat()
	output += g.getEdgesInDotFormat()
	output += "}\n"
	return output
}

func (g *graph) getGraphInPbtxtFormat() string {
	nodes := []*node{}
	for _, currentNode := range g.nodeIdToNode {
		nodes = append(nodes, currentNode)
	}
	sort.Sort(nodeSorter(nodes))

	output := ""
	for _, currentNode := range nodes {
		lines := "node {\n"
		lines += "name: \"" + currentNode.label + "\"\n"
		lines += "op: \"" + currentNode.label + "\"\n"

		neighbours := []*node{}
		for neighbourId := range currentNode.inNeighbourIdToEdgeId {
			neighbours = append(neighbours, g.nodeIdToNode[neighbourId])
		}
		sort.Sort(nodeSorter(neighbours))

		for _, neighbour := range neighbours {
			lines += "input: \"" + neighbour.label + "\"\n"
		}
		lines += "attr {\n"
		lines += "key: " + "\"" + currentNode.attributes + "\"\n"
		lines += "}\n"

		lines += "}\n"
		output += lines
	}
	return output
}
