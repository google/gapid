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

package log

import "github.com/google/gapid/core/app/flags"

// Severity defines the severity of a logging message.
type Severity int32

// The values must be identical to values in gapis/service/service.proto
const (
	// Verbose indicates extremely verbose level messages.
	Verbose Severity = 0
	// Debug indicates debug-level messages.
	Debug Severity = 1
	// Info indicates minor informational messages that should generally be ignored.
	Info Severity = 2
	// Warning indicates issues that might affect performance or compatibility, but could be ignored.
	Warning Severity = 3
	// Error indicates non terminal failure conditions that may have an effect on results.
	Error Severity = 4
	// Fatal indicates a fatal error.
	Fatal Severity = 5
)

func (s Severity) String() string {
	switch s {
	case Verbose:
		return "Verbose"
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	case Fatal:
		return "Fatal"
	}
	return "?"
}

// Short returns the severity string with a single character.
func (s Severity) Short() string {
	switch s {
	case Verbose:
		return "V"
	case Debug:
		return "D"
	case Info:
		return "I"
	case Warning:
		return "W"
	case Error:
		return "E"
	case Fatal:
		return "F"
	}
	return "?"
}

// Choose allows *Severity to be used as a command line flag.
func (s *Severity) Choose(c interface{}) { *s = c.(Severity) }

// Chooser returns a chooser for the set of severities.
func (s *Severity) Chooser() flags.Chooser {
	return flags.Chooser{
		Value:   s,
		Choices: flags.Choices{Verbose, Debug, Info, Warning, Error, Fatal},
	}
}
