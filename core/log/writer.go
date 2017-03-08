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
	"os"
)

// Writer is a function that writes out a formatted log message.
type Writer func(text string, severity Severity)

// Std returns a Writer that writes to stdout if the message severity is less
// than an error, otherwise it writes to stderr.
func Std() Writer {
	return func(text string, severity Severity) {
		out := os.Stdout
		if severity >= Error {
			out = os.Stderr
		}
		out.WriteString(text)
		out.WriteString("\n")
	}
}

// Stdout returns a Writer that writes to stdout for all severities.
func Stdout() Writer {
	return func(text string, severity Severity) {
		out := os.Stdout
		out.WriteString(text)
		out.WriteString("\n")
	}
}

// Buffer returns a Writer that writes to the returned buffer.
func Buffer() (Writer, *bytes.Buffer) {
	buf, nl := &bytes.Buffer{}, false
	return func(text string, severity Severity) {
		buf.WriteString(text)
		if nl {
			buf.WriteString("\n")
		}
		nl = true
	}, buf
}
