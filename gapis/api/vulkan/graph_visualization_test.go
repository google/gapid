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
	"github.com/google/gapid/gapis/api"
	"reflect"
	"testing"
)

const (
	VK_CMD_BIND_DESCRIPTOR_SETS = "vkCmdBindDescriptorSets"
)

func TestGetLabelFromHierarchy1(t *testing.T) {

	commandHierarchyNames := getCommandHierarchyNames()

	auxiliarCommands := []string{
		"SetViewPort",
		"SetBarrier",
		"DrawIndex",
		"SetScissor",
		"SetIndexForDraw",
	}

	commandsName := []string{
		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[2],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[0],
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		VK_END_COMMAND_BUFFER,

		auxiliarCommands[3],
		auxiliarCommands[4],

		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[1],
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_END_COMMAND_BUFFER,

		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[4],
		VK_END_COMMAND_BUFFER,
	}
	wantedLabels := []*api.Label{
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{},
		&api.Label{},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
	}

	currentHierarchy := &api.Hierarchy{}
	obtainedLabels := []*api.Label{}
	for _, name := range commandsName {
		label := getLabelFromHierarchy(name, commandHierarchyNames, currentHierarchy)
		obtainedLabels = append(obtainedLabels, label)
	}
	for i := range wantedLabels {
		if !reflect.DeepEqual(wantedLabels[i], obtainedLabels[i]) {
			t.Errorf("The label for command %s with id %d is different", commandsName[i], i)
			t.Errorf("Obtained %v, wanted %v", obtainedLabels[i], wantedLabels[i])
		}
	}

}

func TestGetLabelFromHierarchy2(t *testing.T) {

	commandHierarchyNames := getCommandHierarchyNames()

	auxiliarCommands := []string{
		"SetViewPort",
		"SetBarrier",
		"DrawIndex",
		"SetScissor",
		"SetIndexForDraw",
	}

	commandsName := []string{
		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[2],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[0],
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_END_COMMAND_BUFFER,

		auxiliarCommands[3],
		auxiliarCommands[4],

		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[2],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[0],
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_END_COMMAND_BUFFER,

		auxiliarCommands[3],
		auxiliarCommands[4],

		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[2],
		auxiliarCommands[1],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[0],
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_END_COMMAND_BUFFER,

		auxiliarCommands[3],
		auxiliarCommands[4],
	}
	wantedLabels := []*api.Label{
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{},
		&api.Label{},

		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{2, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{2, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{2, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{},
		&api.Label{},

		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{3, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{3, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{3, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{3}},
		&api.Label{},
		&api.Label{},
	}

	currentHierarchy := &api.Hierarchy{}
	obtainedLabels := []*api.Label{}
	for _, name := range commandsName {
		label := getLabelFromHierarchy(name, commandHierarchyNames, currentHierarchy)
		obtainedLabels = append(obtainedLabels, label)
	}
	for i := range wantedLabels {
		if !reflect.DeepEqual(wantedLabels[i], obtainedLabels[i]) {
			t.Errorf("The label for command %s with id %d is different", commandsName[i], i)
			t.Errorf("Obtained %v, wanted %v", obtainedLabels[i], wantedLabels[i])
		}
	}
}

func TestGetLabelFromHierarchy3(t *testing.T) {

	commandHierarchyNames := getCommandHierarchyNames()

	auxiliarCommands := []string{
		"SetViewPort",
		"SetBarrier",
		"DrawIndex",
		"SetScissor",
		"SetIndexForDraw",
	}

	commandsName := []string{
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_BEGIN_COMMAND_BUFFER,
		auxiliarCommands[0],
		auxiliarCommands[1],
		VK_CMD_BEGIN_RENDER_PASS,
		auxiliarCommands[1],
		auxiliarCommands[2],
		auxiliarCommands[3],
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[1],
		auxiliarCommands[2],
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_NEXT_SUBPASS,
		auxiliarCommands[1],
		auxiliarCommands[2],
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_NEXT_SUBPASS,
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[0],
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		auxiliarCommands[3],
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		VK_CMD_BEGIN_RENDER_PASS,
		VK_CMD_END_RENDER_PASS,
		VK_END_COMMAND_BUFFER,
		auxiliarCommands[2],
		auxiliarCommands[1],
		VK_BEGIN_COMMAND_BUFFER,
		VK_END_COMMAND_BUFFER,
		auxiliarCommands[1],
	}
	wantedLabels := []*api.Label{
		&api.Label{},
		&api.Label{},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 1, 5}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 2, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 2, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 2, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS}, LevelsID: []int{1, 2, 5}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 3}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 4}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 5}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 5}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 6}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS}, LevelsID: []int{1, 6}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{1}},
		&api.Label{},
		&api.Label{},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{LevelsName: []string{VK_COMMAND_BUFFER}, LevelsID: []int{2}},
		&api.Label{},
	}

	currentHierarchy := &api.Hierarchy{}
	obtainedLabels := []*api.Label{}
	for _, name := range commandsName {
		label := getLabelFromHierarchy(name, commandHierarchyNames, currentHierarchy)
		obtainedLabels = append(obtainedLabels, label)
	}
	for i := range wantedLabels {
		if !reflect.DeepEqual(wantedLabels[i], obtainedLabels[i]) {
			t.Errorf("The label for command %s with id %d is different", commandsName[i], i)
			t.Errorf("Obtained %v, wanted %v", obtainedLabels[i], wantedLabels[i])
		}
	}
}

func TestDebugMarkers1(t *testing.T) {

	dm := &debugMarkerStack{}
	input := []*api.Label{
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 4},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 6},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 7},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 8},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 9},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 10},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 11},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 12},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 20},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 21},
		},
	}
	wantedOutput := []*api.Label{
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 3, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 3, 2, 4},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 3, 2, 1, 6},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 3, 2, 1, 7},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 3, 2, 1, 8},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 1, 1, 3, 2, 1, 9},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 3, 2, 1, 10},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 3, 2, 11},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 1, 3, 12},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 20},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_SUBPASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1, 21},
		},
	}

	for _, label := range input {
		dm.checkDebugMarkers(label)
	}
	obtainedOutput := input

	if !reflect.DeepEqual(wantedOutput, obtainedOutput) {
		t.Errorf("The debug markers were added incorrectly\n")
	}
}

func TestDebugMarkers2(t *testing.T) {

	dm := &debugMarkerStack{}
	input := []*api.Label{
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 2},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 4},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 2, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 2, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 2, 5},
		},

		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 7},
		},
	}

	wantedOutput := []*api.Label{
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 1, 2},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 1, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 3, 4},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_BEGIN},
			LevelsID:   []int{1, 3, 2, 2, 1},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER, VK_CMD_BIND_DESCRIPTOR_SETS},
			LevelsID:   []int{1, 3, 2, 2, 3},
		},
		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_RENDER_PASS, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 3, 2, 2, 5},
		},

		&api.Label{
			LevelsName: []string{VK_COMMAND_BUFFER, VK_CMD_DEBUG_MARKER, VK_CMD_DEBUG_MARKER_END},
			LevelsID:   []int{1, 3, 7},
		},
	}

	for _, label := range input {
		dm.checkDebugMarkers(label)
	}
	obtainedOutput := input

	if !reflect.DeepEqual(wantedOutput, obtainedOutput) {
		t.Errorf("The debug markers were added incorrectly\n")
	}
}
