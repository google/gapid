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

package local

import (
	"reflect"
	"sync"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/data/search"
	"github.com/google/gapid/core/data/search/eval"
	"github.com/google/gapid/core/data/stash"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
)

var entityClass = reflect.TypeOf(&stash.Entity{})

type entityIndex struct {
	mu       sync.Mutex
	entities []*stash.Entity
	byID     map[string]*stash.Entity
	onAdd    event.Broadcast
}

func (i *entityIndex) init() {
	i.entities = []*stash.Entity{}
	i.byID = map[string]*stash.Entity{}
}

func (i *entityIndex) lockedAddEntry(ctx log.Context, entity *stash.Entity) {
	i.entities = append(i.entities, entity)
	i.byID[entity.Upload.Id] = entity
	if err := i.onAdd.Send(ctx, entity); err != nil {
		jot.Fail(ctx, err, "Stash notification failed")
	}
}

func (e *entityIndex) Lookup(ctx log.Context, id string) (*stash.Entity, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	entity, found := e.byID[id]
	if !found {
		return nil, stash.ErrEntityNotFound
	}
	return entity, nil
}

func (e *entityIndex) Search(ctx log.Context, query *search.Query, handler stash.EntityHandler) error {
	filter := eval.Filter(ctx, query, entityClass, event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, e.entities)
	if query.Monitor {
		return event.Monitor(ctx, &e.mu, e.onAdd.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}
