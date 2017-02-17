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

package note

import (
	"bytes"
	"io"

	"github.com/google/gapid/core/app/flags"
)

type (
	// Scribe is the type for something that can make a new Handler that formats notes
	// onto an io stream.
	Scribe func(io.Writer) Handler

	// Style can be used as a flag value, allowing selection of scribe by name.
	Style struct {
		// Name of the choice.
		Name string
		// Style the choice represents.
		Scribe Scribe
	}
)

var (
	// Raw is a style that prints the critial values and no keys only.
	Raw = Style{
		Name: "Raw",
		Scribe: printer(&format{
			include: Critical,
			noKeys:  true,
		}),
	}

	// Brief is a short form style that prints only critical and important entries.
	Brief = Style{
		Name: "Brief",
		Scribe: printer(&format{
			include: Important,
			limit:   64,
			elide:   "...",
		}),
	}

	// Normal is a style that prints critical, important and relevant values all on one line.
	Normal = Style{
		Name: "Normal",
		Scribe: printer(&format{
			include: Relevant,
			limit:   128,
			elide:   "...",
		}),
	}

	// Detailed is a style that prints all values, allowing multi-line sections.
	Detailed = Style{
		Name: "Detailed",
		Scribe: printer(&format{
			include:   Irrelevant,
			multiline: true,
			indent:    "    ",
		}),
	}

	// Canonical is a complete print, but in a fairly compact format.
	// This is useful for things like tests.
	Canonical = Style{
		Name:   "Canonical",
		Scribe: canonical,
	}

	// JSON is a style that prints all values, with each detail on its own line.
	JSON = Style{
		Name:   "JSON",
		Scribe: jsonStyle{Compact: false}.Scribe,
	}

	// CompactJSON is a style that produces a readable but parseable output.
	CompactJSON = Style{
		Name:   "CompactJSON",
		Scribe: jsonStyle{Compact: true}.Scribe,
	}

	// DefaultStyle is the style used when none is provided.
	DefaultStyle = Normal

	styleList flags.Choices
)

// RegisterStyle adds a new style to the set of style choices.
// You can use styles without registering them, this just allows for a set of
// styles that can be chosen by name.
func RegisterStyle(style Style) {
	styleList = append(styleList, style)
}

// String returns the name of the style chosen.
func (s Style) String() string { return s.Name }

// Choose sets the style to the supplied choice.
func (s *Style) Choose(v interface{}) { *s = v.(Style) }

// Chooser returns a chooser for the set of registered styles
func (s *Style) Chooser() flags.Chooser { return flags.Chooser{Value: s, Choices: styleList} }

// Print is used to convert a page to a string using the brief style.
func (s Style) Print(page Page) string {
	buf := &bytes.Buffer{}
	s.Scribe(buf)(page)
	return buf.String()
}

// Print is used to convert a page to a string using the default style.
func Print(page Page) string {
	return DefaultStyle.Print(page)
}

func init() {
	RegisterStyle(Raw)
	RegisterStyle(Brief)
	RegisterStyle(Normal)
	RegisterStyle(Detailed)
	RegisterStyle(Canonical)
	RegisterStyle(JSON)
	RegisterStyle(CompactJSON)
}
