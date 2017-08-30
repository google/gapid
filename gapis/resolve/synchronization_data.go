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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service/path"
)

// SyncData resolves and returns the sync.Data from the path p.
func SyncData(ctx context.Context, p *path.Capture) (*sync.Data, error) {
	data, err := database.Build(ctx, &SynchronizationResolvable{Capture: p})
	if err != nil {
		return nil, err
	}
	s, ok := data.(*sync.Data)
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not get synchronization data")
	}
	return s, nil
}

// Resolve builds a SynchronizationResolvable object for the given capture
func (r *SynchronizationResolvable) Resolve(ctx context.Context) (interface{}, error) {
	capture, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	s := sync.NewData()

	if err := addCallerGroups(ctx, s, r.Capture); err != nil {
		return nil, err
	}

	for _, api := range capture.APIs {
		if sync, ok := api.(sync.SynchronizedAPI); ok {
			if err = sync.ResolveSynchronization(ctx, s, r.Capture); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

func addCallerGroups(ctx context.Context, d *sync.Data, c *path.Capture) error {
	cmds, err := Cmds(ctx, c)
	if err != nil {
		return err
	}
	for i, c := range cmds {
		if caller := c.Caller(); caller != api.CmdNoID {
			id := api.CmdID(i)
			d.Hidden.Add(id)
			if d.Hidden.Contains(caller) {
				continue // Most likely a sub-sub-command, which we don't currently support.
			}
			l := d.SubcommandReferences[caller]
			idx := api.SubCmdIdx{uint64(len(l))}
			l = append(l, sync.SubcommandReference{
				Index:         idx,
				GeneratingCmd: id,
				IsCallerGroup: true,
			})
			d.SubcommandReferences[caller] = l
			d.SubcommandGroups[caller] = []api.SubCmdIdx{idx}
		}
	}
	return nil
}
