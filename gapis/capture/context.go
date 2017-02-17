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

package capture

import (
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/note"
	"github.com/google/gapid/gapis/service/path"
)

type contextKeyTy string

const contextKey = contextKeyTy("captureID")

func (contextKeyTy) Transcribe(context.Context, *note.Page, interface{}) {}

// Put attaches a capture path to a Context.
func Put(ctx log.Context, c *path.Capture) log.Context {
	return log.Wrap(keys.WithValue(ctx.Unwrap(), contextKey, c))
}

// Get retrieves the capture path from a context previously annotated by Put.
func Get(ctx log.Context) *path.Capture {
	val := ctx.Value(contextKey)
	if val == nil {
		panic(contextKey + " not present")
	}
	return val.(*path.Capture)
}

// Resolve resolves the capture from a context previously annotated by Put.
func Resolve(ctx log.Context) (*Capture, error) {
	return ResolveFromPath(ctx, Get(ctx))
}
