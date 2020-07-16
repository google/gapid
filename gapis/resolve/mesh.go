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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Mesh resolves and returns the Mesh from the path p.
func Mesh(ctx context.Context, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	obj, err := ResolveInternal(ctx, p.Parent(), r)
	if err != nil {
		return nil, err
	}
	mesh, err := meshFor(ctx, obj, p, r)
	switch {
	case err != nil:
		return nil, err
	case mesh != nil:
		return mesh, nil
	default:
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshNotAvailable()}
	}
}

func meshFor(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	switch o := o.(type) {
	case api.APIObject:
		if a := o.API(); a != nil {
			if mp, ok := a.(api.MeshProvider); ok {
				return mp.Mesh(ctx, o, p, r)
			}
		}

	case *service.CommandTreeNode:
		cmds, err := Cmds(ctx, o.Commands.Capture)
		if err != nil {
			return nil, err
		}

		if len(o.Commands.From) != len(o.Commands.To) {
			return nil, log.Errf(ctx, nil, "Subcommand indices must be the same length")
		}

		lastSubcommand := len(o.Commands.From) - 1
		for i := 0; i < lastSubcommand; i++ {
			if o.Commands.From[i] != o.Commands.To[i] {
				return nil, log.Errf(ctx, nil, "Subcommand ranges must be identical everywhere but the last element")
			}
		}

		cmd := append([]uint64{}, o.Commands.From...) // make a copy of o.Commands.From
		for i := o.Commands.To[lastSubcommand]; i >= o.Commands.From[lastSubcommand]; i-- {
			cmd[lastSubcommand] = i
			p := o.Commands.Capture.Command(cmd[0], cmd[1:]...).Mesh(p.Options)
			if mesh, err := meshFor(ctx, cmds[o.Commands.From[0]], p, r); mesh != nil || err != nil {
				return mesh, err
			}
		}

		return nil, &service.ErrDataUnavailable{Reason: messages.ErrNotADrawCall()}
	}
	return nil, nil
}
