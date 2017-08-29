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
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Cmds resolves and returns the command list from the path p.
func Cmds(ctx context.Context, p *path.Capture) ([]api.Cmd, error) {
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	return c.Commands, nil
}

// NCmds resolves and returns the command list from the path p, ensuring
// that the number of commands is at least N.
func NCmds(ctx context.Context, p *path.Capture, n uint64) ([]api.Cmd, error) {
	list, err := Cmds(ctx, p)
	if err != nil {
		return nil, err
	}
	if count := uint64(len(list)); n > count {
		return nil, errPathOOB(n-1, "Index", 0, count-1, p.Command(n-1))
	}
	return list, nil
}

// Cmd resolves and returns the command from the path p.
func Cmd(ctx context.Context, p *path.Command) (api.Cmd, error) {
	atomIdx := p.Indices[0]
	if len(p.Indices) > 1 {
		snc, err := SyncData(ctx, p.Capture)
		if err != nil {
			return nil, err
		}

		sg, ok := snc.SubcommandReferences[api.CmdID(atomIdx)]
		if !ok {
			return nil, log.Errf(ctx, nil, "Could not find any subcommands on %v", atomIdx)
		}

		idx := append(api.SubCmdIdx{}, p.Indices[1:]...)
		found := false
		for _, v := range sg {
			if v.Index.Equals(idx) {
				found = true
				atomIdx = uint64(v.GeneratingCmd)
				break
			}
		}
		if !found {
			return nil, log.Errf(ctx, nil, "Could not find subcommand %v", p.Indices)
		}
	}
	cmds, err := NCmds(ctx, p.Capture, atomIdx+1)
	if err != nil {
		return nil, err
	}
	return cmds[atomIdx], nil
}

// Parameter resolves and returns the parameter from the path p.
func Parameter(ctx context.Context, p *path.Parameter) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	cmd := obj.(api.Cmd)
	param, err := api.GetParameter(ctx, cmd, p.Name)
	switch err {
	case nil:
		return param, nil
	case api.ErrParameterNotFound:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist(cmd.CmdName(), p.Name),
			Path:   p.Path(),
		}
	default:
		return nil, err
	}
}

// Result resolves and returns the command's result from the path p.
func Result(ctx context.Context, p *path.Result) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	cmd := obj.(api.Cmd)
	param, err := api.GetResult(ctx, cmd)
	switch err {
	case nil:
		return param, nil
	case api.ErrResultNotFound:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrResultDoesNotExist(cmd.CmdName()),
			Path:   p.Path(),
		}
	default:
		return nil, err
	}
}
