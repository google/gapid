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

package output

import (
	"fmt"
	"os"

	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

// Stdout returns a Handler that writes to os.Stdout.
// This is conceptually the same as ToWriter(style, os.Stdout) except the writer is not bound, so if os.Stdout changes
// the new value will be picked up.
func Stdout(style note.Style) note.Handler {
	return func(page note.Page) error {
		err := style.Scribe(os.Stdout)(page)
		fmt.Fprintln(os.Stdout)
		return err
	}
}

// Stderr returns a Handler that writes to os.Stderr.
// This is conceptually the same as ToWriter(style, os.Stderr) except the writer is not bound, so if os.Stderr changes
// the new value will be picked up.
func Stderr(style note.Style) note.Handler {
	return func(page note.Page) error {
		err := style.Scribe(os.Stderr)(page)
		fmt.Fprintln(os.Stderr)
		return err
	}
}

// Std builds a handler that uses style to write messages to either StdOut or StdErr
// depending on the severity.
func Std(style note.Style) note.Handler {
	return func(page note.Page) error {
		level := severity.FindLevel(page)
		w := os.Stdout
		if level <= severity.Error {
			w = os.Stderr
		}
		err := style.Scribe(w)(page)
		fmt.Fprintln(w)
		return err
	}
}
