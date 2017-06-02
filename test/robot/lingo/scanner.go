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

package lingo

import (
	"bytes"
	"context"
	"regexp"
	"unicode/utf8"
)

type Skipper func(*Scanner)

// Marker is a stored point in a scanner that can be reset to.
type Marker struct {
	offset int
	count  int
}

// Scanner is a basic implementation of the functions used by generated parsers.
// It expects a the full byte slice of the source file.
type Scanner struct {
	ctx       context.Context
	name      string
	data      []byte
	offset    int
	skipper   Skipper
	skipping  bool
	records   *Records
	watermark scanError
}

// NewByteScanner builds a scanner over an input byte slice.
// If records is nil, the cst will not be maintained, otherwise it will be filled in with the parse Record list.
func NewByteScanner(ctx context.Context, name string, input []byte, records *Records) *Scanner {
	return &Scanner{ctx: ctx, name: name, data: input, records: records}
}

// NewStringScanner builds a scanner over an input string.
// If records is nil, the cst will not be maintained, otherwise it will be filled in with the parse Record list.
func NewStringScanner(ctx context.Context, name string, input string, records *Records) *Scanner {
	return NewByteScanner(ctx, name, []byte(input), records)
}

// WasOk is a helper function called by generated parser code.
// It is used to abandon the value result, and return true if there was no error.
// This is used in cases where the sub-parser is optional and the result is not needed.
func WasOk(_ interface{}, err error) bool {
	return err == nil
}

// SetSkip sets the skip function and returns the old one.
func (s *Scanner) SetSkip(skipper Skipper) Skipper {
	old := s.skipper
	s.skipper = skipper
	return old
}

// Skip invokes the current skip function if one is set.
func (s *Scanner) Skip() {
	if s.skipping {
		return
	}
	s.skipping = true
	s.skipper(s)
	s.skipping = false
}

// EOF returns true if the scanner has run out of input.
func (s *Scanner) EOF() bool {
	return s.offset >= len(s.data)
}

// Mark returns a new marker for the current scan position and state.
func (s *Scanner) Mark() Marker {
	m := Marker{offset: s.offset}
	if s.records != nil {
		m.count = len(*s.records)
	}
	return m
}

// PreMark returns an invalid marker, used for before the start sentinels.
func (s *Scanner) PreMark() Marker {
	return Marker{offset: -1}
}

// MustProgress panics if the marker does not move forwards.
// Used to catch when the grammar is broken.
func (s *Scanner) MustProgress(m Marker) (Marker, error) {
	err := error(nil)
	if m.offset >= s.offset {
		err = s.Error(nil, "Failed to make progress")
	}
	return s.Mark(), err
}

// Watermark returns the error that was generatd furthest into the parse stream.
// This is normally included in errors automatically, and often indicates the point where
// the best match failed, and thus the actual error in the source.
func (s *Scanner) Watermark() error {
	return s.watermark
}

// Register adds a node to the cst from the start marker to the current position.
func (s *Scanner) Register(start Marker, object interface{}) {
	if s.records == nil {
		return
	}
	// append to grow
	*s.records = append(*s.records, Record{})
	// shuffle up
	copy((*s.records)[start.count+1:], (*s.records)[start.count:])
	// insert the record
	(*s.records)[start.count] = Record{Start: start.offset, End: s.offset, Object: object}
}

// Reset puts the scanner back in the state it was when the start Marker was taken.
func (s *Scanner) Reset(start Marker) {
	s.offset = start.offset
	if s.records != nil {
		*s.records = (*s.records)[:start.count]
	}
}

// Rune is a parser for a single rune.
// If the next rune in the stream is a match, the rune will be consumed an error will be nil.
// Otherwise an error will be returned.
// In either case, the  requested rune is returned as the value.
func (s *Scanner) Rune(r rune) (rune, error) {
	v, size := utf8.DecodeRune(s.data[s.offset:])
	if v != r {
		return r, scanFailure
	}
	s.offset += size
	return v, nil
}

// Rune is a parser for a literal string.
// If literal string is next in the stream, the string will be consumed an error will be nil.
// Otherwise an error will be returned and the value will be the empty string.
func (s *Scanner) Literal(str string) (string, error) {
	data := []byte(str)
	remains := s.data[s.offset:]
	if len(data) > len(remains) || !bytes.Equal(data, remains[:len(data)]) {
		return "", scanFailure
	}
	s.offset += len(data)
	return str, nil
}

// Pattern is a parser for a regular expression.
// If the pattern matches the start of the stream, the matching string will be consumed and returned.
// Otherwise an error will be returned and the value will be the empty string.
func (s *Scanner) Pattern(re *regexp.Regexp) (string, error) {
	remains := s.data[s.offset:]
	match := re.FindIndex(remains)
	if match == nil {
		return "", scanFailure
	}
	s.offset += match[1]
	return string(remains[:match[1]]), nil
}
