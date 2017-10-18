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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// GlobalState resolves the global *api.GlobalState at a requested point in a
// capture.
func GlobalState(ctx context.Context, p *path.GlobalState) (*api.GlobalState, error) {
	obj, err := database.Build(ctx, &GlobalStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(*api.GlobalState), nil
}

// State resolves the specific API state at a requested point in a capture.
func State(ctx context.Context, p *path.State) (interface{}, error) {
	return database.Build(ctx, &StateResolvable{p})
}

// Resolve implements the database.Resolver interface.
func (r *GlobalStateResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Capture)
	cmdIdx := r.Path.After.Indices[0]
	allCmds, err := Cmds(ctx, r.Path.After.Capture)
	if err != nil {
		return nil, err
	}

	sd, err := SyncData(ctx, r.Path.After.Capture)
	if err != nil {
		return nil, err
	}
	cmds, err := sync.MutationCmdsFor(ctx, r.Path.After.Capture, sd, allCmds, api.CmdID(cmdIdx), r.Path.After.Indices[1:], false)
	if err != nil {
		return nil, err
	}

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, id, s, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Resolve implements the database.Resolver interface.
func (r *StateResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Capture)
	obj, _, _, err := state(ctx, r.Path)
	return obj, err
}

func state(ctx context.Context, p *path.State) (interface{}, path.Node, api.ID, error) {
	cmd, err := Cmd(ctx, p.After)
	if err != nil {
		return nil, nil, api.ID{}, err
	}

	a := cmd.API()
	if a == nil {
		return nil, nil, api.ID{}, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	g, err := GlobalState(ctx, p.After.GlobalStateAfter())
	if err != nil {
		return nil, nil, api.ID{}, err
	}

	state := g.APIs[a.ID()]
	if state == nil {
		return nil, nil, api.ID{}, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	root, err := state.Root(ctx, p)
	if err != nil {
		return nil, nil, api.ID{}, err
	}
	if root == nil {
		return nil, nil, api.ID{}, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	// Transform the State path node to a GlobalState node to prevent the
	// object load recursing back into this function.
	abs := path.Transform(root, func(n path.Node) path.Node {
		switch n := n.(type) {
		case *path.State:
			return APIStateAfter(p.After, a.ID())
		default:
			return n
		}
	})

	obj, err := Get(ctx, abs.Path())
	if err != nil {
		return nil, nil, api.ID{}, err
	}

	return obj, abs, a.ID(), nil
}

// APIStateAfter returns an absolute path to the API state after c.
func APIStateAfter(c *path.Command, a api.ID) path.Node {
	p := &path.GlobalState{After: c}
	return p.Field("APIs").MapIndex(a)
}
