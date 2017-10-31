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

package id

import (
	"context"
)

// Remapper is an interface which allows remapping between ID to int64.
// One such remapper can be stored in the current Context.
// This is used to handle resource when converting to/from proto.
// It needs to live here to break go package dependency cycles.
type Remapper interface {
	RemapIndex(ctx context.Context, index int64) (ID, error)
	RemapID(ctx context.Context, id ID) (int64, error)
}

type remapperKeyTy string

const remapperKey = remapperKeyTy("remapper")

// GetRemapper returns the Remapper attached to the given context.
func GetRemapper(ctx context.Context) Remapper {
	if val := ctx.Value(remapperKey); val != nil {
		return val.(Remapper)
	}
	panic("remapper missing from context")
}

// PutRemapper amends a Context by attaching a Remapper reference to it.
func PutRemapper(ctx context.Context, d Remapper) context.Context {
	if val := ctx.Value(remapperKey); val != nil {
		panic("Context already holds remapper")
	}
	return context.WithValue(ctx, remapperKey, d)
}
