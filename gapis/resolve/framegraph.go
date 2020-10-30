// Copyright (C) 2020 Google Inc.
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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// Framegraph retrieves the framegraph from the database, creating it if need be.
func Framegraph(ctx context.Context, p *path.Framegraph, r *path.ResolveConfig) (interface{}, error) {
	obj, err := database.Build(ctx, &FramegraphResolvable{Capture: p.Capture})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Resolve implements the Resolvable interface.
func (r *FramegraphResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveGraphicsFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	if len(c.APIs) != 1 {
		return nil, log.Errf(ctx, nil, "Framegraph can be obtained only on a capture with a single API, whereas this capture has %v API(s).", len(c.APIs))
	}
	return c.APIs[0].GetFramegraph(ctx, r.Capture)
}
