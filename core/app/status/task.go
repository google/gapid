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

	"github.com/google/gapid/core/log"
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
	background bool
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

// ParentID returns the ID of the parent task, or 0 if there is no parent
func (t *Task) ParentID() uint64 {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if t.parent != nil {
		return t.parent.ID()
	}
	return 0
}

// Name returns the task's name.
func (t *Task) Name() string { t.mutex.RLock(); defer t.mutex.RUnlock(); return t.name }

func (t *Task) Background() bool { t.mutex.RLock(); defer t.mutex.RUnlock(); return t.background }

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
		id:         atomic.AddUint64(&nextID, 1),
		name:       fmt.Sprintf(name, args...),
		begun:      time.Now(),
		parent:     parent,
		children:   map[*Task]struct{}{},
		background: false,
	}
	log.D(ctx, "Starting task: %s", t.name)

	parent.add(t)
	onTaskStart(ctx, t)
	return PutTask(ctx, t)
}

// StartBackground returns a new context for a long running background task.
// End() must be called with the returned context once the task has been
// finished.
func StartBackground(ctx context.Context, name string, args ...interface{}) context.Context {
	parent := GetTask(ctx)
	if parent == nil {
		parent = &app
	}
	t := &Task{
		id:         atomic.AddUint64(&nextID, 1),
		name:       fmt.Sprintf(name, args...),
		begun:      time.Now(),
		parent:     parent,
		children:   map[*Task]struct{}{},
		background: true,
	}
	log.D(ctx, "Starting task: %s", t.name)

	parent.add(t)
	onTaskStart(ctx, t)
	return PutTask(ctx, t)
}

// Do is a convenience function for calling block between Start() and Finish().
func Do(ctx context.Context, name string, block func(context.Context)) {
	ctx = Start(ctx, name)
	defer Finish(ctx)
	block(ctx)
}

// UpdateProgress updates the progress of the task started with Start().
// n is the number of units of completion, which ranges from [0, outof).
func UpdateProgress(ctx context.Context, n, outof uint64) {
	t := GetTask(ctx)
	if t == nil {
		panic("status.UpdateProgress called with no corresponding status.Start")
	}
	t.completion = float32(n) / float32(outof)
	onTaskProgress(ctx, t)
}

// Block marks the current task as blocked
func Block(ctx context.Context) {
	t := GetTask(ctx)
	if t == nil {
		panic("status.Block called with no corresponding status.Start")
	}
	onBlock(ctx, t)
}

// Block marks the current task as blocked
func Unblock(ctx context.Context) {
	t := GetTask(ctx)
	if t == nil {
		panic("status.Unblock called with no corresponding status.Start")
	}
	onUnblock(ctx, t)
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

// Event causes an event to be recorded.
// If EventScope == Task then the the currently executing task will get the scope
func Event(ctx context.Context, scope EventScope, name string, args ...interface{}) {
	t := GetTask(ctx)
	onEvent(ctx, t, scope, name, args)
}

// Traverse calls cb for all descendant tasks.
func (t *Task) Traverse(cb func(*Task)) {
	for _, c := range t.SubTasks() {
		cb(c)
		c.Traverse(cb)
	}
}
