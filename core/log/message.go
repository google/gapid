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

import (
	"strings"
	"time"
)

type Message struct {
	// The message text.
	Text string

	// The time the message was logged.
	Time time.Time

	// The severity of the message.
	Severity Severity

	// StopProcess is true if the message indicates the process should stop.
	StopProcess bool

	// The tag associated with the log record.
	Tag string

	// The name of the process that created the record.
	Process string

	// The callstack at the time the message was logged.
	Callstack []*SourceLocation

	// The stack of enter() calls at the time the message was logged.
	Trace Trace

	// The key-value pairs of extra data.
	Values Values
}

type SourceLocation struct {
	// The file name
	File string
	// 1-based line number
	Line int32
}

type Trace []string

func (t Trace) String() string { return strings.Join(t, "⇒") }

type Values []*Value

func (v Values) Len() int           { return len(v) }
func (v Values) Less(i, j int) bool { return v[i].Name < v[j].Name }
func (v Values) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

type Value struct {
	Name  string
	Value interface{}
}
