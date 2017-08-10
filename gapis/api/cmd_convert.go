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

package api

import (
	"github.com/golang/protobuf/proto"
)

// CmdWithResult is the optional interface implemented by commands that have
// a result value.
type CmdWithResult interface {
	Cmd

	// GetResult returns the result value for this command.
	GetResult() proto.Message

	// SetResult changes the result value. Returns an error if the result proto
	// type does not match this command.
	SetResult(proto.Message) error
}

// CmdCallFor returns the proto message type for the call result of cmd.
func CmdCallFor(cmd Cmd) proto.Message {
	if cmd, ok := cmd.(CmdWithResult); ok {
		return cmd.GetResult()
	}
	return &CmdCall{}
}
