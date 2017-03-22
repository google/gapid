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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/google/gapid/core/app/flags"
)

// Style provides customization for printing messages.
type Style struct {
	Name      string        // Name of the style.
	Timestamp bool          // If true, the timestamp will be printed if part of the message.
	Tag       bool          // If true, the tag will be printed if part of the message.
	Trace     bool          // If true, the trace will be printed if part of the message.
	Process   bool          // If true, the process will be printed if part of the message.
	Severity  SeverityStyle // How the severity of the message will be printed.
	Values    ValueStyle    // How the values of the message will be printed.
}

// SeverityStyle is an enumerator of ways that severities can be printed.
type SeverityStyle int

const (
	// NoSeverity is the option to disable the printing of the severity.
	NoSeverity = SeverityStyle(iota)
	// SeverityShort is the option to display the severity as a single character.
	SeverityShort
	// SeverityLong is the option to display the severity in its full name.
	SeverityLong
)

func (ss SeverityStyle) print(s Severity) string {
	if ss == SeverityShort {
		return s.Short()
	}
	return s.String()
}

// ValueStyle is an enumerator of ways that values can be printed.
type ValueStyle int

const (
	// NoValues is the option to disable the printing of values.
	NoValues = ValueStyle(iota)
	// ValuesSingleLine is the option to display all values on a single line.
	ValuesSingleLine
	// ValuesMultiLine is the option to display each value on a separate line.
	ValuesMultiLine
)

func (vs ValueStyle) print(v Values) string {
	switch vs {
	case ValuesSingleLine:
		t := make([]string, len(v))
		for i, v := range v {
			t[i] = fmt.Sprintf("%v: %v", v.Name, v.Value)
		}
		return fmt.Sprintf("(%v)", strings.Join(t, ", "))
	case ValuesMultiLine:
		buf := bytes.Buffer{}
		for _, v := range v {
			buf.WriteString(fmt.Sprintf("\n  %v: %v", v.Name, v.Value))
		}
		return buf.String()
	}
	return ""
}

var styles flags.Choices

func (s Style) String() string { return s.Name }

// Choose sets the style to the supplied choice.
func (s *Style) Choose(v interface{}) { *s = v.(Style) }

// Chooser returns a chooser for the set of registered styles
func (s *Style) Chooser() flags.Chooser { return flags.Chooser{Value: s, Choices: styles} }

// RegisterStyle registers the style s. The list of registered styles can be
// displayed in command line flags.
func RegisterStyle(s Style) { styles = append(styles, s) }

// Handler returns a new Handler configured to write to out and err with the
// given style.
func (s Style) Handler(w Writer) Handler {
	return handler{
		handle: func(msg *Message) {
			var parts [8]string
			m := append(parts[:0])
			if s.Timestamp && !msg.Time.IsZero() {
				m = append(m, HHMMSSsss(msg.Time))
			}
			if s.Severity != NoSeverity {
				m = append(m, s.Severity.print(msg.Severity)+":")
			}
			if s.Trace && len(msg.Trace) > 0 {
				m = append(m, fmt.Sprintf("[%s]", msg.Trace))
			}
			if s.Tag && msg.Tag != "" {
				m = append(m, fmt.Sprintf("[%s]", msg.Tag))
			}
			if s.Process && msg.Process != "" {
				m = append(m, fmt.Sprintf("<%s>", msg.Process))
			}
			m = append(m, msg.Text)
			if s.Values != NoValues && len(msg.Values) > 0 {
				m = append(m, s.Values.print(msg.Values))
			}
			w(strings.Join(m, " "), msg.Severity)
		},
		close: func() {},
	}
}

// Print returns the message msg printed with the style s.
func (s Style) Print(msg *Message) string {
	w, b := Buffer()
	s.Handler(w).Handle(msg)
	return string(b.String())
}

// HHMMSSsss prints the time as a HH:MM:SS.sss
func HHMMSSsss(t time.Time) string {
	return fmt.Sprintf("%.2d:%.2d:%.2d.%.3d", t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e6)
}
