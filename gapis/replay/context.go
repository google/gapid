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

package replay

import (
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/note"
)

type contextMgrKeyTy string

const contextMgrKey = contextMgrKeyTy("replayMgrID")

func (contextMgrKeyTy) Transcribe(context.Context, *note.Page, interface{}) {}

// PutManager attaches a manager to a Context.
func PutManager(ctx log.Context, m *Manager) log.Context {
	return log.Wrap(keys.WithValue(ctx.Unwrap(), contextMgrKey, m))
}

// GetManager retrieves the manager from a context previously annotated by
// PutManager.
func GetManager(ctx log.Context) *Manager {
	val := ctx.Value(contextMgrKey)
	if val == nil {
		panic(string(contextMgrKey + " not present"))
	}
	return val.(*Manager)
}
