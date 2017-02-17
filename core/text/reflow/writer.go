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

package reflow

import (
	"bytes"
	"io"
	"text/tabwriter"
	"unicode/utf8"
)

const (
	EOL      = '¶' // outputs a line break
	Space    = '•' // outputs a space even where it would normally be stripped
	Column   = '║' // a column marker used to line up text
	Toggle   = '§' // switches the formatter on or off
	Disable  = '⋖' // switches the formatter off
	Enable   = '⋗' // switches the formatter back on again
	Indent   = '»' // increases the current indent level by 1
	Unindent = '«' // decreases the current indent level by 1
	Flush    = 'ø' // flushes text, resets column detection
)

type (
	// Writer is an io.Writer that uses unicode markup to reflow the text passing
	// through it.
	Writer struct {
		Rules     []Rule // The set of rune actions to apply.
		Depth     int    // The current indentation depth.
		Indent    string // The string to repeat as the indentation.
		Disabled  rune   // The current disabled state of the writer.
		Out       io.Writer
		Tabs      *tabwriter.Writer
		To        io.Writer
		space     bytes.Buffer
		lastDepth int
		newline   bool
		stripping bool
		runeBuf   [4]byte
	}

	// Action is the function type invoked for runes being processed by the writer.
	// It is handed the writer and the rune that triggered it.
	Action func(w *Writer, r rune) error

	// Rule is an entry in the rule set that maps a rune to an action.
	Rule struct {
		// Rune is the rune to activate the binding for.
		Rune rune
		// Aciton is the funciton to invoke for the matching rune.
		Action
	}
)

// New constructs a new reflow Writer with the default indent of 2 spaces.
// A copy of the default rules are pre-installed in the bindings, and can be modifed at any time.
// See AppendDefaultRules for the set of rules installed.
func New(to io.Writer) *Writer {
	w := &Writer{
		To:     to,
		Tabs:   tabwriter.NewWriter(to, 1, 2, 1, ' ', tabwriter.StripEscape),
		Indent: "  ",
		Rules:  AppendDefaultRules(nil),
	}
	w.Out = w.Tabs
	return w
}

// AppendDefaultRules adds the default set of rune bindings to a rule set.
//
// The default rules are:
//
// Whitespace at the start and end of input lines, along with the newline
// characters themselves will be stripped.
// ¶ Will be replaced by a newline.
// • Will be converted to a space.
// It will attempt to line up columns indicated by ║ in adjacent lines using a tabwriter.
// The indent level can be increased by » and decreased by «.
// § can be used to disable the reflow behaviours, and reenable them again.
// ø will flush the writers, reseting all column alignment behaviour.
func AppendDefaultRules(rules []Rule) []Rule {
	return append(rules, []Rule{
		{Rune: '\n', Action: func(w *Writer, r rune) error { return w.StrippedEOL() }},
		{Rune: ' ', Action: func(w *Writer, r rune) error { return w.Whitespace(r) }},
		{Rune: '\t', Action: func(w *Writer, r rune) error { return w.Whitespace(r) }},
		{Rune: Column, Action: func(w *Writer, r rune) error { return w.Column() }},
		{Rune: Toggle, Action: func(w *Writer, r rune) error { return w.DisableUntil(Toggle) }},
		{Rune: Disable, Action: func(w *Writer, r rune) error { return w.DisableUntil(Enable) }},
		{Rune: Indent, Action: func(w *Writer, r rune) error { return w.Increase() }},
		{Rune: Unindent, Action: func(w *Writer, r rune) error { return w.Decrease() }},
		{Rune: Space, Action: func(w *Writer, r rune) error { return w.WriteRune(' ') }},
		{Rune: EOL, Action: func(w *Writer, r rune) error { return w.EOL() }},
		{Rune: Flush, Action: func(w *Writer, r rune) error { return w.Flush() }},
	}...)
}

// StrippedEOL indicates that an end of line character was suppressed.
func (w *Writer) StrippedEOL() error {
	w.space.Reset()
	w.stripping = true
	return nil
}

// Whitespace indicates that the rune should be considered whitespace.
func (w *Writer) Whitespace(r rune) error {
	if w.stripping {
		return nil
	}
	_, err := w.space.WriteRune(r)
	return err
}

// Column inidcates that a column marker should be inserted for alignement.
func (w *Writer) Column() error {
	w.stripping = true
	_, err := w.space.WriteRune('\t')
	return err
}

// DisableUntil makes the reflow push all runes through verbatim until the next occurcence of r.
func (w *Writer) DisableUntil(r rune) error {
	w.Disabled = r
	w.beforeRune()
	return nil
}

// Increase the indent level of the reflow.
func (w *Writer) Increase() error {
	w.Depth++
	return nil
}

// Decrease the indent level of the reflow.
func (w *Writer) Decrease() error {
	w.Depth--
	return nil
}

// EOL makes the reflow add an end of line to the output.
func (w *Writer) EOL() error {
	w.newline = true
	w.stripping = true
	w.space.Reset()
	return w.WriteRune('\n')
}

// WriteRune writes the UTF-8 encoding of Unicode code point r, returning an error if it cannot.
func (w *Writer) WriteRune(r rune) error {
	n := utf8.EncodeRune(w.runeBuf[:], r)
	_, err := w.Out.Write(w.runeBuf[:n])
	return err
}

// Flush causes any cached bytes to be flushed to the underlying stream.
// This has the side effect of forcing a reset of column detection.
func (w *Writer) Flush() error {
	return w.Tabs.Flush()
}

// Reset reverts the stream back to the initial state, dropping column, indentation or any
// other state.
func (w *Writer) Reset() {
	w.Tabs.Flush()
	w.Depth = 0
	w.Disabled = 0
	w.space.Reset()
	w.lastDepth = 0
	w.newline = false
	w.stripping = false
}

// Write implements io.Writer with the reflow logic.
func (w *Writer) Write(data []byte) (n int, err error) {
	for _, r := range string(data) {
		if err := w.PushRune(r); err != nil {
			return len(data), err
		}
	}
	return len(data), nil
}

// PushRune pushes a rune through the reflow logic.
func (w *Writer) PushRune(r rune) error {
	if w.Disabled != 0 {
		// We are disabled, so just write runes until we stop being disabled
		if r == w.Disabled {
			// Turn back on, but still skip this rune
			w.Disabled = 0
			return nil
		}
		return w.WriteRune(r)
	}
	// See if this run activates any special rules
	for _, rule := range w.Rules {
		if rule.Rune == r {
			return rule.Action(w, r)
		}
	}
	w.beforeRune()
	// Write the rune out to the underlying writer
	return w.WriteRune(r)
}

func (w *Writer) beforeRune() {
	// We have a normal rune, do we need to start a new line?
	if w.newline {
		w.newline = false
		if w.Depth != w.lastDepth {
			// indentation is different to last real write, so flush the tabwriter
			w.Flush()
			w.lastDepth = w.Depth
		}
		if w.Depth > 0 {
			w.runeBuf[0] = tabwriter.Escape
			w.Out.Write(w.runeBuf[:1])
			for i := 0; i < w.Depth; i++ {
				io.WriteString(w.Out, w.Indent)
			}
			w.Out.Write(w.runeBuf[:1])
		}
	}
	// Do we have any pending whitespace to flush?
	w.stripping = false
	w.space.WriteTo(w.Out)
}
