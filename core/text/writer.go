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

import "io"

// Writer returns a io writer that collects lines out of a stream and hands
// them to the supplied function.
func Writer(to func(string) error) io.WriteCloser {
	return &writer{to: to}
}

type writer struct {
	to      func(string) error
	remains string
}

// Write splits the input into lines, gathering across multiple calls if needed.
// Write implements the io.Writer interface
func (w *writer) Write(p []byte) (n int, err error) {
	s := string(p)
	start := 0
	for i, c := range s {
		if c == '\n' {
			fragment := s[start:i]
			start = i + 1
			if len(w.remains) > 0 {
				fragment = w.remains + fragment
				w.remains = ""
			}
			if err := w.to(fragment); err != nil {
				return 0, err
			}
		}
	}
	fragment := s[start:]
	if len(w.remains) > 0 {
		fragment = w.remains + fragment
	}
	w.remains = fragment
	return len(p), nil
}

// Close flushes any partial lines before closing the stream down.
// The behaviour on Subsequent calles to Write is undefined.
func (w writer) Close() error {
	var err error
	if w.remains != "" {
		err = w.to(w.remains)
		w.remains = ""
	}
	return err
}
