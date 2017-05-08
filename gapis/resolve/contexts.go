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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Contexts resolves the list of contexts belonging to a capture.
func Contexts(ctx context.Context, p *path.Contexts) (*service.Contexts, error) {
	obj, err := database.Build(ctx, &ContextListResolvable{p.Capture})
	if err != nil {
		return nil, err
	}
	return obj.(*service.Contexts), nil
}

// Context resolves the single context.
func Context(ctx context.Context, p *path.Context) (*InternalContext, error) {
	boxed, err := database.Resolve(ctx, p.Id.ID())
	if err != nil {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrContextDoesNotExist(p.Id),
			Path:   p.Path(),
		}
	}
	return boxed.(*InternalContext), nil
}

// Resolve implements the database.Resolver interface.
func (r *ContextListResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	seen := map[gfxapi.ContextID]int{}
	contexts := []*path.Context{}

	var currentAtomIndex int
	var currentAtom atom.Atom
	defer func() {
		if r := recover(); r != nil {
			// Add context information to the panic.
			err, ok := r.(error)
			if !ok {
				err = fault.Const(fmt.Sprint(r))
			}
			panic(log.Errf(ctx, err, "panic at atomID: %v, type: %T", currentAtomIndex, currentAtom))
		}
	}()

	s := c.NewState()
	for i, a := range c.Atoms {
		currentAtomIndex, currentAtom = i, a
		a.Mutate(ctx, s, nil)

		api := a.API()
		if api == nil {
			continue
		}
		if context := api.Context(s); context != nil {
			ctxID := context.ID()
			idx, ok := seen[ctxID]
			if !ok {
				idx = len(contexts)
				seen[ctxID] = idx
				id, err := database.Store(ctx, &InternalContext{
					Id:   string(ctxID[:]),
					Api:  &path.API{Id: path.NewID(id.ID(api.ID()))},
					Name: context.Name(),
				})
				if err != nil {
					return nil, err
				}
				contexts = append(contexts, r.Capture.Context(path.NewID(id)))
			}
		}
	}

	return &service.Contexts{List: contexts}, nil
}
