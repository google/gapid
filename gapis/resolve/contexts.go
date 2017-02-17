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
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Contexts resolves the list of contexts belonging to a capture.
func Contexts(ctx log.Context, p *path.Contexts) ([]*service.Context, error) {
	obj, err := database.Build(ctx, &ContextListResolvable{p.Capture})
	if err != nil {
		return nil, err
	}
	return obj.([]*service.Context), nil
}

// Context resolves the single context.
func Context(ctx log.Context, p *path.Context) (*service.Context, error) {
	contexts, err := Contexts(ctx, p.Contexts)
	if err != nil {
		return nil, err
	}
	id := p.Id.ID()
	for _, c := range contexts {
		if c.Id.ID() == id {
			return c, nil
		}
	}
	return nil, &service.ErrInvalidPath{
		Reason: messages.ErrContextDoesNotExist(p.Id),
		Path:   p.Path(),
	}
}

// Resolve implements the database.Resolver interface.
func (r *ContextListResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	atoms, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	seen := map[gfxapi.ContextID]int{}
	contexts := []*service.Context{}
	ranges := []atom.RangeList{}

	var currentAtomIndex int
	var currentAtom atom.Atom
	defer func() {
		if r := recover(); r != nil {
			// Add context information to the panic.
			err, ok := r.(error)
			if !ok {
				err = fault.Const(fmt.Sprint(r))
			}
			panic(cause.Wrap(ctx, err).With("atomID", currentAtomIndex).With("atom", reflect.TypeOf(currentAtom)))
		}
	}()

	s := c.NewState()
	for i, a := range atoms.Atoms {
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
				contexts = append(contexts, &service.Context{
					Id:   path.NewID(id.ID(ctxID)),
					Name: context.Name(),
					Api:  &path.API{Id: path.NewID(id.ID(api.ID()))},
				})
				ranges = append(ranges, atom.RangeList{})
			}
			interval.Merge(&ranges[idx], interval.U64Span{Start: uint64(i), End: uint64(i) + 1}, true)
		}
	}

	for i, r := range ranges {
		contexts[i].Ranges = service.NewCommandRangeList(r)
	}

	return contexts, nil
}
