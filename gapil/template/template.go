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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime/debug"
	"text/template"
	"unicode/utf8"

	"github.com/google/gapid/core/log"
)

const maxErrors = 10

// Note isTrimSpace is only testing the Latin1 spaces.
func isTrimSpace(b byte) bool {
	switch b {
	case ' ', '\n', '\r':
		return true
	}
	return false
}

// trimWriter, writes to the underlying io.Writer, but with leading
// and trailing spaces trimmed from the output. Only current trailing
// spaces are saved between calls to Write().
type trimWriter struct {
	out     io.Writer
	atStart bool   // at the start we throw away leading spaces.
	spaces  []byte // spaces which are currently trailing.
}

func newTrimWriter(out io.Writer) io.Writer {
	return &trimWriter{out: out, atStart: true}
}

func (t *trimWriter) Write(buf []byte) (int, error) {
	l := len(buf)
	if l == 0 {
		return 0, nil
	}

	begin := 0 // index of the first byte to output
	if t.atStart {
		// Skip over leading spaces
		// Find the start of the interesting content.
		for ; begin < len(buf); begin++ {
			b := buf[begin]
			// If the character is in Latin1, it is safe to treat it is a byte
			if b >= utf8.RuneSelf || !isTrimSpace(b) {
				t.atStart = false
				break
			}
		}

		if t.atStart {
			// The whole buffer is leading spaces
			return l, nil
		}
	}

	// Find the end of the interesting content (remove trailing spaces).
	end := len(buf) // index one beyond the end of the interesting content
	for ; end > begin; end-- {
		b := buf[end-1]
		// If the character is in Latin1, it is safe to treat it is a byte
		if b >= utf8.RuneSelf || !isTrimSpace(b) {
			break
		}
	}

	if begin == end {
		// The whole buffer is trailing spaces
		t.spaces = append(t.spaces, buf...)
		return l, nil
	}

	// The buffer has some content to output.
	// First output any trailing spaces from the previous call
	if len(t.spaces) != 0 {
		if ws, err := t.out.Write(t.spaces); err != nil || ws != len(t.spaces) {
			return ws, err
		}
		// We are done with previous trailing spaces
		t.spaces = nil
	}

	// Output the content.
	if ws, err := t.out.Write(buf[begin:end]); err != nil || ws != end-begin {
		return ws, err
	}

	if end != len(buf) {
		// Save any trailing spaces
		t.spaces = append(t.spaces, buf[end:]...)
	}

	return l, nil
}

func (f *Functions) execute(active *template.Template, writer io.Writer, data interface{}) (err error) {
	olda := f.active
	oldw := f.writer
	f.active = active
	if writer != nil {
		f.writer = writer
	}
	f.writer = newTrimWriter(f.writer)
	defer func() {
		if r := recover(); r != nil {
			// There doesn't appear to be a clean way to get both the panic stack
			// and the template stack. This is the closest I can figure.
			err = fmt.Errorf("panic executing template %v: %v %v", f.active.Name(), r, string(debug.Stack()))
		}

		f.active = olda
		f.writer = oldw
	}()
	return f.active.Execute(f.writer, data)
}

// Include loads each of the templates and executes their main bodies.
// The filenames are relative to the template doing the include.
func (f *Functions) Include(templates ...string) error {
	dir := ""
	if f.active != nil {
		dir = filepath.Dir(f.active.Name())
	}
	for _, t := range templates {
		if dir != "" {
			t = filepath.Join(dir, t)
		}
		if f.templates.Lookup(t) == nil {
			log.D(f.ctx, "Reading %v", t)
			tmplData, err := f.loader(t)
			if err != nil {
				log.E(f.ctx, "Error loading %s: %s", t, err)
				return fmt.Errorf("%s: %s\n", t, err)
			}
			tmpl, err := f.templates.New(t).Parse(string(tmplData))
			if err != nil {
				log.E(f.ctx, "Error parsing %s: %s", t, err)
				return fmt.Errorf("%s: %s\n", t, err)
			}
			log.D(f.ctx, "Executing %v", tmpl.Name())
			var buf bytes.Buffer
			if err = f.execute(tmpl, &buf, f.api); err != nil {
				log.E(f.ctx, "Error executing %s: %s", tmpl.Name(), err)
				return fmt.Errorf("%s: %s\n", tmpl.Name(), err)
			}
		}
	}
	return nil
}

// Write takes a string and writes it into the specified file.
// The filename is relative to the output directory.
func (f *Functions) Write(fileName string, value string) (string, error) {
	outputPath := filepath.Join(f.basePath, fileName)
	log.D(f.ctx, "Writing output to %v", outputPath)

	return "", ioutil.WriteFile(outputPath, []byte(value), 0666)
}
