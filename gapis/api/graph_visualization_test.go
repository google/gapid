// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package api

import (
	"reflect"
	"testing"
)

func TestLabel(t *testing.T) {

	label := &Label{}
	label.PushBack("name1", 1)
	label.PushBack("name2", 2)
	label.PushBack("name3", 3)
	label.PushBack("name4", 4)
	label.PushFront("name5", 5)
	label.PushFront("name6", 6)

	obtainedLabel := label

	wantedLabel := &Label{
		LevelsName: []string{"name6", "name5", "name1", "name2", "name3", "name4"},
		LevelsID:   []int{6, 5, 1, 2, 3, 4},
	}

	if !reflect.DeepEqual(wantedLabel, obtainedLabel) {
		t.Errorf("The Label is different\n")
		t.Errorf("Wanted %v, obtained %v\n", wantedLabel, obtainedLabel)
	}

	label.Insert(0, "name10", 10)
	label.Insert(5, "name11", 11)
	label.Insert(5, "name12", 12)

	wantedLabel = &Label{
		LevelsName: []string{"name10", "name6", "name5", "name1", "name2", "name12", "name11", "name3", "name4"},
		LevelsID:   []int{10, 6, 5, 1, 2, 12, 11, 3, 4},
	}

	if !reflect.DeepEqual(wantedLabel, obtainedLabel) {
		t.Errorf("The Label is different\n")
		t.Errorf("Wanted %v, obtained %v\n", wantedLabel, obtainedLabel)
	}

	obtainedLabel.PushBackLabel(obtainedLabel)
	wantedLabel = &Label{
		LevelsName: []string{"name10", "name6", "name5", "name1", "name2", "name12", "name11", "name3", "name4",
			"name10", "name6", "name5", "name1", "name2", "name12", "name11", "name3", "name4"},
		LevelsID: []int{10, 6, 5, 1, 2, 12, 11, 3, 4,
			10, 6, 5, 1, 2, 12, 11, 3, 4},
	}

	if !reflect.DeepEqual(wantedLabel, obtainedLabel) {
		t.Errorf("The Label is different\n")
		t.Errorf("Wanted %v, obtained %v\n", wantedLabel, obtainedLabel)
	}

	if obtainedLabel.GetCommandName() != "name4" {
		t.Errorf("The command name is different")
	}
	if obtainedLabel.GetCommandId() != 4 {
		t.Errorf("The command ID is different")
	}
	if obtainedLabel.GetTopLevelName() != "name10" {
		t.Errorf("The top name is different")
	}
	if obtainedLabel.GetTopLevelID() != 10 {
		t.Errorf("The top ID is different")
	}
	wantedLabelAsAString := "name1010/name66/name55/name11/name22/name1212/name1111/name33/name44/" +
		"name1010/name66/name55/name11/name22/name1212/name1111/name33/name44"
	if obtainedLabel.GetLabelAsAString() != wantedLabelAsAString {
		t.Errorf("The LabelAsAString is different")
		t.Errorf("Wanted %v, obtained %v\n", wantedLabelAsAString, obtainedLabel.GetLabelAsAString())
	}
}
