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

import (
	"fmt"

	"regexp"

	"github.com/google/gapid/core/os/shell"
)

// MatchTarget is an implementation of Target that passes the command to it's child target if it matches the supplied
// command.
type MatchTarget struct {
	// Match is the command line to match
	Match string
	// Target is the underlying target to dispatch to if this target matches the command.
	Target shell.Target
}

func (t *MatchTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	if t.Match != fmt.Sprint(cmd) {
		return nil, UnhandledCmdError(cmd)
	}
	return t.Target.Start(cmd)
}

// RegexpTarget is an implementation of Target that passes the command to it's child target if it matches the supplied
// regexp match pattern.
type RegexpTarget struct {
	// Match is the regular expression for the command line to match
	Match *regexp.Regexp
	// Target is the underlying target to dispatch to if this target matches the command.
	Target shell.Target
}

func (t *RegexpTarget) Start(cmd shell.Cmd) (shell.Process, error) {
	if !t.Match.MatchString(fmt.Sprint(cmd)) {
		return nil, UnhandledCmdError(cmd)
	}
	return t.Target.Start(cmd)
}
