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
	"fmt"
	"runtime"
	"sync"
)

var listeners = map[int]Listener{}
var listenerMutex sync.RWMutex
var nextListenerID int

// EventScope defines the scope for a particular event
type EventScope int

const (
	// TaskScope is an event that applies to the current task
	TaskScope EventScope = iota
	// ProcessScope is an event that only applies to this process
	ProcessScope
	// GlobalScope is an event that applies globally
	GlobalScope
)

func (s EventScope) String() string {
	switch s {
	case TaskScope:
		return "Task"
	case ProcessScope:
		return "Process"
	case GlobalScope:
		return "Global"
	default:
		return "Unknown"
	}
}

// Listener is the interface implemented by types that want to listen to
// application status messages.
type Listener interface {
	OnTaskStart(context.Context, *Task)
	OnTaskProgress(context.Context, *Task)
	OnTaskFinish(context.Context, *Task)
	OnEvent(context.Context, *Task, string, EventScope)
	OnMemorySnapshot(context.Context, runtime.MemStats)
	OnTaskBlock(context.Context, *Task)
	OnTaskUnblock(context.Context, *Task)
	OnReplayStatusUpdate(context.Context, *Replay, uint64, uint32, uint32)
}

// Unregister is the function returned by RegisterListener and is used to
// unregister the listenenr.
type Unregister func()

// RegisterListener registers l for status updates
func RegisterListener(l Listener) Unregister {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	id := nextListenerID
	nextListenerID++
	listeners[id] = l
	return func() {
		listenerMutex.Lock()
		defer listenerMutex.Unlock()
		delete(listeners, id)
	}
}

func onTaskStart(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskStart(ctx, t)
	}
}

func onTaskProgress(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskProgress(ctx, t)
	}
}

func onBlock(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskBlock(ctx, t)
	}
}

func onUnblock(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskUnblock(ctx, t)
	}
}

func onTaskFinish(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskFinish(ctx, t)
	}
}

func onEvent(ctx context.Context, t *Task, scope EventScope, name string, args []interface{}) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	if len(listeners) > 0 {
		msg := fmt.Sprintf(name, args...)
		for _, l := range listeners {
			l.OnEvent(ctx, t, msg, scope)
		}
	}
}

func onMemorySnapshot(ctx context.Context, snapshot runtime.MemStats) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnMemorySnapshot(ctx, snapshot)
	}
}

func onReplayStatusUpdate(ctx context.Context, r *Replay, label uint64, totalInstrs, finishedInstrs uint32) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnReplayStatusUpdate(ctx, r, label, totalInstrs, finishedInstrs)
	}
}
