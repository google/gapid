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

package executor

import (
	"fmt"

	"github.com/google/gapid/gapis/api"
)

type Encodable interface {
	Encode([]byte) bool
}

func encodeCommand(cmd api.Cmd, buf []byte) {
	e, ok := cmd.(Encodable)
	if !ok {
		panic(fmt.Errorf("Command '%v' is not encodable", cmd.CmdName()))
	}
	if !e.Encode(buf) {
		panic(fmt.Errorf("Failed to encode command '%v'", cmd.CmdName()))
	}
}
