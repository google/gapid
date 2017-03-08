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
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
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

	list, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	resources := []trackedResource{}
	seen := map[gfxapi.Resource]int{}

	var currentAtomIndex uint64
	var currentAtomResourceCount int

	state := c.NewState()
	state.OnResourceCreated = func(r gfxapi.Resource) {
		currentAtomResourceCount++
		seen[r] = len(seen)
		resources = append(resources, trackedResource{
			resource: r,
			id:       genResourceID(currentAtomIndex, currentAtomResourceCount),
			accesses: []uint64{currentAtomIndex},
		})
	}
	state.OnResourceAccessed = func(r gfxapi.Resource) {
		if index, ok := seen[r]; ok { // Update the list of accesses
			c := len(resources[index].accesses)
			if c == 0 || resources[index].accesses[c-1] != currentAtomIndex {
				resources[index].accesses = append(resources[index].accesses, currentAtomIndex)
			}
		}
	}
	for i, a := range list.Atoms {
		currentAtomResourceCount = 0
		currentAtomIndex = uint64(i)
		a.Mutate(ctx, state, nil /* no builder, just mutate */)
	}

	types := map[gfxapi.ResourceType]*service.ResourcesByType{}
	for _, r := range resources {
		ty := r.resource.ResourceType()
		handle := r.resource.ResourceHandle()
		label := r.resource.ResourceLabel()
		order := r.resource.Order()
		b := types[ty]
		if b == nil {
			b = &service.ResourcesByType{Type: ty}
			types[ty] = b
		}
		b.Resources = append(b.Resources, &service.Resource{
			Id:       path.NewID(r.id),
			Handle:   handle,
			Label:    label,
			Order:    order,
			Accesses: r.accesses,
		})
	}

	out := &service.Resources{Types: make([]*service.ResourcesByType, 0, len(types))}
	for _, v := range types {
		out.Types = append(out.Types, v)
	}

	return out, nil
}

type trackedResource struct {
	resource gfxapi.Resource
	id       id.ID
	name     string
	accesses []uint64
}

func genResourceID(createdAt uint64, rCount int) id.ID {
	return id.OfString(fmt.Sprintf("%d %d", createdAt, rCount))
}
