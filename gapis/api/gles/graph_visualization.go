// Copyright (C) 2019 Google Inc.
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

package gles

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/gapis/api"
)

// Interface compliance test
var (
	_ = api.GraphVisualizationBuilder(&labelForGlesCommands{})
	_ = api.GraphVisualizationAPI(API{})
)

type labelForGlesCommands struct{}

func (*labelForGlesCommands) GetCommandLabel(command api.Cmd, cmdId uint64) *api.Label {
	label := &api.Label{}
	commandName := command.CmdName()
	var commandIndexName bytes.Buffer
	fmt.Fprintf(&commandIndexName, "%s_%d", commandName, cmdId)
	label.PushBack(commandIndexName.String(), int(cmdId))
	return label
}

func (*labelForGlesCommands) GetSubCommandLabel(index api.SubCmdIdx, commandName string, cmdId uint64, subCommandName string) *api.Label {
	// TODO(hevrard): understand and implement
	panic("TODO gapis/api/gles:GetSubCommandLabel")
	return nil
}

func (API) GetGraphVisualizationBuilder() api.GraphVisualizationBuilder {
	return &labelForGlesCommands{}
}
