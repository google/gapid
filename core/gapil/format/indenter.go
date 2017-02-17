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

package format

import (
	"bytes"
	"io"
)

type indenter struct {
	out     io.Writer
	indent  int
	midline bool
	prev    rune
}

func (w *indenter) Write(v []byte) (int, error) {
	b := &bytes.Buffer{}
	for _, r := range string(v) {
		switch r {
		case '»':
			w.indent++

		case '«':
			w.indent--

		case '•':
			if w.midline {
				switch w.prev {
				case '•', ' ', '\n':
				default:
					b.WriteRune(' ')
					w.prev = r
				}
			}

		case '\n':
			b.WriteRune(r)
			w.midline = false
			w.prev = r

		default:
			if !w.midline {
				for i := 0; i < w.indent; i++ {
					b.WriteString("  ")
				}
				w.midline = true
			}
			b.WriteRune(r)
			w.prev = r
		}
	}
	w.out.Write(b.Bytes())
	return len(v), nil
}
