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

package vulkan

import (
	"bytes"
	"fmt"
	"github.com/google/gapid/gapis/api"
)

// Interface compliance test
var (
	_ = api.GraphVisualizationBuilder(&labelForVulkanCommands{})
	_ = api.GraphVisualizationAPI(API{})
)

const (
	VK_BEGIN_COMMAND_BUFFER   = "vkBeginCommandBuffer"
	VK_CMD_BEGIN_RENDER_PASS  = "vkCmdBeginRenderPass"
	VK_CMD_NEXT_SUBPASS       = "vkCmdNextSubpass"
	VK_COMMAND_BUFFER         = "vkCommandBuffer"
	VK_RENDER_PASS            = "vkRenderPass"
	VK_SUBPASS                = "vkSubpass"
	VK_END_COMMAND_BUFFER     = "vkEndCommandBuffer"
	VK_CMD_END_RENDER_PASS    = "vkCmdEndRenderPass"
	COMMAND_BUFFER            = "commandBuffer"
	VK_CMD_DEBUG_MARKER_BEGIN = "vkCmdDebugMarkerBeginEXT"
	VK_CMD_DEBUG_MARKER_END   = "vkCmdDebugMarkerEndEXT"
	VK_CMD_DEBUG_MARKER       = "vkCmdDebugMarker"
)

var (
	commandHierarchyNames    = getCommandHierarchyNames()
	subCommandHierarchyNames = getSubCommandHierarchyNames()
)

type labelForVulkanCommands struct {
	labelAsAStringToHierarchy           map[string]*api.Hierarchy
	subCommandIndexNameToHierarchyLabel map[string]*api.Label
	commandBufferIdToHierarchy          map[VkCommandBuffer]*api.Hierarchy
	commandBufferIdToOrderNumber        map[VkCommandBuffer]int
	nameAndIdToDebugMarkerStack         map[string]*debugMarkerStack
}

type debugMarkerStack struct {
	// positionsOfDebugMarkersBegin is used like a stack, representing the position
	// of debug_marker_begin commands in labels to access fast to its Label.
	positionsOfDebugMarkersBegin []int

	// labels are all labels from commands considered.
	labels []*api.Label

	// debugMarkerID is the unique ID assigned to debug_markers created.
	debugMarkerID int
}

func (dm *debugMarkerStack) getSize() int {
	return len(dm.positionsOfDebugMarkersBegin)
}

// top returns the Label of the last debug_marker_begin command if exists.
func (dm *debugMarkerStack) top() *api.Label {
	if dm.getSize() > 0 {
		position := dm.positionsOfDebugMarkersBegin[dm.getSize()-1]
		if position < len(dm.labels) {
			return dm.labels[position]
		}
	}
	return &api.Label{}
}

func (dm *debugMarkerStack) pop() {
	if dm.getSize() > 0 {
		dm.positionsOfDebugMarkersBegin = dm.positionsOfDebugMarkersBegin[:dm.getSize()-1]
	}
}

func (dm *debugMarkerStack) push(label *api.Label) {
	dm.labels = append(dm.labels, label)
	if label.GetCommandName() == VK_CMD_DEBUG_MARKER_BEGIN {
		dm.positionsOfDebugMarkersBegin = append(dm.positionsOfDebugMarkersBegin, len(dm.labels)-1)
	}
}

// addDebugMarker adds a new debug marker from the last debug_marker_begin Label to the current Label.
func (dm *debugMarkerStack) addDebugMarker() {
	position := dm.positionsOfDebugMarkersBegin[dm.getSize()-1]
	size := dm.labels[position].GetSize()
	dm.debugMarkerID++
	for i := position; i < len(dm.labels); i++ {
		dm.labels[i].Insert(size-1, VK_CMD_DEBUG_MARKER, dm.debugMarkerID)
	}
}

