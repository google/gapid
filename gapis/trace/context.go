// Copyright (C) 2018 Google Inc.
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

package trace

import (
	"context"

	"github.com/google/gapid/core/context/keys"
)

type contextKey string

const traceMgrKey = contextKey("traceMgrID")

// PutManager attaches a manager to a Context.
func PutManager(ctx context.Context, m *Manager) context.Context {
	return keys.WithValue(ctx, traceMgrKey, m)
}

// GetManager retrieves the manager from a context previously annotated by
// PutManager.
func GetManager(ctx context.Context) *Manager {
	val := ctx.Value(traceMgrKey)
	if val == nil {
		panic(string(traceMgrKey + " not present"))
	}
	return val.(*Manager)
}
