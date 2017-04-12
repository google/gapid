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

	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Mesh resolves and returns the Mesh from the path p.
func Mesh(ctx context.Context, p *path.Mesh) (*gfxapi.Mesh, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	if ao, ok := obj.(gfxapi.APIObject); ok {
		if api := ao.API(); api != nil {
			if m, ok := api.(gfxapi.MeshProvider); ok {
				mesh, err := m.Mesh(ctx, obj, p)
				switch {
				case err != nil:
					return nil, err
				case mesh != nil:
					return mesh, nil
				}
			}
		}
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshNotAvailable()}
}
