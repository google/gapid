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
	"fmt"
	"time"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/text/note"
)

var (
	// TimeSection is a verbose only section for the note timestamp.
	TimeSection = note.SectionInfo{Key: "Time", Order: -1, Relevance: note.Relevant}
	// TagSection is a critical filter tag section that occurs before the main message.
	TagSection = note.SectionInfo{Key: "Tag", Order: 10, Relevance: note.Critical}
	// PrefixSection is a relevant section that occurs before the main message.
	PrefixSection = note.SectionInfo{Key: "Prefix", Order: 20, Relevance: note.Relevant}
	// TextSection is the critical text message section.
	TextSection = note.SectionInfo{Key: "Text", Order: 30, Relevance: note.Critical}
	// SuffixSection is a relevant section that occurs after the main message.
	SuffixSection = note.SectionInfo{Key: "Suffix", Order: 40, Relevance: note.Relevant}
	// DetailSection contains relevant multiline information at the end of the message.
	DetailSection = note.SectionInfo{Key: "Detail", Order: 50, Relevance: note.Relevant, Multiline: true}
	// ExtraSection contains values that are not normally printed.
	ExtraSection = note.SectionInfo{Key: "Extra", Order: 60, Relevance: note.Irrelevant, Multiline: true}
)

// Timestamp is used to enable transcription timestamping.
// The supplied function is invoked on transcription, and it's time stored on the note.
func Timestamp(ctx context.Context, timestamp func() time.Time) context.Context {
	return keys.WithValue(ctx, TimeKey, timestamp)
}

// Tag is used to set the tag message on the context.
func Tag(ctx context.Context, msg interface{}) context.Context {
	return keys.WithValue(ctx, TagKey, msg)
}

// Tagf is used to set a formatted tag message on the context.
func Tagf(ctx context.Context, msg string, args ...interface{}) context.Context {
	return keys.WithValue(ctx, TagKey, fmt.Sprintf(msg, args...))
}

// Enter returns a context with an entry added to the trace chain.
func Enter(ctx context.Context, name string) context.Context {
	return keys.Chain(ctx, TraceKey, name)
}

// Print is used to add a message to the context.
func Print(ctx context.Context, msg interface{}) context.Context {
	return keys.WithValue(ctx, PrintKey, msg)
}

// Printf is used to add a formatted message to the context.
func Printf(ctx context.Context, msg string, args ...interface{}) context.Context {
	return keys.WithValue(ctx, PrintKey, fmt.Sprintf(msg, args...))
}

// Status is used to set the tag message on the context.
func Status(ctx context.Context, msg interface{}) context.Context {
	return keys.WithValue(ctx, StatusKey, msg)
}

// Statusf is used to set a formatted tag message on the context.
func Statusf(ctx context.Context, msg string, args ...interface{}) context.Context {
	return keys.WithValue(ctx, StatusKey, fmt.Sprintf(msg, args...))
}
