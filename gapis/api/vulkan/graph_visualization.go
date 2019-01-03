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
	beginCommands = map[string]int{
		VK_BEGIN_COMMAND_BUFFER:  1,
		VK_CMD_BEGIN_RENDER_PASS: 2,
		VK_CMD_NEXT_SUBPASS:      3,
	}
	listOfCommandNames = []string{
		VK_COMMAND_BUFFER,
		VK_RENDER_PASS,
		VK_SUBPASS,
	}
	endCommands = map[string]int{
		VK_END_COMMAND_BUFFER:  1,
		VK_CMD_END_RENDER_PASS: 2,
		VK_CMD_NEXT_SUBPASS:    3,
	}
)

func getCommandBuffer(command api.Cmd) string {
	parameters := command.CmdParams()
	for _, parameter := range parameters {
		if parameter.Name == COMMAND_BUFFER {
			commandBuffer := parameter.Name + fmt.Sprintf("%d", parameter.Get()) + "/"
			return commandBuffer
		}
	}
	return ""
}

func (API) GetCommandLabel(hierarchy *api.Hierarchy, command api.Cmd) string {
	commandName := command.CmdName()
	isEndCommand := false
	if level, ok := beginCommands[commandName]; ok {
		hierarchy.PushBackToResize(level + 1)
		hierarchy.IncreaseIDByOne(level)
	} else {
		if level, ok := endCommands[commandName]; ok {
			hierarchy.PopBackToResize(level + 1)
			isEndCommand = true
		}
	}

	label := getCommandBuffer(command)
	for level := 1; level < hierarchy.GetSize(); level++ {
		label += fmt.Sprintf("%s%d/", listOfCommandNames[level-1], hierarchy.GetID(level))
	}

	if level, ok := beginCommands[commandName]; ok && commandName == VK_CMD_BEGIN_RENDER_PASS {
		levelForSubpass := level + 1
		hierarchy.PushBackToResize(levelForSubpass + 1)
		hierarchy.IncreaseIDByOne(levelForSubpass)
	}
	if isEndCommand {
		hierarchy.PopBack()
	}
	return label
}

func (API) GetSubCommandLabel(index api.SubCmdIdx) string {
	label := ""
	for i := 1; i < len(index); i++ {
		label += fmt.Sprintf("/%d", index[i])
	}
	return label
}
