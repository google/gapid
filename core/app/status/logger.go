// Copyright (C) 2018 Google Inc.
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
	"runtime"
	"sync"
	"time"

	"github.com/google/gapid/core/log"
)

// RegisterLogger registers a status listener that logs the updates to the
// logger bound to the context.
// Task completion updates will only be logged at the given frequency.
func RegisterLogger(progressUpdateFreq time.Duration) Unregister {
	l := &statusLogger{
		lastProgressUpdate: map[*Task]time.Time{},
		progressUpdateFreq: progressUpdateFreq,
	}
	return RegisterListener(l)
}

type statusLogger struct {
	lastProgressUpdate     map[*Task]time.Time
	lastProgressUpdateLock sync.Mutex
	progressUpdateFreq     time.Duration
}

func (l statusLogger) OnTaskStart(ctx context.Context, t *Task) {
	l.lastProgressUpdateLock.Lock()
	l.lastProgressUpdate[t] = time.Now()
	l.lastProgressUpdateLock.Unlock()
	log.I(ctx, "%v Started", t)
}

func (l statusLogger) OnTaskProgress(ctx context.Context, t *Task) {
	l.lastProgressUpdateLock.Lock()
	update := time.Since(l.lastProgressUpdate[t]) > l.progressUpdateFreq
	if update {
		l.lastProgressUpdate[t] = time.Now()
	}
	l.lastProgressUpdateLock.Unlock()

	if update {
		log.I(ctx, "%v (%v%%) %v", t, t.Completion(), t.TimeSinceStart())
	}
}

func (l statusLogger) OnTaskBlock(ctx context.Context, t *Task) {
	log.I(ctx, "%v Blocked", t)
}

func (l statusLogger) OnTaskUnblock(ctx context.Context, t *Task) {
	log.I(ctx, "%v Unblocked", t)
}

func (l statusLogger) OnTaskFinish(ctx context.Context, t *Task) {
	log.I(ctx, "%v Finished in %v", t, t.TimeSinceStart())
	l.lastProgressUpdateLock.Lock()
	delete(l.lastProgressUpdate, t)
	l.lastProgressUpdateLock.Unlock()
}

func (l statusLogger) OnEvent(ctx context.Context, t *Task, n string, s EventScope) {
	if s == TaskScope {
		log.I(ctx, "%v Event: %v %v after start", t, n, t.TimeSinceStart())
	} else {
		log.I(ctx, "%v Event: %v %v", s, n, time.Now())
	}
}

func (l statusLogger) OnMemorySnapshot(ctx context.Context, stats runtime.MemStats) {
	log.I(ctx, "Memory %+v bytes", stats.Alloc)
}

func (l statusLogger) OnReplayStatusUpdate(ctx context.Context, r *Replay, label uint64, totalInstrs, finishedInstrs uint32) {
	log.I(ctx, "Replay Status: started: %v finished: %v label: %v, total instructions: %v, finished instructions: %v.", r.Started(), r.Finished(), label, totalInstrs, finishedInstrs)
}
