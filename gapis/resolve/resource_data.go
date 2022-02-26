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
	"github.com/google/gapid/gapis/service"
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

	resources, err := buildResources(ctx, r.After, r.Type, r.Config)

	if err != nil {
		return nil, err
	}
	return resources, nil
}

func buildResources(ctx context.Context, p *path.Command, t path.ResourceType, r *path.ResolveConfig) (*ResolvedResources, error) {
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
		idMap[r.ResourceHandle()] = i
		resources[i] = r
	}

	err = api.ForeachCmd(ctx, initialCmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, state, nil, nil); err != nil {
			log.E(ctx, "Get resources at %v: Initial cmd [%v]%v - %v", p.Indices, id, cmd, err)
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		currentCmdResourceCount = 0
		currentCmdIndex = uint64(id)
		if err := cmd.Mutate(ctx, id, state, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
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

			res, err := v.ResourceData(ctx, state, p, r)
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
	resTypes, err := Resources(ctx, p.After.Capture, r)
	if err != nil {
		return nil, err
	}

	id := p.ID.ID()

	t, ok := resTypes.ResourcesToTypes[id.String()]

	if !ok {
		return nil, log.Errf(ctx, nil, "Could not find resource %v", id)
	}

	resources, err := database.Build(ctx, &AllResourceDataResolvable{After: p.After, Type: t, Config: r})
	if err != nil {
		return nil, err
	}
	res, ok := resources.(*ResolvedResources)
	if !ok {
		return nil, fmt.Errorf("Cannot resolve resources at command: %v", p.After)
	}

	if val, ok := res.resourceData[id]; ok {
		if err, isErr := val.(error); isErr {
			return nil, err
		}
		return val, nil
	}

	return nil, fmt.Errorf("Cannot find resource with id: %v", id)
}

// ResourceDatas resolves the data of multiple resources at the specified point in the capture.
func ResourceDatas(ctx context.Context, p *path.MultiResourceData, r *path.ResolveConfig) (interface{}, error) {
	if len(p.IDs) != 0 || !p.All {
		return nil, fmt.Errorf("Get(MultiResourceData) not supported with a list of IDs")
	}

	resources, err := database.Build(ctx, &AllResourceDataResolvable{After: p.After, Type: p.Type, Config: r})
	if err != nil {
		return nil, err
	}
	res, ok := resources.(*ResolvedResources)
	if !ok {
		return nil, fmt.Errorf("Cannot resolve resources at command: %v", p.After)
	}

	m := map[string]*service.MultiResourceData_ResourceOrError{}
	for id, val := range res.resourceData {
		switch v := val.(type) {
		case error:
			m[id.String()] = &service.MultiResourceData_ResourceOrError{
				Val: &service.MultiResourceData_ResourceOrError_Error{
					Error: service.NewError(v),
				},
			}
		case *api.ResourceData:
			m[id.String()] = &service.MultiResourceData_ResourceOrError{
				Val: &service.MultiResourceData_ResourceOrError_Resource{
					Resource: v,
				},
			}
		}
	}
	return &service.MultiResourceData{Resources: m}, nil
}

func ResourceExtras(ctx context.Context, p *path.ResourceExtras, r *path.ResolveConfig) (interface{}, error) {
	cmdIdx := p.After.Indices[0]

	capture, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	allCmds, err := Cmds(ctx, p.After.Capture)
	if err != nil {
		return nil, err
	}

	s, err := SyncData(ctx, p.After.Capture)
	if err != nil {
		return nil, err
	}
	cmds, err := sync.MutationCmdsFor(ctx, p.After.Capture, s, allCmds, api.CmdID(cmdIdx), p.After.Indices[1:], false)
	if err != nil {
		return nil, err
	}
	initialCmds, ranges, err := initialcmds.InitialCommands(ctx, p.After.Capture)
	if err != nil {
		return nil, err
	}

	var currentCmdIndex uint64
	var currentCmdResourceCount int
	var resource api.Resource
	state := capture.NewUninitializedState(ctx).ReserveMemory(ranges)
	state.OnResourceCreated = func(r api.Resource) {
		currentCmdResourceCount++
		if p.ID.ID() == genResourceID(currentCmdIndex, currentCmdResourceCount) {
			resource = r
		}
	}

	err = api.MutateCmds(ctx, state, nil, nil, initialCmds...)
	if err != nil {
		return nil, err
	}
	err = api.MutateCmds(ctx, state, nil, nil, cmds...)
	if err != nil {
		return nil, err
	}

	if resource == nil {
		return nil, fmt.Errorf("Cannot resolve resource %v at command: %v", p.ID.ID(), p.After)
	}

	return resource.ResourceExtras(ctx, state, p.After, r)
}
