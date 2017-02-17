// Copyright (C) 2017 Google Inc.
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

package langsvr

import "github.com/google/gapid/core/langsvr/protocol"

// Command represents a reference to a command.
// Provides a title which will be used to represent a command in the UI and,
// optionally, an array of arguments which will be passed to the command
// handler function when invoked.
type Command struct {
	// Title of the command, like `save`.
	Title string

	// The identifier of the actual command handler.
	Command string

	// Arguments that the command handler should be invoked with.
	Arguments map[string]interface{}
}

func (c Command) toProtocol() protocol.Command {
	return protocol.Command{
		Title:     c.Title,
		Command:   c.Command,
		Arguments: c.Arguments,
	}
}
