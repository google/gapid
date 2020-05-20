// Copyright (C) 2019 Google Inc.
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
	"math"

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// Delete creates a copy of the capture referenced by p, but without the object, value
// or memory at p. The path returned is identical to p, but with
// the base changed to refer to the new capture.
func Delete(ctx context.Context, p *path.Any, r *path.ResolveConfig) (*path.Any, error) {
	obj, err := database.Build(ctx, &DeleteResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}

	return obj.(*path.Any), nil
}

// Resolve implements the database.Resolver interface.
func (r *DeleteResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, path.FindCapture(r.Path.Node()), r.Config)

	p, err := deleteCommand(ctx, arena.New(), r.Path.Node())
	if err != nil {
		return nil, err
	}

	return p.Path(), nil
}

func deleteCommand(ctx context.Context, a arena.Arena, p path.Node) (*path.Capture, error) {
	switch p := p.(type) {
	case *path.Command:
		if len(p.Indices) > 1 {
			return nil, fmt.Errorf("Cannot modify subcommands") // TODO: Subcommands
		}

		cmdIdx := p.Indices[0]

		// Resolve the command list
		oldCmds, err := NCmds(ctx, p.Capture, cmdIdx+1)
		if err != nil {
			return nil, err
		}

		cmds := removeCommandFromList(cmdIdx, oldCmds, a)

		// Create the new capture
		old, err := capture.ResolveGraphicsFromPath(ctx, p.Capture)
		if err != nil {
			return nil, err
		}

		c, err := capture.NewGraphicsCapture(ctx, a, old.Name()+"*", old.Header, old.InitialState, cmds)
		if err != nil {
			return nil, err
		}

		newCapure, err := capture.New(ctx, c)
		if err != nil {
			return nil, err
		}

		return newCapure, nil
	}
	return nil, fmt.Errorf("Incorrect path type %T", p)
}

func removeCommandFromList(cmdIdx uint64, oldCmds []api.Cmd, a arena.Arena) []api.Cmd {
	const MAXID = math.MaxUint64

	cmds := make([]api.Cmd, 0, uint64(len(oldCmds))-1)
	cmds = append(cmds, oldCmds[:cmdIdx]...)
	cmds = append(cmds, oldCmds[cmdIdx+1:]...)

	return cmds
}
