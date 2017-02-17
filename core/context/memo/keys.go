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
	"time"

	"github.com/google/gapid/core/text/note"
)

type (
	// TimeValue is for the timestamp value if enabled.
	TimeValue string
	// TagValue is for keys that want to transcribe their value to the tag section.
	TagValue string
	// PrefixValue is for keys that want to transcribe their value to the prefix section.
	PrefixValue string
	// PrefixPair is for keys that want to transcribe their key and value to the prefix section.
	PrefixPair string
	// TextValue is for keys that want to transcribe their value to the text section.
	TextValue string
	// SuffixValue is for keys that want to transcribe their value to the suffix section.
	SuffixValue string
	// SuffixPair is for keys that want to transcribe their key and value to the suffix section.
	SuffixPair string
	// DetailPair is for keys that want to transcribe their key and value to the detail section.
	DetailPair string
)

const (
	// TagKey is the key for the default tag entry.
	TimeKey = TimeValue("Time")
	// TagKey is the key for the default tag entry.
	TagKey = TagValue("Tag")
	// TraceKey is the key for a call chain trace.
	TraceKey = PrefixValue("Trace")
	// PrintKey is the key for simple message entries.
	PrintKey = TextValue("Text")
	// StatusKey is the key for the default message status.
	StatusKey = SuffixValue("Status")
)

func (k TimeValue) OmitKey() bool { return true }
func (k TimeValue) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	switch value := value.(type) {
	case func() time.Time:
		page.Append(TimeSection, k, value())
	default:
		page.Append(TimeSection, k, value)
	}
}

func (k TagValue) OmitKey() bool { return true }
func (k TagValue) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(TagSection, k, value)
}

func (k PrefixValue) OmitKey() bool { return true }
func (k PrefixValue) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(PrefixSection, k, value)
}

func (k PrefixPair) OmitKey() bool { return false }
func (k PrefixPair) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(PrefixSection, k, value)
}

func (k TextValue) OmitKey() bool { return true }
func (k TextValue) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(TextSection, k, value)
}

func (k SuffixValue) OmitKey() bool { return true }
func (k SuffixValue) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(SuffixSection, k, value)
}

func (k SuffixPair) OmitKey() bool { return false }
func (k SuffixPair) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(SuffixSection, k, value)
}

func (k DetailPair) OmitKey() bool { return false }
func (k DetailPair) Transcribe(ctx context.Context, page *note.Page, value interface{}) {
	page.Append(DetailSection, k, value)
}
