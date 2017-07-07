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

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service/path"
)

type filter func(api.Cmd, *api.State) bool

func buildFilter(ctx context.Context, p *path.Capture, f *path.CommandFilter) (filter, error) {
	filters := []filter{}
	if c := f.GetContext(); c.IsValid() {
		c, err := Context(ctx, p.Context(c))
		if err != nil {
			return nil, err
		}
		id, err := id.Parse(c.Id)
		if err != nil {
			return nil, err
		}
		ctxID := api.ContextID(id)
		filters = append(filters, func(cmd api.Cmd, s *api.State) bool {
			if api := cmd.API(); api != nil {
				if ctx := api.Context(s, cmd.Thread()); ctx != nil {
					return ctx.ID() == ctxID
				}
			}
			return false
		})
	}
	if len(f.GetThreads()) > 0 {
		filters = append(filters, func(cmd api.Cmd, s *api.State) bool {
			thread := cmd.Thread()
			for _, t := range f.Threads {
				if t == thread {
					return true
				}
			}
			return false
		})
	}
	return func(cmd api.Cmd, s *api.State) bool {
		for _, f := range filters {
			if !f(cmd, s) {
				return false
			}
		}
		return true
	}, nil
}
