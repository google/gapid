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

package database

import (
	"context"

	"github.com/google/gapid/core/data/id"
)

var resolvablePrefix = []byte("resolvable:")

// Resolvable is the interface for types that redirects database resolves to an
// object lazily built using Resolve(). The Resolve() method will be called
// the first time the object is resolved, and all subsequent resolves will
// return the same pre-built object.
// Resolvable is commonly implemented by objects that generate data that is
// expensive to calculate but can be deterministically produced using the
// information stored in the Resolvable.
type Resolvable interface {
	// Resolve constructs and returns the lazily-built object.
	Resolve(ctx context.Context) (interface{}, error)
}

// resolvedID returns the identifier of a resolved object given the identifier
// of the Resolvable.
func resolvedID(in id.ID) id.ID {
	return id.OfBytes(resolvablePrefix, in[:])
}
