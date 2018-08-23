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
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var app = Task{
	begun:    time.Now(),
	children: map[*Task]struct{}{},
}

var nextID = uint64(1)

// Task represents a long running job which should be reported as part of the
// application's status.
type Task struct {
	id         uint64
	name       string
	begun      time.Time
	completion float32
	parent     *Task
	children   map[*Task]struct{}
	mutex      sync.RWMutex
}

func (t *Task) add(c *Task) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.children[c] = struct{}{}
}

func (t *Task) remove(c *Task) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.children, c)
}

func (t *Task) String() string {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	parent := ""
	if t.parent != nil {
		parent = fmt.Sprint(t.parent)
	}
	if parent != "" {
		return fmt.Sprintf("%v → %v", parent, t.name)
	}
	return t.name
}

// ID returns the task's unique identifier.
func (t *Task) ID() uint64 { t.mutex.RLock(); defer t.mutex.RUnlock(); return t.id }

// Name returns the task's name.
func (t *Task) Name() string { t.mutex.RLock(); defer t.mutex.RUnlock(); return t.name }

// TimeSinceStart returns the time the task was started.
func (t *Task) TimeSinceStart() time.Duration {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return time.Since(t.begun)
}

// Completion returns the task's completion as a percentage.
func (t *Task) Completion() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return int(t.completion * 100)
}

// SubTasks returns all the sub-tasks of this task.
func (t *Task) SubTasks() []*Task {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	l := make([]*Task, 0, len(t.children))
	for t := range t.children {
		l = append(l, t)
	}
	sort.Slice(l, func(i, j int) bool { return l[i].begun.Before(l[j].begun) })

	return l
}

// Start returns a new context for a long running task.
// End() must be called with the returned context once the task has been
// finished.
func Start(ctx context.Context, name string, args ...interface{}) context.Context {
	parent := GetTask(ctx)
	if parent == nil {
		parent = &app
	}
	t := &Task{
		id:       atomic.AddUint64(&nextID, 1),
		name:     fmt.Sprintf(name, args...),
		begun:    time.Now(),
		parent:   parent,
		children: map[*Task]struct{}{},
	}
	parent.add(t)
	onTaskStart(ctx, t)
	return PutTask(ctx, t)
}

// UpdateProgress updates the progress of the task started with Start().
// n is the number of units of completion, which ranges from [0, outof).
func UpdateProgress(ctx context.Context, n, outof int) {
	t := GetTask(ctx)
	if t == nil {
		panic("status.UpdateProgress called with no corresponding status.Start")
	}
	t.completion = float32(n) / float32(outof)
	onTaskProgress(ctx, t)
}

// Finish marks the task started with Start() as finished.
func Finish(ctx context.Context) {
	t := GetTask(ctx)
	if t == nil {
		panic("status.Finish called with no corresponding status.Start")
	}
	onTaskFinish(ctx, t)
	t.parent.remove(t)
}

