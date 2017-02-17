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

// Sequence is an implementation of Target that holds a list of 'consumable'
// targets. Once a target's Start() returns an error that is not a
// UnhandledCmdError, the result is returned and the target it is removed from
// the Sequence list.
type Sequence []shell.Target

func (s *Sequence) Start(cmd shell.Cmd) (shell.Process, error) {
	for i, t := range *s {
		process, err := t.Start(cmd)
		if _, unhandled := err.(UnhandledCmdError); !unhandled {
			copy((*s)[i:], (*s)[i+1:])
			*s = (*s)[:len(*s)-1]
			return process, err
		}
	}
	return nil, UnhandledCmdError(cmd)
}

func (Sequence) String() string { return "stub" }
