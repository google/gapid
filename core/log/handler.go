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

package log

import (
	"io"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/text/note"
)

// GetHandler gets the active Handler for this context.
func GetHandler(ctx Context) note.Handler {
	return output.FromContext(ctx.Unwrap())
}

// Handler returns a new Builder with the given Handler set on it.
func (ctx logContext) Handler(h note.Handler) Context {
	return Wrap(output.NewContext(ctx.Unwrap(), note.Sorter(h)))
}

// Stdout returns a Handler that writes to os.Stdout.
// This is conceptually the same as Writer(style, os.Stdout) except the writer is not bound, so if os.Stdout changes
// the new value will be picked up.
func Stdout(style note.Style) note.Handler {
	return output.Stdout(style)
}

// Stderr returns a Handler that writes to os.Stderr.
// This is conceptually the same as Writer(style, os.Stderr) except the writer is not bound, so if os.Stderr changes
// the new value will be picked up.
func Stderr(style note.Style) note.Handler {
	return output.Stderr(style)
}

// Std returns a Handler that splits writes between to Stderr and Stdout.
func Std(style note.Style) note.Handler {
	return output.Std(style)
}

// Writer is a Handler that uses the active Style to write records to an underlying io.Writer
func Writer(style note.Style, to io.Writer) note.Handler {
	return style.Scribe(to)
}

// Channel is a Handler that passes log records to another Handler through a blocking chan.
// This makes this Handler safe to use from multiple threads
func Channel(to note.Handler) (note.Handler, func()) {
	return note.Channel(to, 0)
}

// Fork forwards all messages to all supplied handlers.
func Fork(handlers ...note.Handler) note.Handler {
	return note.Broadcast(handlers...)
}
