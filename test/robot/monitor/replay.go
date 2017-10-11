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

package monitor

import (
	"context"

	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/replay"
)

// Replay is the in memory representation/wrapper for a replay.Action
type Replay struct {
	replay.Action
}

// Replays is the type that manages a set of Replay objects.
type Replays struct {
	entries []*Replay
}

// All returns the complete set of Replay objects we have seen so far.
func (r *Replays) All() []*Replay {
	return r.entries
}

func (o *DataOwner) updateReplay(ctx context.Context, action *replay.Action) error {
	o.Write(func(data *Data) {
		entry, _ := data.Replays.FindOrCreate(ctx, action)
		entry.Action = *action
	})
	return nil
}

// Find searches the replays for the one that matches the supplied action.
// See worker.EquivalentAction for more information about how actions are compared.
func (r *Replays) Find(ctx context.Context, action *replay.Action) *Replay {
	for _, entry := range r.entries {
		if worker.EquivalentAction(&entry.Action, action) {
			return entry
		}
	}
	return nil
}

// FindOrCreate returns the replay that matches the supplied action if it exists, if not
// it creates a new replay object, and returns it.
// It does not register the newly created replay object for you, that will happen only if
// a call is made to trigger the action on the replay service.
func (r *Replays) FindOrCreate(ctx context.Context, action *replay.Action) (*Replay, bool) {
	entry := r.Find(ctx, action)
	if entry != nil {
		return entry, true
	}
	entry = &Replay{Action: *action}
	r.entries = append(r.entries, entry)
	return entry, false
}
