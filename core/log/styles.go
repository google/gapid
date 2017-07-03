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

var (
	// Raw is a style that only prints the text of the message.
	Raw = Style{
		Name:      "raw",
		Timestamp: false,
		Tag:       false,
		Trace:     false,
		Process:   false,
		Severity:  NoSeverity,
		Values:    NoValues,
	}

	// Brief is a style that only prints the text and short severity of the
	// message.
	Brief = Style{
		Name:      "brief",
		Timestamp: false,
		Tag:       false,
		Trace:     false,
		Process:   false,
		Severity:  SeverityShort,
		Values:    NoValues,
	}

	// Normal is a style that prints the timestamp, tag, trace, process,
	// short severity.
	Normal = Style{
		Name:      "normal",
		Timestamp: true,
		Tag:       true,
		Trace:     true,
		Process:   true,
		Severity:  SeverityShort,
		Values:    NoValues,
	}

	// Detailed is a style that prints the timestamp, tag, trace, process,
	// long severity and multi-line values.
	Detailed = Style{
		Name:      "detailed",
		Timestamp: true,
		Tag:       true,
		Trace:     true,
		Process:   true,
		Severity:  SeverityLong,
		Values:    ValuesMultiLine,
	}
)

func init() {
	RegisterStyle(Raw)
	RegisterStyle(Brief)
	RegisterStyle(Normal)
	RegisterStyle(Detailed)
}
