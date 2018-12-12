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
	"strconv"
)

const (
	NO_VISITED       = 0
	VISITED_AND_USED = -1
)

type Node struct {
	inNeighbourIdToEdgeId  map[int]int
	outNeighbourIdToEdgeId map[int]int
	id                     int
	commandTypeId          int
	label                  string
	name                   string
	attributes             string
}

type Edge struct {
	source *Node
	sink   *Node
	id     int
	label  string
}

type Graph struct {
	nodeIdToNode  map[int]*Node
	edgeIdToEdge  map[int]*Edge
	maxNodeId     int
	maxEdgeId     int
	numberOfNodes int
	numberOfEdges int
}

func createGraph(numberOfNodes int) *Graph {
	newGraph := &Graph{nodeIdToNode: map[int]*Node{}, edgeIdToEdge: map[int]*Edge{}}
	for i := 0; i < numberOfNodes; i++ {
		newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{}, id: newGraph.maxNodeId + 1}
		newGraph.nodeIdToNode[newNode.id] = newNode
		newGraph.numberOfNodes++
		newGraph.maxNodeId++
	}
	return newGraph
}

func (g *Graph) addNodeByDefault() int {
	id := g.maxNodeId + 1
	newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{}, id: id}
	g.nodeIdToNode[id] = newNode
	g.numberOfNodes++
	g.maxNodeId++
	return id
}

func (g *Graph) addNodeByIdAndLabel(id int, label string) bool {
	if _, ok := g.nodeIdToNode[id]; ok {
		return false
	}

	newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{}, id: id, label: label}
	g.nodeIdToNode[id] = newNode
	g.numberOfNodes++
	if id > g.maxNodeId {
		g.maxNodeId = id
	}
	return true
}

func (g *Graph) addNodeByIdAndLabelAndCommandTypeId(id int, label string, commandTypeId int) bool {
	if _, ok := g.nodeIdToNode[id]; ok {
		return false
	}

	newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{},
		id: id, label: label, commandTypeId: commandTypeId}
	g.nodeIdToNode[id] = newNode
	g.numberOfNodes++
	if id > g.maxNodeId {
		g.maxNodeId = id
	}
	return true
}

func (g *Graph) addNodeByIdAndLabelAndCommandTypeIdAndAttributes(id int, label string, commandTypeId int, attributes string) bool {
	if _, ok := g.nodeIdToNode[id]; ok {
		return false
	}

	newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{},
		id: id, label: label, commandTypeId: commandTypeId, attributes: attributes}
	g.nodeIdToNode[id] = newNode
	g.numberOfNodes++
	if id > g.maxNodeId {
		g.maxNodeId = id
	}
	return true
}

func (g *Graph) addNodeByIdAndLabelAndNameAndAttributes(id int, label string, name string, attributes string) bool {
	if _, ok := g.nodeIdToNode[id]; ok {
		return false
	}

	newNode := &Node{inNeighbourIdToEdgeId: map[int]int{}, outNeighbourIdToEdgeId: map[int]int{},
		id: id, label: label, name: name, attributes: attributes}
	g.nodeIdToNode[id] = newNode
	g.numberOfNodes++
	if id > g.maxNodeId {
		g.maxNodeId = id
	}
	return true
}

func (g *Graph) addEdge(newEdge *Edge) bool {
	source, sink := newEdge.source, newEdge.sink
	if _, ok := source.outNeighbourIdToEdgeId[sink.id]; ok {
		return false
	}

	g.edgeIdToEdge[newEdge.id] = newEdge
	g.numberOfEdges++
	source.outNeighbourIdToEdgeId[sink.id] = newEdge.id
	sink.inNeighbourIdToEdgeId[source.id] = newEdge.id
	if newEdge.id > g.maxEdgeId {
		g.maxEdgeId = newEdge.id
	}
	return true
}

func (g *Graph) addEdgeBetweenNodes(source, sink *Node) {
	id := g.maxEdgeId + 1
	newEdge := &Edge{source: source, sink: sink, id: id}
	g.addEdge(newEdge)
}

func (g *Graph) addEdgeBetweenNodesById(idSource, idSink int) (int, bool) {
	source, ok := g.nodeIdToNode[idSource]
	if ok == false {
		return 0, false
	}
	sink, ok := g.nodeIdToNode[idSink]
	if ok == false {
		return 0, false
	}
	id := g.maxEdgeId + 1
	newEdge := &Edge{source: source, sink: sink, id: id}
	g.addEdge(newEdge)
	return id, true
}

func (g *Graph) removeEdgeById(id int) bool {
	edge, ok := g.edgeIdToEdge[id]
	if ok == false {
		return false
	}

	source, sink := edge.source, edge.sink
	delete(source.outNeighbourIdToEdgeId, sink.id)
	delete(sink.inNeighbourIdToEdgeId, source.id)

	delete(g.edgeIdToEdge, id)
	g.numberOfEdges--
	return true
}

func (g *Graph) removeNodeById(id int) bool {
	node, ok := g.nodeIdToNode[id]
	if ok == false {
		return false
	}

	in, out := node.inNeighbourIdToEdgeId, node.outNeighbourIdToEdgeId
	for _, edgeId := range in {
		g.removeEdgeById(edgeId)
	}
	for _, edgeId := range out {
		g.removeEdgeById(edgeId)
	}
	delete(g.nodeIdToNode, id)
	g.numberOfNodes--
	return true
}

