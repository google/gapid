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
)

func (g *graph) writeEdgesInDotFormat(output *bytes.Buffer) {
	nodes := g.getSortedNodes()
	for _, currentNode := range nodes {
		inNeighbours := g.getSortedNeighbours(currentNode.inNeighbourIdToEdgeId)
		for _, neighbour := range inNeighbours {
			fmt.Fprintf(output, "%d -> %d;\n", neighbour.id, currentNode.id)
		}
	}
}

func (g *graph) writeNodesInDotFormat(output *bytes.Buffer) {
	nodes := g.getSortedNodes()
	for _, currentNode := range nodes {
		fmt.Fprintf(output, "%d[label=%s];\n", currentNode.id, currentNode.label.GetCommandName())
	}
}

func (g *graph) getGraphInDotFormat() []byte {
	var output bytes.Buffer
	output.WriteString("digraph g {\n")
	g.writeNodesInDotFormat(&output)
	g.writeEdgesInDotFormat(&output)
	output.WriteString("}\n")
	return output.Bytes()
}

func (g *graph) getGraphInPbtxtFormat() []byte {
	nodes := g.getSortedNodes()
	var output bytes.Buffer
	for _, currentNode := range nodes {
		output.WriteString("node {\n")
		output.WriteString("name: \"" + currentNode.label.GetLabelAsAString() + "\"\n")
		fmt.Fprintf(&output, "op: \"%s%d\"\n", currentNode.label.GetCommandName(), currentNode.label.GetCommandId())

		neighbours := g.getSortedNeighbours(currentNode.inNeighbourIdToEdgeId)
		for _, neighbour := range neighbours {
			output.WriteString("input: \"" + neighbour.label.GetLabelAsAString() + "\"\n")
		}
		if currentNode.color != "" {
			output.WriteString("device: \"" + currentNode.color + "\"\n")
		}
		output.WriteString("attr {\n")
		output.WriteString("key: \"" + currentNode.attributes + "\"\n")
		output.WriteString("}\n")
		output.WriteString("}\n")
	}
	return output.Bytes()
}
