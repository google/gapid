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
	"sort"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
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

// Importance is the interface implemeneted by commands that provide an
// "importance score". This value is used to prioritize contexts.
type Importance interface {
	Importance() int
}

// Resolve implements the database.Resolver interface.
func (r *ContextListResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	priorities := map[api.ContextID]int{}
	contexts := []api.Context{}

	s := c.NewState()
	err = api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, i api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, i, s, nil)

		api := cmd.API()
		if api == nil {
			return nil
		}

		context := api.Context(s, cmd.Thread())
		if context == nil {
			return nil
		}

		id := context.ID()
		p, ok := priorities[id]
		if !ok {
			priorities[id] = p
			contexts = append(contexts, context)
		}
		if i, ok := cmd.(Importance); ok {
			p += i.Importance()
			priorities[id] = p
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(contexts, func(i, j int) bool {
		return priorities[contexts[i].ID()] < priorities[contexts[j].ID()]
	})

	out := &service.Contexts{
		List: make([]*path.Context, len(contexts)),
	}

	for i, c := range contexts {
		api := c.API()
		ctxID := c.ID()
		id, err := database.Store(ctx, &InternalContext{
			Id:       ctxID[:],
			Api:      &path.API{Id: path.NewID(id.ID(api.ID()))},
			Name:     c.Name(),
			Priority: uint32(i),
		})
		if err != nil {
			return nil, err
		}
		out.List[i] = r.Capture.Context(path.NewID(id))
	}
	return out, nil
}

func (i *InternalContext) ID() api.ContextID {
	var out api.ContextID
	copy(out[:], i.Id)
	return out
}
