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

// This is the error type returned when a command is not handled by the stub target.
type UnhandledCmdError shell.Cmd

func (u UnhandledCmdError) Error() string {
	return fmt.Sprint("unmatched:", shell.Cmd(u))
}

// OneOf returns a Delegate that uses the supplied handlers.
func OneOf(handlers ...shell.Target) shell.Target {
	return &Delegate{Handlers: handlers}
}

// Respond returns the simplest type of Response that just writes the response to the stdout.
func Respond(response string) shell.Target {
	return &Response{Stdout: response}
}

// Match returns a MatchTarget that compares the command and delegates to handler if it matches.
func Match(command string, handler shell.Target) shell.Target {
	return &MatchTarget{Match: command, Target: handler}
}

// Regex returns a RegexTarget that compares the command to the pattern and delegates to handler if it matches.
func Regex(pattern string, handler shell.Target) shell.Target {
	return &RegexpTarget{Match: regexp.MustCompile(pattern), Target: handler}
}

// RespondTo builds a target that matches the supplied command and writes the response to stdout if it matches.
func RespondTo(command string, response string) shell.Target {
	return Match(command, Respond(response))
}
