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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// GlobalState resolves the global *gfxapi.State at a requested point in a
// capture.
func GlobalState(ctx context.Context, p *path.State) (*gfxapi.State, error) {
	obj, err := database.Build(ctx, &GlobalStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(*gfxapi.State), nil
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
	atomIdx := r.Path.After.Indices[0]
	if len(r.Path.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	list, err := NAtoms(ctx, r.Path.After.Capture, atomIdx+1)
	if err != nil {
		return nil, err
	}
	s := capture.NewState(ctx)
	for _, a := range list.Atoms[:atomIdx+1] {
		a.Mutate(ctx, s, nil)
	}
	return s, nil
}

// Resolve implements the database.Resolver interface.
func (r *APIStateResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Capture)
	atomIdx := r.Path.After.Indices[0]
	if len(r.Path.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	list, err := NAtoms(ctx, r.Path.After.Capture, atomIdx+1)
	if err != nil {
		return nil, err
	}
	return apiState(ctx, list.Atoms, r.Path)
}

func apiState(ctx context.Context, atoms []atom.Atom, p *path.State) (interface{}, error) {
	atomIdx := p.After.Indices[0]
	if len(p.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	if count := uint64(len(atoms)); atomIdx >= count {
		return nil, errPathOOB(atomIdx, "Index", 0, count-1, p)
	}
	api := atoms[atomIdx].API()
	if api == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	s := capture.NewState(ctx)
	for _, a := range atoms[:atomIdx+1] {
		a.Mutate(ctx, s, nil)
	}
	res, found := s.APIs[api]
	if !found {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	return res, nil
}
