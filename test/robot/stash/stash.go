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

package stash

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/test/robot/search"
)

const (
	Unknown   = Status_Unknown
	Uploading = Status_Uploading
	Present   = Status_Present

	ErrEntityNotFound = fault.Const("Entity not found")
)

type (
	// EntityHandler is a fuction that can be invoked for each entry in a
	// stream of entities.
	EntityHandler func(context.Context, *Entity) error

	// Service is the interface to a stash storage implementation.
	// It abstracts away the actual storage part from the service logic.
	Service interface {
		// Close can be called to shut down the store.
		// Behaviour of any other method after close is called is undefined.
		Close()
		// Lookup returns information about a single entity by id from the store.
		Lookup(ctx context.Context, id string) (*Entity, error)
		// Search returns a channel of matching entities from the store.
		// The handler will be invoked once per matching entry.
		// If the handler returns an error, the search will be terminated.
		Search(ctx context.Context, query *search.Query, handler EntityHandler) error
		// Open returns a reader to an entity's data.
		// The reader may be nil if the entity is not present.
		Open(ctx context.Context, id string) (io.ReadSeeker, error)
		// Read returns a complete entity from the store.
		// It may return an empty byte array if the entity is not present.
		Read(ctx context.Context, id string) ([]byte, error)
		// Create is used to add a new entity to the store.
		// It returns a writer that can be used to write the content of the entity.
		Create(ctx context.Context, info *Upload) (io.WriteCloser, error)
	}
)

// Format provices pretty printing of stash entities.
// It conforms to the fmt.Formatter interface.
func (e *Entity) Format(f fmt.State, c rune) {
	name := "?"
	if len(e.Upload.Name) > 0 {
		name = e.Upload.Name[0]
	}
	t, _ := ptypes.Timestamp(e.Timestamp)
	fmt.Fprintf(f, "%s %s : %s", name, t.Format(time.Stamp), e.Upload.Id)
}
