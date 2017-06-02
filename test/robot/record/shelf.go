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

package record

import (
	"context"
	"net/url"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
)

// Shelf is the interface to an object that maintains a set of ledgers stored in a
// consistent way.
type Shelf interface {
	// Open is used to open a ledger by name.
	// All records in the ledger must be of the same type as the null value.
	Open(ctx context.Context, name string, null interface{}) (Ledger, error)
	// Create is used to make and return a new ledger in the shelf.
	// All records in the ledger must be of the same type as the null value.
	Create(ctx context.Context, name string, null interface{}) (Ledger, error)
}

// NewShelf returns a new record shelf from the supplied url.
// The type of shelf will depend on the url given.
func NewShelf(ctx context.Context, shelfURL string) (Shelf, error) {
	ctx = log.V{"ShelfURL": shelfURL}.Bind(ctx)
	location, err := url.Parse(shelfURL)
	if err != nil {
		return nil, log.Err(ctx, err, "Invalid record shelf location")
	}
	switch location.Scheme {
	case "", "file":
		if location.Host != "" {
			return nil, log.Err(ctx, nil, "Host not supported for file shelves")
		}
		if location.Path == "" {
			return nil, log.Err(ctx, nil, "Path must be specified for file shelves")
		}
		log.I(ctx, "Build a file record shelf on %s", location.Path)
		return NewFileShelf(ctx, file.Abs(location.Path))
	case "memory":
		log.I(ctx, "Start an in memory record shelf")
		return NewNullShelf(ctx)
	default:
		return nil, log.Err(ctx, nil, "Unknown record shelf url type")
	}
}