func (dm *debugMarkerStack) checkDebugMarkers(label *api.Label) {
	// It preserves Labels with no-decreasing sizes in the stack of debug_markers_begin avoiding
	// insert debug markers in Labels with smaller sizes that would break the levels of hierarchy.
	for dm.getSize() > 0 && label.GetSize() < dm.top().GetSize() {
		dm.pop()
	}
	dm.push(label)

	if label.GetCommandName() == VK_CMD_DEBUG_MARKER_END {
		if dm.getSize() > 0 {
			if label.GetSize() == dm.top().GetSize() && getMaxCommonPrefix(label, dm.top()) == label.GetSize()-1 {
				dm.addDebugMarker()
			}
			dm.pop()
		}
	}
}

func getCommandHierarchyNames() *api.HierarchyNames {
	commandHierarchyNames := &api.HierarchyNames{BeginNameToLevel: map[string]int{}, EndNameToLevel: map[string]int{},
		NameOfLevels: []string{}}

	commandHierarchyNames.PushBack(VK_BEGIN_COMMAND_BUFFER, VK_END_COMMAND_BUFFER, VK_COMMAND_BUFFER)
	commandHierarchyNames.PushBack(VK_CMD_BEGIN_RENDER_PASS, VK_CMD_END_RENDER_PASS, VK_RENDER_PASS)
	commandHierarchyNames.PushBack(VK_CMD_NEXT_SUBPASS, VK_CMD_NEXT_SUBPASS, VK_SUBPASS)
	return commandHierarchyNames
}

func getSubCommandHierarchyNames() *api.HierarchyNames {
	subCommandHierarchyNames := &api.HierarchyNames{BeginNameToLevel: map[string]int{}, EndNameToLevel: map[string]int{},
		NameOfLevels: []string{}}

	subCommandHierarchyNames.PushBack(VK_CMD_BEGIN_RENDER_PASS, VK_CMD_END_RENDER_PASS, VK_RENDER_PASS)
	subCommandHierarchyNames.PushBack(VK_CMD_NEXT_SUBPASS, VK_CMD_NEXT_SUBPASS, VK_SUBPASS)
	return subCommandHierarchyNames
}

func getMaxCommonPrefix(label1 *api.Label, label2 *api.Label) int {
	size := len(label1.LevelsID)
	if len(label2.LevelsID) < size {
		size = len(label2.LevelsID)
	}
	for i := 0; i < size; i++ {
		if label1.LevelsName[i] != label2.LevelsName[i] || label1.LevelsID[i] != label2.LevelsID[i] {
			return i
		}
	}
	return size
}

func getCommandBufferId(command api.Cmd) (VkCommandBuffer, bool) {
	parameters := command.CmdParams()
	for _, parameter := range parameters {
		if parameter.Name == COMMAND_BUFFER {
			return parameter.Get().(VkCommandBuffer), true
		}
	}
	return 0, false
}

// GetCommandLabel returns the Label for the Vulkan command.
func (builder *labelForVulkanCommands) GetCommandLabel(command api.Cmd, cmdId uint64) *api.Label {
	label := &api.Label{}
	commandName := command.CmdName()
	nameAndId := ""
	if commandBufferId, ok := getCommandBufferId(command); ok {
		hierarchy, ok := builder.commandBufferIdToHierarchy[commandBufferId]
		if !ok {
			hierarchy = &api.Hierarchy{}
			builder.commandBufferIdToHierarchy[commandBufferId] = hierarchy
			builder.commandBufferIdToOrderNumber[commandBufferId] = len(builder.commandBufferIdToOrderNumber) + 1
		}
		label.PushBack(COMMAND_BUFFER, builder.commandBufferIdToOrderNumber[commandBufferId])
		label.PushBackLabel(getLabelFromHierarchy(commandName, commandHierarchyNames, hierarchy))
		label.PushBack(commandName, int(cmdId))
		nameAndId = fmt.Sprintf("%s%d", COMMAND_BUFFER, builder.commandBufferIdToOrderNumber[commandBufferId])
	} else {
		label.PushBack(commandName, int(cmdId))
		nameAndId = fmt.Sprintf("%s%d", commandName, cmdId)
	}

	currentDebugMarker, ok := builder.nameAndIdToDebugMarkerStack[nameAndId]
	if !ok {
		currentDebugMarker = &debugMarkerStack{}
		builder.nameAndIdToDebugMarkerStack[nameAndId] = currentDebugMarker
	}
	currentDebugMarker.checkDebugMarkers(label)
	return label
}

