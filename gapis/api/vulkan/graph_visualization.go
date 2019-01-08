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
	"fmt"
	"github.com/google/gapid/gapis/api"
)

// Interface compliance test
var (
	_ = api.GraphVisualizationBuilder(&labelForVulkanCommands{})
	_ = api.GraphVisualizationAPI(API{})
)

const (
	VK_BEGIN_COMMAND_BUFFER  = "vkBeginCommandBuffer"
	VK_CMD_BEGIN_RENDER_PASS = "vkCmdBeginRenderPass"
	VK_CMD_NEXT_SUBPASS      = "vkCmdNextSubpass"
	VK_COMMAND_BUFFER        = "vkCommandBuffer"
	VK_RENDER_PASS           = "vkRenderPass"
	VK_SUBPASS               = "vkSubpass"
	VK_END_COMMAND_BUFFER    = "vkEndCommandBuffer"
	VK_CMD_END_RENDER_PASS   = "vkCmdEndRenderPass"
	COMMAND_BUFFER           = "commandBuffer"
)

var (
	commandHierarchyNames    = getCommandHierarchyNames()
	subCommandHierarchyNames = getSubCommandHierarchyNames()
)

type labelForVulkanCommands struct {
	labelToHierarchy                    map[string]*api.Hierarchy
	subCommandIndexNameToHierarchyLabel map[string]string
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

func getCommandBuffer(command api.Cmd) string {
	parameters := command.CmdParams()
	for _, parameter := range parameters {
		if parameter.Name == COMMAND_BUFFER {
			commandBuffer := fmt.Sprintf("%s_%d", parameter.Name, parameter.Get())
			return commandBuffer
		}
	}
	return ""
}

func (builder *labelForVulkanCommands) GetCommandLabel(command api.Cmd, commandNodeId uint64) string {
	commandName := command.CmdName()
	label := ""
	if commandBuffer := getCommandBuffer(command); commandBuffer != "" {
		if _, ok := builder.labelToHierarchy[commandBuffer]; !ok {
			builder.labelToHierarchy[commandBuffer] = &api.Hierarchy{}
		}
		hierarchy := builder.labelToHierarchy[commandBuffer]
		label += commandBuffer + "/"
		label += getLabelFromHierarchy(commandName, commandHierarchyNames, hierarchy)
		label += fmt.Sprintf("%s_%d", commandName, commandNodeId)
	} else {
		label += fmt.Sprintf("%s_%d", commandName, commandNodeId)
	}
	return label
}

func (builder *labelForVulkanCommands) GetSubCommandLabel(index api.SubCmdIdx, commandName, subCommandName string) string {
	label := commandName
	subCommandIndexName := commandName
	for i := 1; i < len(index); i++ {
		subCommandIndexName += fmt.Sprintf("/%d", index[i])
		if i+1 < len(index) {
			if hierarchyLabel, ok := builder.subCommandIndexNameToHierarchyLabel[subCommandIndexName]; ok {
				label += "/" + hierarchyLabel
			} else {
				label += fmt.Sprintf("/%d", index[i])
			}
		}
	}
	if _, ok := builder.labelToHierarchy[label]; !ok {
		builder.labelToHierarchy[label] = &api.Hierarchy{}
	}
	hierarchy := builder.labelToHierarchy[label]
	labelFromHierarchy := getLabelFromHierarchy(subCommandName, subCommandHierarchyNames, hierarchy)
	labelFromHierarchy += fmt.Sprintf("%s_%d", subCommandName, index[len(index)-1])
	builder.subCommandIndexNameToHierarchyLabel[subCommandIndexName] = labelFromHierarchy

	label += "/" + labelFromHierarchy
	return label
}

func getLabelFromHierarchy(name string, hierarchyNames *api.HierarchyNames, hierarchy *api.Hierarchy) string {
	isEndCommand := false
	if level, ok := hierarchyNames.BeginNameToLevel[name]; ok {
		hierarchy.PushBackToResize(level + 1)
		hierarchy.IncreaseIDByOne(level)
	} else if level, ok := hierarchyNames.EndNameToLevel[name]; ok {
		hierarchy.PopBackToResize(level + 1)
		isEndCommand = true
	}

	label := ""
	for level := 1; level < hierarchy.GetSize(); level++ {
		label += fmt.Sprintf("%d_%s/", hierarchy.GetID(level), hierarchyNames.GetName(level))
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

func (API) GetGraphVisualizationBuilder() api.GraphVisualizationBuilder {
	return &labelForVulkanCommands{
		labelToHierarchy:                    map[string]*api.Hierarchy{},
		subCommandIndexNameToHierarchyLabel: map[string]string{},
	}
}
