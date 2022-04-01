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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/shell"
	"github.com/google/gapid/core/text/reflow"
	"golang.org/x/tools/imports"
)

const (
	disable    = '⋖'
	enable     = '⋗'
	indent     = '»'
	unindent   = '«'
	suppress   = '§'
	newline    = '¶'
	whitespace = '•'
)

// NewReflow reformats the value using the reflow package.
func (f *Functions) NewReflow(indent string, value string) (string, error) {
	log.D(f.ctx, "Reflowing string")
	buf := &bytes.Buffer{}
	flow := reflow.New(buf)
	flow.Indent = indent
	io.WriteString(flow, value)
	return buf.String(), nil
}

// Reflow does the primitive reflow, but no language specific handling.
func (f *Functions) Reflow(indentSize int, value string) (string, error) {
	log.D(f.ctx, "Reflowing string")
	result, err := legacyReflow(value, indentSize)
	if err != nil {
		return "", fmt.Errorf("%s : %s", f.active.Name(), err)
	}
	return string(result) + "\n", nil
}

const goIndent = 2 // Required by go style guide

// GoFmt reflows the string as if it were go code using the standard go fmt library.
func (f *Functions) GoFmt(value string) (string, error) {
	log.D(f.ctx, "Reflowing go code")
	result, err := legacyReflow(value, goIndent)
	if err != nil {
		return "", fmt.Errorf("%s : %s", f.active.Name(), err)
	}
	opt := &imports.Options{
		TabWidth:   goIndent,
		TabIndent:  true,
		Comments:   true,
		Fragment:   true,
		FormatOnly: true,
	}
	formatted, err := imports.Process(f.active.Name(), result, opt)
	if err != nil {
		log.D(f.ctx, "GoFmt errored: %v", err)
		return string(result), nil
	}
	return string(formatted), nil
}

// Format reflows the string using an external command.
func (f *Functions) Format(command stringList, value string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("%s : Invalid Format command", f.active.Name())
	}
	binary := command[0]
	log.D(f.ctx, "Reflowing code. Binary: %v", binary)
	// indent level is arbitrary, because we expect the external formatter to redo it anyway
	result, err := legacyReflow(value, 4)
	if err != nil {
		return "", fmt.Errorf("%s : %s", f.active.Name(), err)
	}
	stdin := bytes.NewBuffer(result)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = shell.Command(binary, command[1:]...).Capture(stdout, stderr).Read(stdin).Run(f.ctx)
	if err != nil {
		log.D(f.ctx, "Reformat errored: %v", stderr.String())
		return string(result), nil
	}
	return stdout.String(), nil
}

func panicWrite(buf *bytes.Buffer, r rune) {
	_, err := buf.WriteRune(r)
	if err != nil {
		panic(err)
	}
}

func legacyReflow(in string, indentSize int) ([]byte, error) {
	depth := 0
	wasNewline := false
	suppressing := true
	join := false
	enabled := true
	buf := &bytes.Buffer{}
	flushPending := func() {
		if wasNewline && !suppressing {
			// write the indent
			panicWrite(buf, '\n')
			for i := 0; i < depth*indentSize; i++ {
				panicWrite(buf, ' ')
			}
		}
		suppressing = false
		wasNewline = false
		join = false
	}
	for _, ch := range in {
		if !enabled {
			if ch == enable {
				enabled = true
			} else {
				panicWrite(buf, ch)
			}
		} else {
			switch ch {
			case disable:
				flushPending()
				enabled = false
				ch = 0
			case whitespace:
				ch = ' '
			case suppress:
				suppressing = true
				ch = 0
			case newline:
				panicWrite(buf, '\n')
				fallthrough
			case '\n', '\r':
				if !join {
					wasNewline = true
				}
				ch = 0
			case '\t', ' ':
				if wasNewline {
					ch = 0
				}
			case indent:
				ch = 0
				depth += 1
			case '{', '[':
				flushPending()
				depth += 1
			case unindent:
				ch = 0
				fallthrough
			case '}', ']':
				depth -= 1
			}

			if ch != 0 {
				flushPending()
				panicWrite(buf, ch)
			}
		}
	}
	return buf.Bytes(), nil
}