func (g *Graph) removeNodesWithZeroDegree() {
	for id, node := range g.nodeIdToNode {
		if (len(node.inNeighbourIdToEdgeId) + len(node.outNeighbourIdToEdgeId)) == 0 {
			g.removeNodeById(id)
		}
	}
}

func (g *Graph) addEdgesBetweenInNeighboursAndOutNeighbours(idNode int) bool {
	node, ok := g.nodeIdToNode[idNode]
	if ok == false {
		return false
	}
	for idSource := range node.inNeighbourIdToEdgeId {
		for idSink := range node.outNeighbourIdToEdgeId {
			g.addEdgeBetweenNodesById(idSource, idSink)
		}
	}
	return true
}

func (g *Graph) removeNodePreservingEdges(idNode int) bool {
	if g.addEdgesBetweenInNeighboursAndOutNeighbours(idNode) == false {
		return false
	}
	if g.removeNodeById(idNode) == false {
		return false
	}
	return true
}

func (g *Graph) traverseGraph(node *Node, visitTime, minVisitTime, idInStronglyConnectedComponents, visitedNodesId *[]int, currentId, currentTime *int) {
	*visitedNodesId = append(*visitedNodesId, node.id)
	(*visitTime)[node.id] = *currentTime
	(*minVisitTime)[node.id] = *currentTime
	(*currentTime)++

	for idNeighbour := range node.outNeighbourIdToEdgeId {
		neighbour := g.nodeIdToNode[idNeighbour]
		if (*visitTime)[neighbour.id] == NO_VISITED {
			g.traverseGraph(neighbour, visitTime, minVisitTime, idInStronglyConnectedComponents, visitedNodesId, currentId, currentTime)
		}
		if (*visitTime)[neighbour.id] != VISITED_AND_USED {
			if (*minVisitTime)[neighbour.id] < (*minVisitTime)[node.id] {
				(*minVisitTime)[node.id] = (*minVisitTime)[neighbour.id]
			}
		}
	}

	if (*minVisitTime)[node.id] == (*visitTime)[node.id] {
		for {
			lastNodeId := (*visitedNodesId)[len(*visitedNodesId)-1]
			(*visitTime)[lastNodeId] = VISITED_AND_USED
			*visitedNodesId = (*visitedNodesId)[:len(*visitedNodesId)-1]
			(*idInStronglyConnectedComponents)[lastNodeId] = *currentId
			if lastNodeId == node.id {
				break
			}
		}
		(*currentId)++
	}
}

func (g *Graph) getIdInStronglyConnectedComponents() []int {
	currentId := 0
	currentTime := 1
	visitTime := make([]int, g.maxNodeId+1)
	minVisitTime := make([]int, g.maxNodeId+1)
	idInStronglyConnectedComponents := make([]int, g.maxNodeId+1)
	visitedNodesId := make([]int, 0)

	for _, node := range g.nodeIdToNode {
		if visitTime[node.id] == NO_VISITED {
			g.traverseGraph(node, &visitTime, &minVisitTime, &idInStronglyConnectedComponents, &visitedNodesId, &currentId, &currentTime)
		}
	}
	return idInStronglyConnectedComponents
}

func (g *Graph) makeStronglyConnectedComponentsByCommandTypeId() {
	newGraph := createGraph(0)
	for _, node := range g.nodeIdToNode {
		newGraph.addNodeByIdAndLabel(node.commandTypeId, "")
	}

	for _, node := range g.nodeIdToNode {
		for idNeighbour := range node.outNeighbourIdToEdgeId {
			neighbour := g.nodeIdToNode[idNeighbour]
			newGraph.addEdgeBetweenNodesById(node.commandTypeId, neighbour.commandTypeId)
		}
	}
	idInStronglyConnectedComponents := newGraph.getIdInStronglyConnectedComponents()
	for _, node := range g.nodeIdToNode {
		id := idInStronglyConnectedComponents[node.commandTypeId]
		node.label = node.label + "/" + fmt.Sprintf("SCC%d", id)
	}
}

func (g *Graph) getEdgesInDotFormat() string {
	output := ""
	for _, edge := range g.edgeIdToEdge {
		lines := strconv.Itoa(edge.source.id) + " -> " + strconv.Itoa(edge.sink.id) + ";\n"
		output += lines
	}
	return output
}

func (g *Graph) getNodesInDotFormat() string {
	output := ""
	for _, node := range g.nodeIdToNode {
		lines := strconv.Itoa(node.id) + "[label=" + node.label + "]" + ";\n"
		output += lines
	}
	return output
}

func (g *Graph) getGraphInDotFormat() string {
	output := "digraph g {\n"
	output += g.getNodesInDotFormat()
	output += g.getEdgesInDotFormat()
	output += "}\n"
	return output
}

func (g *Graph) getGraphInPbtxtFormat() string {
	output := ""
	for _, node := range g.nodeIdToNode {
		lines := "node {\n"
		lines += "name: " + node.label + "\n"
		lines += "op: " + node.label + "\n"
		for idNeighbour := range node.inNeighbourIdToEdgeId {
			neighbour := g.nodeIdToNode[idNeighbour]
			lines += "input: " + neighbour.label + "\n"
		}
		lines += "attr {\n"
		lines += "key: " + "\"" + node.name + "\"\n"
		lines += "}\n"

		lines += "}\n"
		output += lines
	}
	return output
}
