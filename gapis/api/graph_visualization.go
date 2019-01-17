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

package api

import (
	"bytes"
	"fmt"
)

// Label describes the levels of hierarchy for nodes in the graph
// visualization using TensorBoard which reads pbtxt format.
type Label struct {
	// LevelsName is the name for each level that node belongs
	// from top level to down level.
	LevelsName []string

	// LevelsID is the ID for each level that node belongs
	// from top level to down level.
	LevelsID []int
}

// GetSize returns the number of levels.
func (label *Label) GetSize() int {
	return len(label.LevelsName)
}

// PushBack adds a level in the back of the current Label.
func (label *Label) PushBack(name string, id int) {
	label.LevelsName = append(label.LevelsName, name)
	label.LevelsID = append(label.LevelsID, id)
}

// PushFront adds a level in the front of the current Label..
func (label *Label) PushFront(name string, id int) {
	newLabel := &Label{LevelsName: []string{name}, LevelsID: []int{id}}
	newLabel.PushBackLabel(label)
	label.LevelsName = newLabel.LevelsName
	label.LevelsID = newLabel.LevelsID
}

// PushBackLabel adds a Label in the back of the current Label.
func (label *Label) PushBackLabel(labelToPush *Label) {
	label.LevelsName = append(label.LevelsName, labelToPush.LevelsName...)
	label.LevelsID = append(label.LevelsID, labelToPush.LevelsID...)
}

// Insert a new level in the current Label.
func (label *Label) Insert(level int, name string, id int) {
	if level < len(label.LevelsName) {
		label.LevelsName = append(label.LevelsName, "")
		label.LevelsID = append(label.LevelsID, 0)
		copy(label.LevelsName[level+1:], label.LevelsName[level:])
		copy(label.LevelsID[level+1:], label.LevelsID[level:])
		label.LevelsName[level] = name
		label.LevelsID[level] = id
	}
}

// GetCommandName returns the name of the last level
// corresponding to the node name.
func (label *Label) GetCommandName() string {
	if len(label.LevelsName) > 0 {
		return label.LevelsName[len(label.LevelsName)-1]
	}
	return ""
}

// GetCommandId returns the ID of the last level
// corresponding to the node ID.
func (label *Label) GetCommandId() int {
	if len(label.LevelsID) > 0 {
		return label.LevelsID[len(label.LevelsID)-1]
	}
	return 0
}

// GetTopLevelName returns the name of the first level
// corresponding to the top level in hierarchy.
func (label *Label) GetTopLevelName() string {
	if len(label.LevelsName) > 0 {
		return label.LevelsName[0]
	}
	return ""
}

// GetTopLevelID returns the ID of the first level
// corresponding to the top level in hierarchy.
func (label *Label) GetTopLevelID() int {
	if len(label.LevelsID) > 0 {
		return label.LevelsID[0]
	}
	return 0
}

// GetLabelAsAString returns the Label as a string concatenating
// names and ID for each level delimited by '/'.
func (label *Label) GetLabelAsAString() string {
	var output bytes.Buffer
	for i := range label.LevelsID {
		output.WriteString(label.LevelsName[i])
		fmt.Fprintf(&output, "%d", label.LevelsID[i])
		if i+1 < len(label.LevelsID) {
			output.WriteString("/")
		}
	}
	return output.String()
}

// Hierarchy describes the levels ID of hierarchy for vulkan
// commands and vulkan subcommands.
type Hierarchy struct {
	LevelsID []int
}

// GetSize returns the number of levels in Hierarchy.
func (h *Hierarchy) GetSize() int {
	return len(h.LevelsID)
}

// GetID returns ID for a specific level, indexed from 1.
func (h *Hierarchy) GetID(level int) int {
	return h.LevelsID[level-1]
}

// PopBack removes the last level in Hierarchy.
func (h *Hierarchy) PopBack() {
	if len(h.LevelsID) > 0 {
		h.LevelsID = h.LevelsID[:len(h.LevelsID)-1]
	}
}

// PushBackToResize keeps adding a new level in the back
// until to get newSize levels in Hierarchy.
func (h *Hierarchy) PushBackToResize(newSize int) {
	for len(h.LevelsID) < newSize {
		h.LevelsID = append(h.LevelsID, 0)
	}
}

// PopBackToResize keeps removing the back level until
// to get newSize levels in Hierarchy.
func (h *Hierarchy) PopBackToResize(newSize int) {
	for len(h.LevelsID) > newSize {
		h.PopBack()
	}
}

// IncreaseIDByOne increases in one a level ID, indexed from 1.
func (h *Hierarchy) IncreaseIDByOne(level int) {
	h.LevelsID[level-1]++
}

// HierarchyNames describes the levels name of Hierarchy for
// vulkan commands and vulkan subcommands.
type HierarchyNames struct {
	// BeginNameToLevel are the vulkan commands name that begin a new level.
	BeginNameToLevel map[string]int

	// EndNameToLevel are the vulkan commands name that end a new level.
	EndNameToLevel map[string]int

	// NameOfLevels are the names assigned to new levels.
	NameOfLevels []string
}

// GetName returns name for a specific level, indexed from 1.
func (hierarchyNames *HierarchyNames) GetName(level int) string {
	return hierarchyNames.NameOfLevels[level-1]
}

// PushBack adds in the back a new level with beginName,
// endName and the name for this level.
func (hierarchyNames *HierarchyNames) PushBack(beginName, endName, name string) {
	size := len(hierarchyNames.NameOfLevels) + 1
	hierarchyNames.BeginNameToLevel[beginName] = size
	hierarchyNames.EndNameToLevel[endName] = size
	hierarchyNames.NameOfLevels = append(hierarchyNames.NameOfLevels, name)
}

// GraphVisualizationAPI is the common interface for graph visualization.
type GraphVisualizationAPI interface {
	// GetGraphVisualizationBuilder returns a interface to GraphVisualizationBuilder
	GetGraphVisualizationBuilder() GraphVisualizationBuilder
}

// GraphVisualizationBuilder is the common interface used to process commands from
// graphics API in order to get the Label for nodes in the graph visualization.
type GraphVisualizationBuilder interface {
	// GetCommandLabel returns the Label for the command
	GetCommandLabel(command Cmd, cmdId uint64) *Label

	// GetSubCommandLabel returns the Label for the subcommand
	GetSubCommandLabel(index SubCmdIdx, commandName string, cmdId uint64, subCommandName string) *Label
}
