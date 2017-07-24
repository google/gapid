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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// GlobalState resolves the global *api.State at a requested point in a
// capture.
func GlobalState(ctx context.Context, p *path.State) (*api.State, error) {
	obj, err := database.Build(ctx, &GlobalStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(*api.State), nil
}

// APIState resolves the specific API state at a requested point in a capture.
func APIState(ctx context.Context, p *path.State) (interface{}, error) {
	obj, err := database.Build(ctx, &APIStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Resolve implements the database.Resolver interface.
func (r *GlobalStateResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Capture)
	cmdIdx := r.Path.After.Indices[0]
	allCmds, err := Cmds(ctx, r.Path.After.Capture)
	if err != nil {
		return nil, err
	}
	cmds, err := sync.MutationCmdsFor(ctx, r.Path.After.Capture, allCmds, api.CmdID(cmdIdx), r.Path.After.Indices[1:])
	if err != nil {
		return nil, err
	}

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, s, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Resolve implements the database.Resolver interface.
func (r *APIStateResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Capture)
	cmdIdx := r.Path.After.Indices[0]
	if len(r.Path.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported for api state") // TODO: Subcommands
	}
	cmds, err := NCmds(ctx, r.Path.After.Capture, cmdIdx+1)
	if err != nil {
		return nil, err
	}
	return apiState(ctx, cmds, r.Path)
}

func apiState(ctx context.Context, cmds []api.Cmd, p *path.State) (interface{}, error) {
	cmdIdx := p.After.Indices[0]
	if len(p.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported for api state") // TODO: Subcommands
	}
	if count := uint64(len(cmds)); cmdIdx >= count {
		return nil, errPathOOB(cmdIdx, "Index", 0, count-1, p)
	}
	a := cmds[cmdIdx].API()
	if a == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ForeachCmd(ctx, cmds[:cmdIdx+1], func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		cmd.Mutate(ctx, s, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	res, found := s.APIs[a]
	if !found {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	return res, nil
}
