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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/service/path"
)

type filter func(api.CmdID, api.Cmd, *api.GlobalState) bool

func buildFilter(ctx context.Context, p *path.Capture, f *path.CommandFilter, sd *sync.Data) (filter, error) {
	filters := []filter{
		func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
			return !sd.Hidden.Contains(id)
		},
	}
	if c := f.GetContext(); c.IsValid() {
		c, err := Context(ctx, p.Context(c))
		if err != nil {
			return nil, err
		}
		ctxID := c.ID()
		filters = append(filters, func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
			if api := cmd.API(); api != nil {
				if ctx := api.Context(s, cmd.Thread()); ctx != nil {
					return ctx.ID() == ctxID
				}
			}
			return false
		})
	}
	if len(f.GetThreads()) > 0 {
		filters = append(filters, func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
			thread := cmd.Thread()
			for _, t := range f.Threads {
				if t == thread {
					return true
				}
			}
			return false
		})
	}
	return func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
		for _, f := range filters {
			if !f(id, cmd, s) {
				return false
			}
		}
		return true
	}, nil
}
