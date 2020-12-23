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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// ResourceMeta returns the metadata for the specified resource.
func ResourceMeta(ctx context.Context, ids []*path.ID, after *path.Command, r *path.ResolveConfig) (*api.ResourceMeta, error) {
	obj, err := database.Build(ctx, &ResourceMetaResolvable{IDs: ids, After: after, Config: r})
	if err != nil {
		return nil, err
	}
	return obj.(*api.ResourceMeta), nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourceMetaResolvable) Resolve(ctx context.Context) (interface{}, error) {
	resources, err := database.Build(ctx, &AllResourceDataResolvable{After: r.After, Config: r.Config})
	if err != nil {
		return nil, err
	}
	res, ok := resources.(*ResolvedResources)
	if !ok {
		return nil, fmt.Errorf("Cannot resolve resources at command: %v", r.After)
	}
	ids := r.IDs
	values := make([]api.Resource, len(ids))
	for i, id := range ids {
		val, ok := res.resources[id.ID()]
		if !ok {
			return nil, fmt.Errorf("Could not find resource %v", id.ID())
		}
		values[i] = val
	}
	result := &api.ResourceMeta{
		IDMap:     res.resourceMap,
		Resources: values,
	}
	return result, nil
}

// ResourceIDMap returns the ResourceMap at the given command.
func ResourceIDMap(ctx context.Context, after *path.Command, r *path.ResolveConfig) (api.ResourceMap, error) {
	resources, err := Resources(ctx, after.Capture, r)
	if err != nil {
		return nil, err
	}

	m := api.ResourceMap{}
	created := map[string]*path.Command{}
	for _, ty := range resources.Types {
		for _, res := range ty.Resources {
			if !res.Created.IsAfter(after) {
				if other, ok := created[res.Handle]; ok && other.IsAfter(res.Created) {
					// This handle is being re-used, and the previously seen one was
					// created later, so ignore the current resource.
					continue
				}

				m[res.Handle] = res.ID.ID()
				created[res.Handle] = res.Created
			}
		}
	}

	return m, nil
}
