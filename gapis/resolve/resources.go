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

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Resources resolves all the resources used by the specified capture.
func Resources(ctx context.Context, c *path.Capture, r *path.ResolveConfig) (*service.Resources, error) {
	obj, err := database.Build(ctx, &ResourcesResolvable{Capture: c, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*service.Resources), nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourcesResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = SetupContext(ctx, r.Capture, r.Config)

	c, err := capture.ResolveGraphics(ctx)
	if err != nil {
		return nil, err
	}

	resources := []trackedResource{}
	resourceTypes := map[string]api.ResourceType{}
	seen := map[api.Resource]int{}

	var currentCmdIndex uint64
	var currentCmdResourceCount int
	var currentThread uint64
	var currentAPI api.API
	// If the capture contains initial state, build the necessary commands to recreate it.
	initialCmds, ranges, err := initialcmds.InitialCommands(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	state := c.NewUninitializedState(ctx).ReserveMemory(ranges)
	state.OnResourceCreated = func(res api.Resource) {
		currentCmdResourceCount++
		context := currentAPI.Context(ctx, state, currentThread)
		tr := trackedResource{
			resource: res,
			id:       genResourceID(currentCmdIndex, currentCmdResourceCount),
			context:  r.Capture.Context(id.ID(context.ID())),
			accesses: []uint64{currentCmdIndex},
			created:  currentCmdIndex,
		}
		resources = append(resources, tr)
		seen[res] = len(resources) - 1
		resourceTypes[tr.id.String()] = res.ResourceType(ctx)
	}
	state.OnResourceAccessed = func(r api.Resource) {
		if index, ok := seen[r]; ok { // Update the list of accesses
			c := len(resources[index].accesses)
			if c == 0 || resources[index].accesses[c-1] != currentCmdIndex {
				resources[index].accesses = append(resources[index].accesses, currentCmdIndex)
			}
		}
	}

	// Resources destroyed during state reconstructions should be hidden from the user, as they are
	// temporary objects created to correctly reconstruct the state.
	state.OnResourceDestroyed = func(r api.Resource) {
		delete(seen, r)
	}

	api.ForeachCmd(ctx, initialCmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		currentThread = cmd.Thread()
		currentAPI = cmd.API()
		if err := cmd.Mutate(ctx, id, state, nil, nil); err != nil {
			log.W(ctx, "Get resources: Initial cmd [%v]%v - %v", id, cmd, err)
		}
		return nil
	})

	state.OnResourceDestroyed = func(r api.Resource) {
		if index, ok := seen[r]; ok {
			resources[index].deleted = currentCmdIndex
		}
	}

	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		currentCmdResourceCount = 0
		currentCmdIndex = uint64(id)
		currentThread = cmd.Thread()
		currentAPI = cmd.API()
		cmd.Mutate(ctx, id, state, nil, nil)
		return nil
	})

	types := map[api.ResourceType]*service.ResourcesByType{}
	for _, tr := range resources {
		if _, ok := seen[tr.resource]; !ok {
			continue
		}
		ty := tr.resource.ResourceType(ctx)
		b := types[ty]
		if b == nil {
			b = &service.ResourcesByType{Type: ty}
			types[ty] = b
		}
		b.Resources = append(b.Resources, tr.asService(r.Capture))
	}

	out := &service.Resources{Types: make([]*service.ResourcesByType, 0, len(types))}
	for _, v := range types {
		out.Types = append(out.Types, v)
	}
	out.ResourcesToTypes = resourceTypes

	return out, nil
}

type trackedResource struct {
	resource api.Resource
	id       id.ID
	context  *path.Context
	name     string
	accesses []uint64
	deleted  uint64
	created  uint64
}

func (r trackedResource) asService(p *path.Capture) *service.Resource {
	out := &service.Resource{
		ID:       path.NewID(r.id),
		Context:  r.context,
		Handle:   r.resource.ResourceHandle(),
		Label:    r.resource.ResourceLabel(),
		Order:    r.resource.Order(),
		Accesses: make([]*path.Command, len(r.accesses)),
	}
	for i, a := range r.accesses {
		out.Accesses[i] = p.Command(a)
	}
	if r.deleted > 0 {
		out.Deleted = p.Command(r.deleted)
	}
	out.Created = p.Command(r.created)
	return out
}

func genResourceID(createdAt uint64, rCount int) id.ID {
	return id.OfString(fmt.Sprintf("%d %d", createdAt, rCount))
}
