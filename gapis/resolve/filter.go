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

// CommandFilter is a predicate used for filtering commands.
// If the function returns true then the command is considered, otherwise it is
// ignored.
type CommandFilter func(api.CmdID, api.Cmd, *api.GlobalState) bool

// CommandFilters is a list of CommandFilters.
type CommandFilters []CommandFilter

// All is a CommandFilter that needs all the contained filters to pass.
func (l CommandFilters) All(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
	for _, f := range l {
		if !f(id, cmd, s) {
			return false
		}
	}
	return true
}

func buildFilter(ctx context.Context, p *path.Capture, f *path.CommandFilter, sd *sync.Data) (CommandFilter, error) {
	filters := CommandFilters{
		func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
			return !sd.Hidden.Contains(id)
		},
	}
	if f := f.GetContext(); f.IsValid() {
		c, err := Context(ctx, p.Context(f.ID()))
		if err != nil {
			return nil, err
		}
		filters = append(filters, func(id api.CmdID, cmd api.Cmd, s *api.GlobalState) bool {
			if api := cmd.API(); api != nil {
				if ctx := api.Context(s, cmd.Thread()); ctx != nil {
					return ctx.ID() == c.ID
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
	return filters.All, nil
}
