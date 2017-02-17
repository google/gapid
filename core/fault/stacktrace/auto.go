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

package stacktrace

import (
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/text/note"
)

type (
	// Controls are the settings that affect how a stack trace is taken when notes
	// are transcribed.
	Controls struct {
		// Condition controls whether a stack trace is take or not.
		Condition func(context.Context) bool
		// Source is the function used to capture the stacktrace.
		Source Source
	}

	keyType string
)

// Key is the key for a value that captures the stack trace into the note.
const Key = keyType("Stacktrace")

func (keyType) OrderBy() string    { return "__" }
func (keyType) KeyToValue() string { return "⇒" }

// CaptureOn adds a key to the context that causes stacktraces to be collected if the
// cond returns true.
func CaptureOn(ctx context.Context, controls Controls) context.Context {
	return keys.WithValue(ctx, Key, controls)
}

// OnError is a simple function that can be used as a cond to CaptureOn.
// It returns true if the level of the context is at least severity.Error.
func OnError(ctx context.Context) bool {
	return severity.FromContext(ctx) <= severity.Error
}

// Transcribe triggers a stacktrace capture and stores the results on the note.
func (k keyType) Transcribe(ctx context.Context, p *note.Page, value interface{}) {
	controls := ctx.Value(k).(Controls)
	if controls.Condition != nil && !controls.Condition(ctx) {
		return
	}
	callers := controls.Source()
	if len(callers) <= 0 {
		return
	}
	trace := note.Page{}
	trace.Append(memo.TextSection, nil, callers[0].Location)
	for _, caller := range callers[1:] {
		trace.Append(memo.ExtraSection, keyType(caller.Function.Name), caller.Location)
	}
	p.Append(memo.DetailSection, keyType(callers[0].Function.Name), trace)
}
