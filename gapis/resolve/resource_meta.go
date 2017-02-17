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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/service/path"
)

// ResourceMeta returns the metadata for the specified resource.
func ResourceMeta(ctx log.Context, id *path.ID, after *path.Command) (*gfxapi.ResourceMeta, error) {
	obj, err := database.Build(ctx, &ResourceMetaResolvable{id, after})
	if err != nil {
		return nil, err
	}
	return obj.(*gfxapi.ResourceMeta), nil
}

// Resolve implements the database.Resolver interface.
func (r *ResourceMetaResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.After.Commands.Capture)
	state := capture.NewState(ctx)
	result := &gfxapi.ResourceMeta{
		IDMap: gfxapi.ResourceMap{},
	}
	p := &path.ResourceData{Id: r.Id, After: r.After}
	resource, err := buildResource(ctx, state, p, result.IDMap)
	if err != nil {
		return nil, err
	}
	result.Resource = resource
	return result, nil
}
