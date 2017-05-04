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
	"github.com/google/gapid/gapis/service/path"
)

// ResolvedResources contains all of the resolved resources for a
// particular point in the trace.
type ResolvedResources struct {
	resourceMap  gfxapi.ResourceMap
	resources    map[id.ID]gfxapi.Resource
	resourceData map[id.ID]interface{}
}

// Resolve builds a ResolvedResources object for all of the resources
// at the path r.After
func (r *AllResourceDataResolvable) Resolve(ctx context.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.After.Capture)
	resources, err := buildResources(ctx, r.After)

	if err != nil {
		return nil, err
	}
	return resources, nil
}

func buildResources(ctx context.Context, p *path.Command) (*ResolvedResources, error) {
	atomIdx := p.Index[0]
	if len(p.Index) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}

	list, err := NAtoms(ctx, p.Capture, atomIdx+1)
	if err != nil {
		return nil, err
	}
	state := capture.NewState(ctx)
	var currentAtomIndex uint64
	var currentAtomResourceCount int
	idMap := gfxapi.ResourceMap{}

	resources := make(map[id.ID]gfxapi.Resource)

	state.OnResourceCreated = func(r gfxapi.Resource) {
		currentAtomResourceCount++
		i := genResourceID(currentAtomIndex, currentAtomResourceCount)
		idMap[r] = i
		resources[i] = r
	}

	for i, a := range list.Atoms[:atomIdx+1] {
		currentAtomResourceCount = 0
		currentAtomIndex = uint64(i)
		a.Mutate(ctx, state, nil /* no builder, just mutate */)
	}

	resourceData := make(map[id.ID]interface{})
	for k, v := range resources {
		res, err := v.ResourceData(ctx, state)
		if err != nil {
			resourceData[k] = err
		} else {
			resourceData[k] = res
		}
	}
	return &ResolvedResources{idMap, resources, resourceData}, nil
}

// ResourceData resolves the data of the specified resource at the specified
// point in the capture.
func ResourceData(ctx context.Context, p *path.ResourceData) (interface{}, error) {
	obj, err := database.Build(ctx, &ResourceDataResolvable{p})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourceDataResolvable) Resolve(ctx context.Context) (interface{}, error) {
	resources, err := database.Build(ctx, &AllResourceDataResolvable{r.Path.After})
	if err != nil {
		return nil, err
	}
	res, ok := resources.(*ResolvedResources)
	if !ok {
		return nil, fmt.Errorf("Cannot resolve resources at command: %v", r.Path.After)
	}
	id := r.Path.Id.ID()
	if val, ok := res.resourceData[id]; ok {
		if err, isErr := val.(error); isErr {
			return nil, err
		}
		return val, nil
	}

	return nil, fmt.Errorf("Cannot find resource with id: %v", id)
}
