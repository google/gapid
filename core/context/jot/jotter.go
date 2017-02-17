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

package jot

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/output"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/text/note"
)

// Jotter is a helper that makes note taking more fluent.
// Unlike a context a jotter is a mutable object, it is expected that you will build and use one
// in a single line.
// If you need to emit from a jotter multiple times you should either use the context functions, or clone
// the jotter each time.
type Jotter struct {
	// Context is the context this jotter was built from.
	Context context.Context
	// Page is the note page this jotter is generating.
	Page note.Page
}

// Clone creates a copy of the jotter.
func (j Jotter) Clone() Jotter {
	j.Page = j.Page.Clone()
	return j
}

// With adds a value to the detail section.
func (j Jotter) With(key interface{}, value interface{}) Jotter {
	if j.Context != nil {
		j.Page.Append(memo.DetailSection, key, value)
	}
	return j
}

// Tag sets the tag on a note.
func (j Jotter) Tag(msg string) Jotter {
	if j.Context != nil {
		memo.TagKey.Transcribe(j.Context, &j.Page, msg)
	}
	return j
}

// Jot adds text to the note.
func (j Jotter) Jot(msg string) Jotter {
	if j.Context != nil && msg != "" {
		memo.PrintKey.Transcribe(j.Context, &j.Page, msg)
	}
	return j
}

// Jotf adds a formatted message to the jotter.
func (j Jotter) Jotf(msg string, args ...interface{}) Jotter {
	if j.Context != nil {
		memo.PrintKey.Transcribe(j.Context, &j.Page, fmt.Sprintf(msg, args...))
	}
	return j
}

// Cause sets the cause onto the note.
func (j Jotter) Cause(err error) Jotter {
	if j.Context != nil {
		j.Page = cause.New(j.Page, err).Page
	}
	return j
}

// Send writes the page out to the default handler.
func (j Jotter) Send() {
	if j.Context != nil {
		output.Send(j.Context, j.Page)
	}
}

// Print is j.Jot(msg).Send()
// It is used to immediately log a message.
func (j Jotter) Print(msg string) {
	if j.Context != nil {
		j.Jot(msg).Send()
	}
}

// Printf is j.Jotf(msg, args...).Send()
// It is used to immediately log a formatted message.
func (j Jotter) Printf(msg string, args ...interface{}) {
	if j.Context != nil {
		j.Jotf(msg, args...).Send()
	}
}
