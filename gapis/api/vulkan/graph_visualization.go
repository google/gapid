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
	labelsInsideDebugMarkers            []*api.Label
	positionOfDebugMarkersBegin         []int
	numberOfDebugMarker                 int
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

func addDebugMarker(builder *labelForVulkanCommands, from, to int) {
	level := builder.labelsInsideDebugMarkers[len(builder.labelsInsideDebugMarkers)-1].GetSize() - 1
	builder.numberOfDebugMarker++
	for i := from; i <= to; i++ {
		builder.labelsInsideDebugMarkers[i].Insert(level, VK_CMD_DEBUG_MARKER, builder.numberOfDebugMarker)
	}
}

func checkDebugMarkers(builder *labelForVulkanCommands, currentLabel *api.Label) {
	commandName := currentLabel.GetCommandName()
	positionOfLastDebugMarkerBegin := 0
	labelOfLastDebugMarkerBegin := &api.Label{}
	if len(builder.positionOfDebugMarkersBegin) > 0 {
		positionOfLastDebugMarkerBegin = builder.positionOfDebugMarkersBegin[len(builder.positionOfDebugMarkersBegin)-1]
		labelOfLastDebugMarkerBegin = builder.labelsInsideDebugMarkers[positionOfLastDebugMarkerBegin]
	}

	if commandName == VK_CMD_DEBUG_MARKER_BEGIN {
		if len(builder.positionOfDebugMarkersBegin) > 0 {
			if currentLabel.GetSize() < labelOfLastDebugMarkerBegin.GetSize() {
				builder.positionOfDebugMarkersBegin = builder.positionOfDebugMarkersBegin[:len(builder.positionOfDebugMarkersBegin)-1]
			}
		}
		builder.labelsInsideDebugMarkers = append(builder.labelsInsideDebugMarkers, currentLabel)
		builder.positionOfDebugMarkersBegin = append(builder.positionOfDebugMarkersBegin, len(builder.labelsInsideDebugMarkers)-1)

	} else if commandName == VK_CMD_DEBUG_MARKER_END {

		if len(builder.positionOfDebugMarkersBegin) > 0 {
			if currentLabel.GetSize() != labelOfLastDebugMarkerBegin.GetSize() {
				builder.positionOfDebugMarkersBegin = builder.positionOfDebugMarkersBegin[:len(builder.positionOfDebugMarkersBegin)-1]

			} else if getMaxCommonPrefix(labelOfLastDebugMarkerBegin, currentLabel) == currentLabel.GetSize()-1 {
				builder.labelsInsideDebugMarkers = append(builder.labelsInsideDebugMarkers, currentLabel)
				builder.positionOfDebugMarkersBegin = append(builder.positionOfDebugMarkersBegin, len(builder.labelsInsideDebugMarkers)-1)
				addDebugMarker(builder, positionOfLastDebugMarkerBegin, len(builder.labelsInsideDebugMarkers)-1)
				builder.positionOfDebugMarkersBegin = builder.positionOfDebugMarkersBegin[:len(builder.positionOfDebugMarkersBegin)-1]
			}
		}

	} else if len(builder.positionOfDebugMarkersBegin) > 0 {
		if currentLabel.GetSize() < labelOfLastDebugMarkerBegin.GetSize() {
			builder.positionOfDebugMarkersBegin = builder.positionOfDebugMarkersBegin[:len(builder.positionOfDebugMarkersBegin)-1]
		} else {
			builder.labelsInsideDebugMarkers = append(builder.labelsInsideDebugMarkers, currentLabel)
		}
	}
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
	} else {
		label.PushBack(commandName, int(cmdId))
	}
	checkDebugMarkers(builder, label)
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
	checkDebugMarkers(builder, label)
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
	}
}