// GetSubCommandLabel returns the Label for the Vulkan subcommand.
func (builder *labelForVulkanCommands) GetSubCommandLabel(index api.SubCmdIdx, commandName string, cmdId uint64, subCommandName string) *api.Label {
	label := &api.Label{}
	label.PushBack(commandName, int(cmdId))
	var subCommandIndexName bytes.Buffer
	fmt.Fprintf(&subCommandIndexName, "%s_%d", commandName, cmdId)
	for i := 1; i < len(index); i++ {
		fmt.Fprintf(&subCommandIndexName, "/%d", index[i])
		if i+1 < len(index) {
			if hierarchyLabel, ok := builder.subCommandIndexNameToHierarchyLabel[subCommandIndexName.String()]; ok {
				label.PushBackLabel(hierarchyLabel)
			} else {
				label.PushBack("", int(index[i]))
			}
		}
	}
	temporaryLabelAsAString := label.GetLabelAsAString()
	hierarchy, ok := builder.labelAsAStringToHierarchy[temporaryLabelAsAString]
	if !ok {
		hierarchy = &api.Hierarchy{}
		builder.labelAsAStringToHierarchy[temporaryLabelAsAString] = hierarchy
	}
	labelFromHierarchy := getLabelFromHierarchy(subCommandName, subCommandHierarchyNames, hierarchy)
	labelFromHierarchy.PushBack(subCommandName, int(index[len(index)-1]))
	builder.subCommandIndexNameToHierarchyLabel[subCommandIndexName.String()] = labelFromHierarchy
	label.PushBackLabel(labelFromHierarchy)

	nameAndId := fmt.Sprintf("%s%d", commandName, cmdId)
	currentDebugMarker, ok := builder.nameAndIdToDebugMarkerStack[nameAndId]
	if !ok {
		currentDebugMarker = &debugMarkerStack{}
		builder.nameAndIdToDebugMarkerStack[nameAndId] = currentDebugMarker
	}
	currentDebugMarker.checkDebugMarkers(label)
	return label
}

func getLabelFromHierarchy(name string, hierarchyNames *api.HierarchyNames, hierarchy *api.Hierarchy) *api.Label {
	isEndCommand := false
	if level, ok := hierarchyNames.BeginNameToLevel[name]; ok {
		hierarchy.PushBackToResize(level + 1)
		hierarchy.IncreaseIDByOne(level)
	} else if level, ok := hierarchyNames.EndNameToLevel[name]; ok {
		hierarchy.PopBackToResize(level + 1)
		isEndCommand = true
	}

	label := &api.Label{}
	for level := 1; level < hierarchy.GetSize(); level++ {
		label.PushBack(hierarchyNames.GetName(level), hierarchy.GetID(level))
	}

	if level, ok := hierarchyNames.BeginNameToLevel[name]; ok && name == VK_CMD_BEGIN_RENDER_PASS {
		levelForSubpass := level + 1
		hierarchy.PushBackToResize(levelForSubpass + 1)
		hierarchy.IncreaseIDByOne(levelForSubpass)
	}
	if isEndCommand {
		hierarchy.PopBack()
	}
	return label
}

// GetGraphVisualizationBuilder returns a builder to process commands from
// Vulkan graphics API in order to get the Label for commands.
func (API) GetGraphVisualizationBuilder() api.GraphVisualizationBuilder {
	return &labelForVulkanCommands{
		labelAsAStringToHierarchy:           map[string]*api.Hierarchy{},
		subCommandIndexNameToHierarchyLabel: map[string]*api.Label{},
		commandBufferIdToHierarchy:          map[VkCommandBuffer]*api.Hierarchy{},
		commandBufferIdToOrderNumber:        map[VkCommandBuffer]int{},
		nameAndIdToDebugMarkerStack:         map[string]*debugMarkerStack{},
	}
}
