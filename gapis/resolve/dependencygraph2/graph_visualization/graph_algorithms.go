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
)

const (
	NO_VISITED       = 0
	VISITED_AND_USED = -1
	FRAME            = "FRAME"
	UNUSED           = "UNUSED"
)

// It is used to find the Strongly Connected Components (SCC) in a directed graph based on Tarjan algorithm
type tarjan struct {
	visitTime      []int
	minVisitTime   []int
	idInSCC        []int
	visitedNodesId []int
	currentId      int
	currentTime    int
}

func (g *graph) traverseGraphToFindSCC(currentNode *node, tarjanParameters *tarjan) {
	tarjanParameters.visitedNodesId = append(tarjanParameters.visitedNodesId, currentNode.id)
	tarjanParameters.visitTime[currentNode.id] = tarjanParameters.currentTime
	tarjanParameters.minVisitTime[currentNode.id] = tarjanParameters.currentTime
	tarjanParameters.currentTime++

	for neighbourId := range currentNode.outNeighbourIdToEdgeId {
		neighbour := g.nodeIdToNode[neighbourId]
		if tarjanParameters.visitTime[neighbour.id] == NO_VISITED {
			g.traverseGraphToFindSCC(neighbour, tarjanParameters)
		}
		if tarjanParameters.visitTime[neighbour.id] != VISITED_AND_USED {
			if tarjanParameters.minVisitTime[neighbour.id] < tarjanParameters.minVisitTime[currentNode.id] {
				tarjanParameters.minVisitTime[currentNode.id] = tarjanParameters.minVisitTime[neighbour.id]
			}
		}
	}

	if tarjanParameters.minVisitTime[currentNode.id] == tarjanParameters.visitTime[currentNode.id] {
		for {
			lastNodeId := tarjanParameters.visitedNodesId[len(tarjanParameters.visitedNodesId)-1]
			tarjanParameters.visitTime[lastNodeId] = VISITED_AND_USED
			tarjanParameters.visitedNodesId = tarjanParameters.visitedNodesId[:len(tarjanParameters.visitedNodesId)-1]
			tarjanParameters.idInSCC[lastNodeId] = tarjanParameters.currentId
			if lastNodeId == currentNode.id {
				break
			}
		}
		tarjanParameters.currentId++
	}
}

func (g *graph) getIdInStronglyConnectedComponents() []int {

	tarjanParameters := tarjan{
		visitTime:    make([]int, g.maxNodeId+1),
		minVisitTime: make([]int, g.maxNodeId+1),
		idInSCC:      make([]int, g.maxNodeId+1),
		currentId:    1,
		currentTime:  1,
	}

	for _, currentNode := range g.nodeIdToNode {
		if tarjanParameters.visitTime[currentNode.id] == NO_VISITED {
			g.traverseGraphToFindSCC(currentNode, &tarjanParameters)
		}
	}
	return tarjanParameters.idInSCC
}

func (g *graph) makeStronglyConnectedComponentsByCommandTypeId() {
	newGraph := createGraph(0)
	for _, currentNode := range g.nodeIdToNode {
		newNode := getNewNode(currentNode.commandTypeId, &api.Label{})
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
		currentNode.label.PushBack("SCC", id)
	}
}

func (g *graph) bfs(sourceNode *node, visited []bool, visitedNodes *[]*node) {
	head := len(*visitedNodes)
	visited[sourceNode.id] = true
	*visitedNodes = append(*visitedNodes, sourceNode)
	for head < len(*visitedNodes) {
		currentNode := (*visitedNodes)[head]
		head++
		neighbours := g.getSortedNeighbours(currentNode.outNeighbourIdToEdgeId)
		for _, neighbour := range neighbours {
			if !visited[neighbour.id] {
				visited[neighbour.id] = true
				*visitedNodes = append(*visitedNodes, neighbour)
			}
		}

		for _, subCommandNode := range currentNode.subCommandNodes {
			if !visited[subCommandNode.id] {
				visited[subCommandNode.id] = true
				*visitedNodes = append(*visitedNodes, subCommandNode)
			}
		}
	}
}

func (g *graph) joinNodesByFrame() {
	visited := make([]bool, g.maxNodeId+1)
	frameNumber := 1
	nodes := g.getSortedNodes()
	for _, currentNode := range nodes {
		if !visited[currentNode.id] && currentNode.isEndOfFrame {
			visitedNodes := []*node{}
			g.bfs(currentNode, visited, &visitedNodes)
			for _, visitedNode := range visitedNodes {
				visitedNode.label.PushFront(FRAME, frameNumber)
			}
			frameNumber++
		}
	}
}

func (g *graph) joinNodesWithZeroDegree() {
	for _, currentNode := range g.nodeIdToNode {
		if (len(currentNode.inNeighbourIdToEdgeId) + len(currentNode.outNeighbourIdToEdgeId)) == 0 {
			currentNode.label.PushFront(UNUSED, 0)
		}
	}
}
