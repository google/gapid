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
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Mesh resolves and returns the Mesh from the path p.
func Mesh(ctx context.Context, p *path.Mesh) (*api.Mesh, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	mesh, err := meshFor(ctx, obj, p)
	switch {
	case err != nil:
		return nil, err
	case mesh != nil:
		return mesh, nil
	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshNotAvailable()}
	}
}

func meshFor(ctx context.Context, o interface{}, p *path.Mesh) (*api.Mesh, error) {
	switch o := o.(type) {
	case api.APIObject:
		if a := o.API(); a != nil {
			if mp, ok := a.(api.MeshProvider); ok {
				return mp.Mesh(ctx, o, p)
			}
		}

	case *service.CommandTreeNode:
		all, err := Atoms(ctx, o.Commands.Capture)
		if err != nil {
			return nil, err
		}
		s, e := o.Commands.From[0], o.Commands.To[0] // TODO: Subcommands
		for i := e; int64(i) >= int64(s); i-- {
			p := o.Commands.Capture.Command(i).Mesh(p.Options.Faceted)
			if mesh, err := meshFor(ctx, all.Atoms[i], p); mesh != nil || err != nil {
				return mesh, err
			}
		}

		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNotADrawCall()}
	}
	return nil, nil
}
