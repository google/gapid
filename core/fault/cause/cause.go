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

package cause

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/context/memo"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text/note"
)

type (
	// Error is the type use to represent structured errors.
	// It conforms to the error interface.
	Error struct {
		note.Page
		cause error
	}
)

var (
	// CauseSection contains detail information at the end of the message.
	CauseSection = note.SectionInfo{Key: "Cause", Order: 45, Relevance: note.Important, Multiline: true}
)

// Error is implemented to conform to error.
func (e Error) Error() string { return note.Brief.Print(e.Page) }

// Cause reports the underlying cause of the error wrapper.
func (e Error) Cause() error { return e.cause }

// With adds a new value to the structured error.
func (e Error) With(key interface{}, value interface{}) Error {
	e.Page.Append(memo.DetailSection, key, value)
	return e
}

// New returns an wrapped error from the supplied page and error.
func New(page note.Page, err error) Error {
	result := Error{
		Page:  page,
		cause: err,
	}
	if inner, isWrapped := err.(Error); isWrapped {
		// TODO: Would be nice to leave shared values only on one of the nested pages
		result.Page.Append(CauseSection, nil, inner.Page)
	} else {
		result.Page.Append(CauseSection, nil, err)
	}
	return result
}

// Wrap returns an Error from the supplied context and cause.
func Wrap(ctx context.Context, err error) Error {
	return New(memo.Transcribe(ctx), err)
}

// Explain returns an Error from the supplied context and cause with the given
// explanation message.
func Explain(ctx context.Context, err error, msg string) Error {
	if err == nil {
		return Wrap(ctx, fault.Const(msg))
	}
	result := Wrap(ctx, err)
	memo.PrintKey.Transcribe(ctx, &result.Page, msg)
	return result
}

// Explainf returns an error Note from the supplied context and cause, using Sprintf
// to form the explanation message from msg and args.
func Explainf(ctx context.Context, err error, msg string, args ...interface{}) Error {
	return Explain(ctx, err, fmt.Sprintf(msg, args...))
}
