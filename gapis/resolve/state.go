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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/framework/binary"
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
func GlobalState(ctx log.Context, p *path.State) (*gfxapi.State, error) {
	obj, err := database.Build(ctx, &GlobalStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(*gfxapi.State), nil
}

// APIState resolves the specific API state at a requested point in a capture.
func APIState(ctx log.Context, p *path.State) (binary.Object, error) {
	obj, err := database.Build(ctx, &APIStateResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj.(binary.Object), nil
}

// Resolve implements the database.Resolver interface.
func (r *GlobalStateResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Commands.Capture)
	list, err := NCommands(ctx, r.Path.After.Commands, r.Path.After.Index+1)
	if err != nil {
		return nil, err
	}
	s := capture.NewState(ctx)
	for _, a := range list.Atoms[:r.Path.After.Index+1] {
		a.Mutate(ctx, s, nil)
	}
	return s, nil
}

// Resolve implements the database.Resolver interface.
func (r *APIStateResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Path.After.Commands.Capture)
	list, err := NCommands(ctx, r.Path.After.Commands, r.Path.After.Index+1)
	if err != nil {
		return nil, err
	}
	return apiState(ctx, list.Atoms, r.Path)
}

func apiState(ctx log.Context, atoms []atom.Atom, p *path.State) (binary.Object, error) {
	if p.After.Index >= uint64(len(atoms)) {
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrValueOutOfBounds(p.After.Index, "Index", uint64(0), uint64(len(atoms)-1)),
			Path:   p.Path(),
		}
	}
	api := atoms[p.After.Index].API()
	if api == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	s := capture.NewState(ctx)
	for _, a := range atoms[:p.After.Index+1] {
		a.Mutate(ctx, s, nil)
	}
	res, found := s.APIs[api]
	if !found {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}
	return res, nil
}
