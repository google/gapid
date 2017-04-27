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

package log_pb

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
)

// From returns a new protobuf Message constructed from the log.Message.
func From(m *log.Message) *Message {
	out := &Message{
		Text:     m.Text,
		Time:     &timestamp.Timestamp{Seconds: m.Time.Unix(), Nanos: int32(m.Time.Nanosecond())},
		Severity: Severity(m.Severity),
		Tag:      m.Tag,
		Process:  m.Process,
		Trace:    m.Trace,
	}
	for _, v := range m.Callstack {
		out.Callstack = append(out.Callstack, &SourceLocation{
			File: v.File, Line: v.Line,
		})
	}
	for _, v := range m.Values {
		p := pod.NewValue(v.Value)
		if p == nil {
			p = pod.NewValue(fmt.Sprint(v.Value))
		}
		out.Values = append(out.Values, &Value{Name: v.Name, Value: p})
	}
	return out
}

// Message returns a log.Message from the protobuf Message.
func (m *Message) Message() *log.Message {
	out := &log.Message{
		Text:     m.Text,
		Time:     time.Unix(m.Time.Seconds, int64(m.Time.Nanos)),
		Severity: log.Severity(m.Severity),
		Tag:      m.Tag,
		Process:  m.Process,
		Trace:    m.Trace,
	}
	for _, v := range m.Callstack {
		out.Callstack = append(out.Callstack, &log.SourceLocation{
			File: v.File, Line: v.Line,
		})
	}
	for _, v := range m.Values {
		out.Values = append(out.Values, &log.Value{Name: v.Name, Value: v.Value.Get()})
	}
	return out
}
