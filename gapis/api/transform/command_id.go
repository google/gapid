// Copyright (C) 2020 Google Inc.
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

// Package transform contains the elements to be able to transform
// commands which consist of interfaces for individual transform operations
// and a transform chain to run all of them.
package transform

import (
	"github.com/google/gapid/gapis/api"
)

type CommandType uint32

const (
	TransformCommand CommandType = 0
	EndCommand       CommandType = 1
)

func (cmdType CommandType) String() string {
	switch cmdType {
	case TransformCommand:
		return "Transform Command"
	case EndCommand:
		return "End Command"
	default:
		panic("Unknown Command Type")
	}
}

type CommandID struct {
	id          api.CmdID
	commandType CommandType
}

func NewTransformCommandID(id api.CmdID) CommandID {
	return CommandID{
		id:          id,
		commandType: TransformCommand,
	}
}

func NewEndCommandID() CommandID {
	return CommandID{
		id:          0,
		commandType: EndCommand,
	}
}

func (c *CommandID) GetID() api.CmdID {
	if c.commandType != TransformCommand {
		panic("cmdID should only exist for transform commands")
	}

	return c.id
}

func (c *CommandID) Increment() {
	if c.commandType != TransformCommand {
		panic("cmdID should only exist for transform commands")
	}

	c.id++
}

func (c *CommandID) GetCommandType() CommandType {
	return c.commandType
}
