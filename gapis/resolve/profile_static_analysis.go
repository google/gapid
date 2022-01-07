// Copyright (C) 2021 Google Inc.
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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service/path"
)

// ProfileStaticAnalysis resolves the static analysis profiling data.
func ProfileStaticAnalysis(ctx context.Context, p *path.Capture) (*api.StaticAnalysisProfileData, error) {
	// TODO(pmuetschard): maybe put this into the database?

	c, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(c.APIs) != 1 {
		return nil, log.Errf(ctx, nil, "Profile static analysis can be obtained only on a capture with a single API, whereas this capture has %v API(s).", len(c.APIs))
	}
	return c.APIs[0].ProfileStaticAnalysis(ctx, p)
}
