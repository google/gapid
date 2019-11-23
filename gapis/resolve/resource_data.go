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

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service/path"
)

// ResolvedResources contains all of the resolved resources for a
// particular point in the trace.
type ResolvedResources struct {
	resourceMap  api.ResourceMap
	resources    map[id.ID]api.Resource
	resourceData map[id.ID]interface{}
}

// Resolve builds a ResolvedResources object for all of the resources
// at the path r.After
func (r *AllResourceDataResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, r.After.Capture, r.Config)

	resources, err := buildResources(ctx, r.After, r.Type)

	if err != nil {
		return nil, err
	}
	return resources, nil
}

func buildResources(ctx context.Context, p *path.Command, t api.ResourceType) (*ResolvedResources, error) {
	cmdIdx := p.Indices[0]

	capture, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	allCmds, err := Cmds(ctx, p.Capture)

	if err != nil {
		return nil, err
	}

	s, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	cmds, err := sync.MutationCmdsFor(ctx, p.Capture, s, allCmds, api.CmdID(cmdIdx), p.Indices[1:], false)
	if err != nil {
		return nil, err
	}
	initialCmds, ranges, err := initialcmds.InitialCommands(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	state := capture.NewUninitializedState(ctx).ReserveMemory(ranges)
	var currentCmdIndex uint64
	var currentCmdResourceCount int
	idMap := api.ResourceMap{}

	resources := make(map[id.ID]api.Resource)

	state.OnResourceCreated = func(r api.Resource) {
		currentCmdResourceCount++
		i := genResourceID(currentCmdIndex, currentCmdResourceCount)
		idMap[r] = i
		resources[i] = r
	}

	api.ForeachCmd(ctx, initialCmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, state, nil, nil); err != nil {
			log.W(ctx, "Get resources at %v: Initial cmd [%v]%v - %v", p.Indices, id, cmd, err)
		}
		return nil
	})
	err = api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		currentCmdResourceCount = 0
		currentCmdIndex = uint64(id)
		cmd.Mutate(ctx, id, state, nil, nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	resourceData := make(map[id.ID]interface{})
	i := uint64(0)
	for k, v := range resources {
		if v.ResourceType(ctx) == t {
			// To prevent too much spam, only log 1% of the data
			if i%100 == 1 || i == uint64(len(resources))-1 {
				status.UpdateProgress(ctx, i, uint64(len(resources)))
			}
			i++

			res, err := v.ResourceData(ctx, state, p)
			if err != nil {
				resourceData[k] = err
			} else {
				resourceData[k] = res
			}
		}
	}
	return &ResolvedResources{idMap, resources, resourceData}, nil
}

// ResourceData resolves the data of the specified resource at the specified
// point in the capture.
func ResourceData(ctx context.Context, p *path.ResourceData, r *path.ResolveConfig) (interface{}, error) {
	obj, err := database.Build(ctx, &ResourceDataResolvable{Path: p, Config: r})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourceDataResolvable) Resolve(ctx context.Context) (interface{}, error) {
	resTypes, err := Resources(ctx, r.Path.After.Capture, r.Config)
	if err != nil {
		return nil, err
	}

	id := r.Path.ID.ID()

	t, ok := resTypes.ResourcesToTypes[id.String()]

	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find resource %v", id)
	}

	resources, err := database.Build(ctx, &AllResourceDataResolvable{After: r.Path.After, Type: t, Config: r.Config})
	if err != nil {
		return nil, err
	}
	res, ok := resources.(*ResolvedResources)
	if !ok {
		return nil, fmt.Errorf("Cannot resolve resources at command: %v", r.Path.After)
	}

	if val, ok := res.resourceData[id]; ok {
		if err, isErr := val.(error); isErr {
			return nil, err
		}
		return val, nil
	}

	return nil, fmt.Errorf("Cannot find resource with id: %v", id)
}
