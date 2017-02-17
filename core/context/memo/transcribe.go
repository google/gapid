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

package memo

import (
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/text/note"
)

type (
	// Filter is the interface for keys that want to trigger page filtering
	// during transcription.
	Filter interface {
		// Filter returns true the page should be dropped.
		Filter(ctx context.Context) bool
	}

	// Transcriber is the interface for keys that want to do transcription.
	// Transcribers have full access to the context, but in general should only be
	// adding their own value to the page.
	Transcriber interface {
		// Transcribe is called to copy the value associated with this key from the
		// context to the page.
		Transcribe(ctx context.Context, page *note.Page, value interface{})
	}
)

// IsFiltered returns whether, for the supplied set of keys, pages formed
// from the value source should be filtered or not.
func IsFiltered(ctx context.Context, keys []interface{}) bool {
	for _, k := range keys {
		if f, ok := k.(Filter); ok {
			if f.Filter(ctx) {
				return true
			}
		}
	}
	return false
}

// Keys creates a page from a set of keys and a value source.
func Keys(ctx context.Context, keys []interface{}) note.Page {
	page := note.Page{}
	for _, k := range keys {
		value := ctx.Value(k)
		if value == nil {
			continue
		}
		switch k := k.(type) {
		case Transcriber:
			// The key controlls the transcription.
			k.Transcribe(ctx, &page, value)
		default:
			// Add the key value pair as an extra item
			page.Append(ExtraSection, k, value)
		}
	}
	return page
}

// Transcribe creates a page from a context.
// It does not test or use the filter conditions.
func Transcribe(ctx context.Context) note.Page {
	keys := keys.Get(ctx)
	return Keys(ctx, keys)
}

// From conditionally transcribes a page from a context.
// If the page should be filtered, it will be empty and the second return value will be false.
// It is possible for the page to be empty even if the second value is true.
func From(ctx context.Context) (note.Page, bool) {
	keys := keys.Get(ctx)
	if IsFiltered(ctx, keys) {
		return nil, false
	}
	return Keys(ctx, keys), true
}
