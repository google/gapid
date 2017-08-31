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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Resources resolves all the resources used by the specified capture.
func Resources(ctx context.Context, c *path.Capture) (*service.Resources, error) {
	obj, err := database.Build(ctx, &ResourcesResolvable{c})
	if err != nil {
		return nil, err
	}
	return obj.(*service.Resources), nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourcesResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	resources := []trackedResource{}
	seen := map[api.Resource]int{}

	var currentCmdIndex uint64
	var currentCmdResourceCount int

	state := c.NewState()
	state.OnResourceCreated = func(r api.Resource) {
		currentCmdResourceCount++
		seen[r] = len(seen)
		resources = append(resources, trackedResource{
			resource: r,
			id:       genResourceID(currentCmdIndex, currentCmdResourceCount),
			accesses: []uint64{currentCmdIndex},
		})
	}
	state.OnResourceAccessed = func(r api.Resource) {
		if index, ok := seen[r]; ok { // Update the list of accesses
			c := len(resources[index].accesses)
			if c == 0 || resources[index].accesses[c-1] != currentCmdIndex {
				resources[index].accesses = append(resources[index].accesses, currentCmdIndex)
			}
		}
	}

	api.ForeachCmd(ctx, c.Commands, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		currentCmdResourceCount = 0
		currentCmdIndex = uint64(id)
		cmd.Mutate(ctx, id, state, nil)
		return nil
	})

	types := map[api.ResourceType]*service.ResourcesByType{}
	for _, tr := range resources {
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

	return out, nil
}

type trackedResource struct {
	resource api.Resource
	id       id.ID
	name     string
	accesses []uint64
}

func (r trackedResource) asService(p *path.Capture) *service.Resource {
	out := &service.Resource{
		Id:       path.NewID(r.id),
		Handle:   r.resource.ResourceHandle(),
		Label:    r.resource.ResourceLabel(),
		Order:    r.resource.Order(),
		Accesses: make([]*path.Command, len(r.accesses)),
	}
	for i, a := range r.accesses {
		out.Accesses[i] = p.Command(a)
	}
	return out
}

func genResourceID(createdAt uint64, rCount int) id.ID {
	return id.OfString(fmt.Sprintf("%d %d", createdAt, rCount))
}
