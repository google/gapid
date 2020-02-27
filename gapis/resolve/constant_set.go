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
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// ConstantSet resolves and returns the constant set from the path p.
func ConstantSet(ctx context.Context, p *path.ConstantSet, r *path.ResolveConfig) (*service.ConstantSet, error) {
	apiID := api.ID(p.API.ID.ID())
	api := api.Find(apiID)
	if api == nil {
		return nil, fmt.Errorf("Unknown API: %v", apiID)
	}

	cs := api.ConstantSets()

	if count := int32(len(cs.Sets)); p.Index >= count {
		return nil, errPathOOB(uint64(p.Index), "Index", 0, uint64(count)-1, p)
	}

	set := cs.Sets[p.Index]

	out := &service.ConstantSet{
		IsBitfield: set.IsBitfield,
		Constants:  make([]*service.Constant, len(set.Entries)),
	}

	for i, e := range set.Entries {
		out.Constants[i] = &service.Constant{
			Name:  cs.Symbols.Get(e),
			Value: e.V,
		}
	}

	return out, nil
}
