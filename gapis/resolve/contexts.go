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
	"reflect"
	"sort"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/extensions"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Contexts resolves the list of contexts belonging to a capture.
func Contexts(ctx context.Context, p *path.Contexts) ([]*api.ContextInfo, error) {
	obj, err := database.Build(ctx, &ContextListResolvable{p.Capture})
	if err != nil {
		return nil, err
	}
	return obj.([]*api.ContextInfo), nil
}

// ContextsByID resolves the list of contexts belonging to a capture.
func ContextsByID(ctx context.Context, p *path.Contexts) (map[api.ContextID]*api.ContextInfo, error) {
	ctxs, err := Contexts(ctx, p)
	if err != nil {
		return nil, err
	}
	out := map[api.ContextID]*api.ContextInfo{}
	for _, c := range ctxs {
		out[c.ID] = c
	}
	return out, nil
}

// Context resolves the single context.
func Context(ctx context.Context, p *path.Context) (*api.ContextInfo, error) {
	contexts, err := Contexts(ctx, p.Capture.Contexts())
	if err != nil {
		return nil, err
	}
	id := api.ContextID(p.Id.ID())
	for _, c := range contexts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, &service.ErrInvalidPath{
		Reason: messages.ErrContextDoesNotExist(p.Id),
		Path:   p.Path(),
	}
}

// Importance is the interface implemeneted by commands that provide an
// "importance score". This value is used to prioritize contexts.
type Importance interface {
	Importance() int
}

// Named is the interface implemented by context that have a name.
type Named interface {
	Name() string
}

// Resolve implements the database.Resolver interface.
func (r *ContextListResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	type ctxInfo struct {
		ctx  api.Context
		cnts map[reflect.Type]int
		pri  int
	}

	seen := map[api.ContextID]int{}
	contexts := []*ctxInfo{}

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
		idx, ok := seen[id]
		if !ok {
			idx = len(contexts)
			seen[id] = idx
			contexts = append(contexts, &ctxInfo{
				ctx:  context,
				cnts: map[reflect.Type]int{},
			})
		}

		c := contexts[idx]
		cmdTy := reflect.TypeOf(cmd)
		c.cnts[cmdTy] = c.cnts[cmdTy] + 1
		if i, ok := cmd.(Importance); ok {
			c.pri += i.Importance()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].pri > contexts[j].pri
	})

	out := make([]*api.ContextInfo, len(contexts))
	for i, c := range contexts {
		name := fmt.Sprintf("Context %v", i)
		if n, ok := c.ctx.(Named); ok {
			name = n.Name()
		}
		out[i] = &api.ContextInfo{
			Path:              r.Capture.Context(id.ID(c.ctx.ID())),
			ID:                c.ctx.ID(),
			API:               c.ctx.API().ID(),
			NumCommandsByType: c.cnts,
			Name:              name,
			Priority:          len(contexts)-i,
			UserData:          map[interface{}]interface{}{},
		}
	}

	for _, e := range extensions.Get() {
		if e.AdjustContexts != nil {
			e.AdjustContexts(ctx, out)
		}
	}

	return out, nil
}
