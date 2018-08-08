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
	lastProgressUpdate map[*Task]time.Time
	progressUpdateFreq time.Duration
}

func (statusLogger) OnTaskStart(ctx context.Context, t *Task) {
	log.I(ctx, "%v Started", t)
}

func (l statusLogger) OnTaskProgress(ctx context.Context, t *Task) {
	if time.Since(l.lastProgressUpdate[t]) > l.progressUpdateFreq {
		log.I(ctx, "%v (%v%%) %v", t, t.Completion(), t.TimeSinceStart())
		l.lastProgressUpdate[t] = time.Now()
	}
}

func (l statusLogger) OnTaskFinish(ctx context.Context, t *Task) {
	log.I(ctx, "%v Finished in %v", t, t.TimeSinceStart())
	delete(l.lastProgressUpdate, t)
}
