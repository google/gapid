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

package output_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

func TestStdout(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = output.NewContext(ctx, output.Stdout(note.Normal))
	for _, test := range tests {
		stdout := trap(&os.Stdout)
		testHandler(ctx, test)
		assert.For(test.severity.String()).That(stdout.read()).Equals(test.expect + "\n")
	}
}

func TestStderr(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = output.NewContext(ctx, output.Stderr(note.Normal))
	for _, test := range tests {
		stderr := trap(&os.Stderr)
		testHandler(ctx, test)
		assert.For(test.severity.String()).That(stderr.read()).Equals(test.expect + "\n")
	}
}

func TestStd(t *testing.T) {
	assert := assert.To(t)
	ctx := context.Background()
	ctx = output.NewContext(ctx, output.Std(note.Normal))
	for _, test := range tests {
		stdout := trap(&os.Stdout)
		stderr := trap(&os.Stderr)
		testHandler(ctx, test)
		var expectErr, expectOut string
		if test.severity <= severity.Error {
			expectErr = test.expect + "\n"
		} else {
			expectOut = test.expect + "\n"
		}
		assert.For("stderr->%s", test.severity).That(stderr.read()).Equals(expectErr)
		assert.For("stdout->%s", test.severity).That(stdout.read()).Equals(expectOut)
	}
}

type trapped struct {
	file **os.File
	old  *os.File
	r    *os.File
	w    *os.File
}

// Trap replaces file with a pipe, but does nothing to make writes to that pipe non blocking
func trap(file **os.File) trapped {
	t := trapped{file: file, old: *file}
	t.r, t.w, _ = os.Pipe()
	*file = t.w
	return t
}

// read collects closes the pipe, and then collects everything written to it.
// there is an implicit assumption that the os will buffer inside the pipe everything we wrote to it
// if the os buffer is not big enough, the write will block
func (t trapped) read() string {
	t.w.Close()
	buf := &bytes.Buffer{}
	io.Copy(buf, t.r)
	*t.file = t.old
	return buf.String()
}
