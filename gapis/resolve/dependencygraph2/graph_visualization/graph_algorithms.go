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
	"bytes"
	"fmt"
	"github.com/google/gapid/gapis/api"
)

const (
	NO_VISITED       = 0
	VISITED_AND_USED = -1
	FRAME            = "FRAME"
	UNUSED           = "UNUSED"
	SUPER            = "SUPER"
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

func (g *graph) assignColorToNodes() {
	for _, currentNode := range g.nodeIdToNode {
		currentNode.color = currentNode.label.GetTopLevelName()
	}
}

func (g *graph) joinNodesThatDoNotBelongToAnyFrame() {
	for _, currentNode := range g.nodeIdToNode {
		if currentNode.label.GetSize() > 0 && currentNode.label.GetTopLevelName() != FRAME {
			currentNode.label.PushFront(UNUSED, 0)
		}
	}
}

type chunkConfig struct {
	maximumNumberOfNodesByLevel   int
	minimumNumberOfNodesToBeChunk int
}

type chunk struct {
	// levelIDToPosition maps a single levelID from Label to a position in
	// the range [0, 1, 2, 3, ...].
	levelIDToPosition map[int]int

	// positionToChunksID are the chunks ID obtained from the K-ary tree built
	// for the range [0, 1, 2, 3, ...].
	positionToChunksID [][]int

	// built checks if the K-ary tree was built for this chunk.
	built bool

	// config contains the minimum number of nodes to build the K-ary tree and also
	// the value of k, which is called maximum number of nodes by level.
	config chunkConfig
}

// assignChunksIDToNodesInTheKaryTreeBuilt builds a K-ary tree for the range of
// nodes ID [left, left+1, left+2, ..... , right-1, right] (chunk). Then it assigns
// the chunks ID for the nodes ID obtained from the K-ary tree.
func (c *chunk) assignChunksIDToNodesInTheKaryTreeBuilt(left, right int, currentChunksID *[]int) {

	// Base case: if the number of nodes ID in the current chunk is at most K,
	// then the currentChunksID is assigned to nodes ID.
	if (right - left + 1) <= c.config.maximumNumberOfNodesByLevel {
		for i := left; i <= right; i++ {
			c.positionToChunksID[i] = make([]int, len(*currentChunksID))
			copy(c.positionToChunksID[i], *currentChunksID)
		}
	} else {
		// General case: It creates at most K smaller chunks as balanced as possible
		// and build the K-ary tree for each of them.

		// It is append to give ID to the smaller chunks.
		*currentChunksID = append(*currentChunksID, 1)
		// It computes the ceiling to take all nodes ID of the current chunk
		chunkSize := (right - left + 1 + c.config.maximumNumberOfNodesByLevel - 1) / c.config.maximumNumberOfNodesByLevel
		newLeft := left
		newRight := newLeft + chunkSize - 1
		chunkID := 1
		for newLeft <= right {
			if newRight > right {
				newRight = right
			}
			// It assigns the ID for the new smaller chunk.
			(*currentChunksID)[len(*currentChunksID)-1] = chunkID
			// Call recursively to build the K-ary tree for the new smaller chunk.
			c.assignChunksIDToNodesInTheKaryTreeBuilt(newLeft, newRight, currentChunksID)
			newLeft = newRight + 1
			newRight = newLeft + chunkSize - 1
			chunkID++
		}
		// It removes the last element before go backwards in the recursion.
		*currentChunksID = (*currentChunksID)[:len(*currentChunksID)-1]
	}
}

func (g *graph) makeChunks(config chunkConfig) {

	// This first part builds an implicit tree of nodes defined like a pair
	// {key -> set_of_values}. The key of a node is a string and the set_of_values
	// is a set of integers being LevelsID from Labels. For each level i in
	// Label starting from top level, a new node is created with:
	// key = LevelsName[0] + levelsID[0] + / + LevelsName[1] + LevelsID[1] + /  + ... + LevelsName[i],
	// where '+' means concatenation and LevelsID[i] is inserted in set_of_values,
	// if the node with such key already exists only the LevelsID[i] is inserted.
	nodes := g.getSortedNodes()
	labelAsAStringToChunk := map[string]*chunk{}
	for _, currentNode := range nodes {
		var labelAsAString bytes.Buffer
		for i, name := range currentNode.label.LevelsName {
			id := currentNode.label.LevelsID[i]
			labelAsAString.WriteString(name)
			if name != FRAME {
				currentChunk, ok := labelAsAStringToChunk[labelAsAString.String()]
				if !ok {
					currentChunk = &chunk{levelIDToPosition: map[int]int{}, config: config}
					labelAsAStringToChunk[labelAsAString.String()] = currentChunk
				}
				if _, ok := currentChunk.levelIDToPosition[id]; !ok {
					currentChunk.levelIDToPosition[id] = len(currentChunk.levelIDToPosition)
				}
			}
			fmt.Fprintf(&labelAsAString, "%d/", id)
		}
	}

	// This second part builds a K-ary tree for each set_of_values in nodes of the
	// implicit tree built in the first part. In a K-ary tree every node consist
	// of at most K children (chunks).
	for _, currentNode := range nodes {
		var labelAsAString bytes.Buffer
		newLabel := &api.Label{}
		for i, name := range currentNode.label.LevelsName {
			id := currentNode.label.LevelsID[i]
			labelAsAString.WriteString(name)
			if name != FRAME {
				currentChunk := labelAsAStringToChunk[labelAsAString.String()]
				chunkSize := len(currentChunk.levelIDToPosition)
				if chunkSize >= currentChunk.config.minimumNumberOfNodesToBeChunk {
					if !currentChunk.built {
						currentChunk.built = true
						currentChunk.positionToChunksID = make([][]int, chunkSize)
						currentChunksID := make([]int, 1)
						currentChunk.assignChunksIDToNodesInTheKaryTreeBuilt(0, chunkSize-1, &currentChunksID)
					}
					position := currentChunk.levelIDToPosition[id]
					newLevelsID := currentChunk.positionToChunksID[position]
					for _, id := range newLevelsID {
						newLabel.PushBack(SUPER+name, id)
					}
				}
			}
			fmt.Fprintf(&labelAsAString, "%d/", id)
			newLabel.PushBack(name, id)
		}
		currentNode.label = newLabel
	}
}
