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

	"github.com/google/gapid/core/log"
)

// Library is the main interface to record storage.
// It manages a set of shelves, and uses them to open and create ledgers.
type Library interface {
	// Add appends new shelves to the library.
	Add(ctx context.Context, shelf ...Shelf)
	// Open opens a ledger from a shelf in the library.
	Open(ctx context.Context, name string, null interface{}) (Ledger, error)
}

// library is the default implementation of Library, it just holds a list of shelves.
type library struct {
	shelves []Shelf
}

// NewLibrary returns a new record library.
func NewLibrary(ctx context.Context) Library {
	return &library{}
}

// Add implements Shelf.Add for the default library implementation.
func (l *library) Add(ctx context.Context, shelf ...Shelf) {
	l.shelves = append(l.shelves, shelf...)
}

// Open implements Library.Open for the default library implementation.
// All shelves are searched for a matching ledger, if none is found then a new ledger is
// opened in the first shelf that was added.
func (l *library) Open(ctx context.Context, name string, null interface{}) (Ledger, error) {
	if len(l.shelves) == 0 {
		return nil, log.Err(ctx, nil, "Cannot open ledger with no shelves")
	}
	for _, shelf := range l.shelves {
		if ledger, err := shelf.Open(ctx, name, null); err != nil {
			return nil, err
		} else if ledger != nil {
			return ledger, nil
		}
	}
	return l.shelves[0].Create(ctx, name, null)
}
