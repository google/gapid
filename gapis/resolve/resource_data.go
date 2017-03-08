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

	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service/path"
)

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
	ctx = capture.Put(ctx, r.Path.After.Commands.Capture)

	state := capture.NewState(ctx)
	IDMap := gfxapi.ResourceMap{}
	// TODO: Persist state in getResourceMeta and reuse it here.
	resource, err := buildResource(ctx, state, r.Path, IDMap)
	if err != nil {
		return nil, err
	}
	return resource.ResourceData(ctx, state, IDMap)
}

func buildResource(ctx context.Context, state *gfxapi.State, p *path.ResourceData, IDMap gfxapi.ResourceMap) (gfxapi.Resource, error) {
	list, err := NCommands(ctx, p.After.Commands, p.After.Index+1)
	if err != nil {
		return nil, err
	}
	var currentAtomIndex uint64
	var resource gfxapi.Resource
	var currentAtomResourceCount int
	id := p.Id.ID()
	state.OnResourceCreated = func(r gfxapi.Resource) {
		currentAtomResourceCount++
		i := genResourceID(currentAtomIndex, currentAtomResourceCount)
		IDMap[r] = i
		if i == id {
			resource = r
		}
	}
	for i, a := range list.Atoms[:p.After.Index+1] {
		currentAtomResourceCount = 0
		currentAtomIndex = uint64(i)
		a.Mutate(ctx, state, nil /* no builder, just mutate */)
	}
	if resource != nil {
		return resource, nil
	}
	return nil, fmt.Errorf("Resource with id %v not found", p.Id.ID())
}
