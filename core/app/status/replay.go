// Copyright (C) 2019 Google Inc.
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

package status

import (
	"context"
	"sync"

	"github.com/google/gapid/core/data/id"
)

// Replay contains status information about a replay.
type Replay struct {
	ID       uint32
	Device   id.ID
	started  bool
	finished bool
	mutex    sync.RWMutex
}

// ReplayQueued notifies listeners that a new replay has been queued.
func ReplayQueued(ctx context.Context, id uint32, device id.ID) *Replay {
	r := &Replay{
		ID:       id,
		Device:   device,
		started:  false,
		finished: false,
	}
	onReplayStatusUpdate(ctx, r, 0, 0, 0)
	return r
}

// Started returns wether a replay has started.
func (r *Replay) Started() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.started
}

// Finished returns wether a replay has finished.
func (r *Replay) Finished() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.finished
}

// Start notifies listeners that a replay has started.
func (r *Replay) Start(ctx context.Context) {
	r.start()
	onReplayStatusUpdate(ctx, r, 0, 0, 0)
}

// Progress notifies listeners of progress on a currently running replay.
func (r *Replay) Progress(ctx context.Context, label uint64, totalInstrs, finishedInstrs uint32) {
	onReplayStatusUpdate(ctx, r, label, totalInstrs, finishedInstrs)
}

// Finish notifies listeners that a replay has finished.
func (r *Replay) Finish(ctx context.Context) {
	r.finish()
	onReplayStatusUpdate(ctx, r, 0, 0, 0)
}

func (r *Replay) start() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.started, r.finished = true, false
}

func (r *Replay) finish() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.started, r.finished = true, true
}
