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
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/text/note"
)

type (
	// key is an unexported type for context keys defined in this package.
	key string
)

// Key is the context key for handlers.
const Key = key("Handler")

func (key) Transcribe(context.Context, *note.Page, interface{}) {}

// Default is the handler to use when one has not been set on
// the context.
var Default = note.Sorter(Std(note.DefaultStyle))

// FromContext returns the current handler of the context.
func FromContext(ctx context.Context) note.Handler {
	h := ctx.Value(Key)
	if h != nil {
		return h.(note.Handler)
	}
	return Default
}

// NewContext returns a new context with the note handler added to it.
func NewContext(ctx context.Context, handler note.Handler) context.Context {
	return keys.WithValue(ctx, Key, handler)
}

// Send writes a note out to the handler stored in the context.
func Send(ctx context.Context, p note.Page) {
	FromContext(ctx)(p)
}
