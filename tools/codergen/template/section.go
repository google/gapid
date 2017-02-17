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

package template

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/google/gapid/core/text/reflow"
)

const (
	sectionMarker    = `<<<%s:%s:%s>>>`
	sectionStart     = `Start`
	sectionEnd       = `End`
	sectionParameter = `([^\:>]+)`
)

var (
	section = regexp.MustCompile(fmt.Sprintf(sectionMarker, sectionParameter, sectionParameter, sectionParameter))
)

// Section writes a new file section with associated templates.
// It takes the name of the template to use to fill the section in, and wraps it
// in the section markers.
func (t *Templates) Section(name string) (string, error) {
	w, ok := t.writer.(*reflow.Writer)
	if !ok {
		return "", fmt.Errorf("Section called inside nested writer")
	}
	depth := strconv.Itoa(w.Depth)
	io.WriteString(t.writer, fmt.Sprintf(sectionMarker, sectionStart, name, depth))
	err := t.execute(name, t.writer, t.File)
	io.WriteString(t.writer, fmt.Sprintf(sectionMarker, sectionEnd, name, depth))
	return "", err
}

// Section represents a span of text in a file, segmented using SectionSplit.
// If the section has a Name it will also have a StartMarker and EndMarker, otherwise
// they will be empty.
type Section struct {
	Name        string // The name the section was given.
	Indentation int    // The indentation level of the section.
	StartMarker []byte // The bytes that marked the start of the section.
	Body        []byte // The content of the section.
	EndMarker   []byte // The bytes that marked the end of the section.
}

// SectionSplit breaks a file up into it's sections, and returns the slices with
// the section header.
// Named sections are delimited by <<<Start:name:depth>>> to <<<End:name:depth>>>
// markers, areas between named sections are returned as unnamed sections.
func SectionSplit(data []byte) ([]Section, error) {
	matches := section.FindAllSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	s := Section{}
	sections := make([]Section, 0, len(matches)+1)
	last := 0
	// we are doing a partial update...
	for _, match := range matches {
		marker := data[match[0]:match[1]]
		mode := string(data[match[2]:match[3]])
		name := string(data[match[4]:match[5]])
		level := string(data[match[6]:match[7]])
		depth, err := strconv.Atoi(string(data[match[6]:match[7]]))
		if err != nil {
			return nil, fmt.Errorf("Indentation depth malformed, got %s in %s", level, name)
		}
		switch mode {
		case sectionStart:
			if s.Name != "" {
				return nil, fmt.Errorf("Overlapping template %s found starting %s", s.Name, name)
			}
			// section start, add the prefix section
			if last != match[0] {
				s.Body = data[last:match[0]]
				sections = append(sections, s)
			}
			// And prepare the section itself
			s = Section{Name: name, Indentation: depth, StartMarker: marker}
			last = match[1]
		case sectionEnd:
			// section end marker, check it matches
			if name != s.Name {
				return nil, fmt.Errorf("Invalid end %s found, expected %s", name, s.Name)
			}
			// write the end marker out throught the formatting writer
			s.Body = data[last:match[0]]
			s.EndMarker = marker
			sections = append(sections, s)
			// prepare for the next section
			s = Section{}
			last = match[1]
		default:
			return nil, fmt.Errorf("Invalid section marker %q", mode)
		}
	}
	if s.Name != "" {
		return nil, fmt.Errorf("Unclosed template %s found", s.Name)
	}
	// section start, add the prefix section
	if last != len(data) {
		s.Body = data[last:]
		sections = append(sections, s)
	}
	return sections, nil
}
