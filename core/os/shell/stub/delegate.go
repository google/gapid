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

package stub

import "github.com/google/gapid/core/os/shell"

// Delegate is an implementation of Target that passes all command requests to a the first child that accepts
// them.
// A child target refuses a command by returning nil,nil from it's Start method.
type Delegate struct {
	// Handlers holds the set of possible command handlers.
	Handlers []shell.Target
}

func (t *Delegate) Start(cmd shell.Cmd) (shell.Process, error) {
	for _, handler := range t.Handlers {
		p, err := handler.Start(cmd)
		if _, unhandled := err.(UnhandledCmdError); !unhandled {
			return p, err
		}
	}
	return nil, UnhandledCmdError(cmd)
}

func (Delegate) String() string { return "stub" }
