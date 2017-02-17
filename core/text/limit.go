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

package text

import (
	"io"
	"unicode/utf8"
)

type (
	// LimitWriter limits the length of string written through it.
	LimitWriter struct {
		to            io.Writer
		remains       int
		elisionMarker string
		elisionLength int
		unwritten     []byte
	}
)

// NewLimitWriter returns an io.Writer that limits the amount of text written through it
// and optionally adds a marker if it causes the limit to be reached.
func NewLimitWriter(to io.Writer, limit int, elisionMarker string) *LimitWriter {
	return &LimitWriter{
		to:            to,
		remains:       limit,
		elisionMarker: elisionMarker,
		elisionLength: utf8.RuneCountInString(elisionMarker),
	}
}

// Write will write as much of the string as it can to the underlying writer, if
// he total number of runes written would exceed the limit, the last part of the
// string is replaced by the elisionMarker.
// This method may buffer data, you must call Flush for it all to appear in the
// underlying stream.
func (w *LimitWriter) Write(data []byte) (int, error) {
	if w.remains <= 0 {
		return len(data), nil
	}
	//TODO: presumes that no runes are torn across bytes...
	runeLength := utf8.RuneCount(data)
	if len(w.unwritten) > 0 {
		//we have spare data from the previous read, which means we were near the boundary
		oldLength := utf8.RuneCount(w.unwritten)
		if runeLength+oldLength > w.remains {
			// does not fit
			w.remains = 0
			_, err := w.to.Write(([]byte)(w.elisionMarker))
			return len(data), err
		}
		// add to the unwritten block, it might still fit
		w.unwritten = append(w.unwritten, data...)
		return len(data), nil
	}
	if runeLength <= 0 {
		return 0, nil
	}
	safe := w.remains - w.elisionLength
	if runeLength < safe {
		// it all fits, so write it out
		w.remains -= runeLength
		return w.to.Write(data)
	}
	// might not fit, work out the safe amount
	seek := data
	writeLen := 0
	for ; safe > 0; safe-- {
		_, size := utf8.DecodeRune(seek)
		seek = seek[size:]
		writeLen++
	}
	// write out the bytes up to the unsafe area
	w.remains -= writeLen
	data = data[:len(data)-len(seek)]
	_, err := w.to.Write(data)
	if err == nil && runeLength-writeLen > w.elisionLength {
		// not enough space for the rest
		w.remains = 0
		_, err = w.to.Write(([]byte)(w.elisionMarker))
	}
	w.unwritten = append([]byte{}, seek...)
	return len(data), err
}

// Flush any writes to the underlying stream.
// It also makes the writer assume the limit has been reached
// for the purpose of any further writes.
func (w *LimitWriter) Flush() error {
	if w.remains <= 0 {
		return nil
	}
	_, err := w.to.Write(w.unwritten)
	return err
}
