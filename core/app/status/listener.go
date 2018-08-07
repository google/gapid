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
	"sync"
)

var listeners = map[int]Listener{}
var listenerMutex sync.RWMutex
var nextListenerID int

// Listener is the interface implemented by types that want to listen to
// application status messages.
type Listener interface {
	OnTaskStart(context.Context, *Task)
	OnTaskProgress(context.Context, *Task)
	OnTaskFinish(context.Context, *Task)
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

func onTaskFinish(ctx context.Context, t *Task) {
	listenerMutex.RLock()
	defer listenerMutex.RUnlock()
	for _, l := range listeners {
		l.OnTaskFinish(ctx, t)
	}
}
