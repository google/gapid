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

package scheduler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/event/task"
)

var lastTaskID uint32

// Executor is the executor of Executables.
// The executor can only work on one list of Executables at a time.
type Executor func(context.Context, *status.Replay, []Executable, Batch)

// Executable holds a task and it's result.
type Executable struct {
	Task      Task        // The work to be done.
	Cancelled task.Signal // Has this work been cancelled?
	Result    Result      // The result callback.
}

// Result is the result of an executed Task.
type Result func(val interface{}, err error)

// Task represents a single unit or work.
type Task interface{}

// Batch describes the batching rules for a scheduled Task.
type Batch struct {
	// Precondition that needs to be satisfied before this batch will be
	// executed.
	// Can be a Time or Duration to indicate a time delay for batching.
	// Can be a chan that is satisfied when closed.
	// Can be nil for no precondition.
	Precondition interface{}

	// Key is used to batch together Tasks with the same key.
	Key interface{}

	// Priority is used to prioritize batches.
	// The larger numbers represent higher priorities.
	Priority int
}

// Scheduler schedules Tasks to Executors, batching where possible.
type Scheduler struct {
	device   id.ID
	pending  chan *job
	exec     Executor
	queueLen uint32
}

// New returns a new Scheduler that will execute Tasks with exec.
func New(ctx context.Context, device id.ID, exec Executor) *Scheduler {
	s := &Scheduler{device: device, exec: exec, pending: make(chan *job, 32)}
	crash.Go(func() { s.run(ctx) })
	return s
}

// NumTasksQueued returns the number of queued tasks.
func (s *Scheduler) NumTasksQueued() int { return int(s.queueLen) }

// Schedule schedules t to be executed on s. Tasks with compatible batches may
// be executed together.
func (s *Scheduler) Schedule(ctx context.Context, t Task, b Batch) (val interface{}, err error) {
	type res struct {
		val interface{}
		err error
	}

	out := make(chan res, 1)
	c := task.ShouldStop(ctx)
	r := func(val interface{}, err error) { out <- res{val, err} }

	select {
	case s.pending <- &job{executable: Executable{t, c, r}, batch: b}:
	case <-c: // cancelled
		return nil, task.StopReason(ctx)
	}

	select {
	case r := <-out:
		return r.val, r.err
	case <-c: // cancelled
		return nil, task.StopReason(ctx)
	}
}

func (s *Scheduler) run(ctx context.Context) {
	ctx = status.StartBackground(ctx, "Replay Scheduler")
	defer status.Finish(ctx)

	bins := map[Batch]*bin{}
	var binLock sync.RWMutex

	const (
		caseShouldStop = iota
		casePending
		casePreconditions
	)

	interrupts := make([]reflect.SelectCase, casePreconditions, 100)
	interrupts[caseShouldStop] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(task.ShouldStop(ctx)),
	}
	interrupts[casePending] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(s.pending),
	}

	addJob := func(j *job) {
		binLock.Lock()
		defer binLock.Unlock()

		if b, ok := bins[j.batch]; ok {
			b.jobs = append(b.jobs, j)
		} else {
			interrupt := reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: preconditionChan(j.batch.Precondition),
			}
			bins[j.batch] = &bin{
				batch:     j.batch,
				jobs:      []*job{j},
				interrupt: interrupt,
				status:    status.ReplayQueued(ctx, atomic.AddUint32(&lastTaskID, 1), s.device),
			}
			interrupts = append(interrupts, interrupt)
		}
		atomic.AddUint32(&s.queueLen, 1)
	}

	readyChan := make(chan *bin, 32)
	crash.Go(func() {
		for {
			// Check if we should stop.
			select {
			case <-readyChan: // pro-actively drain the ready chan.
			case <-task.ShouldStop(ctx):
				return
			default:
			}

			// Find the highest priority bin to execute.
			binLock.RLock()
			var best *bin
			for _, b := range bins {
				if b.isReady() {
					if best == nil || best.batch.Priority < b.batch.Priority {
						best = b
					}
				}
			}
			binLock.RUnlock()

			// If no bin is ready, wait for one on the ready channel.
			if best == nil {
				select {
				case <-readyChan:
					continue
				case <-task.ShouldStop(ctx):
					return
				}
			}

			binLock.Lock()
			delete(bins, best.batch)
			binLock.Unlock()

			// Execute the batch.
			best.exec(ctx, s.exec)
			atomic.AddUint32(&s.queueLen, -uint32(len(best.jobs)))
		}
	})

	for !task.Stopped(ctx) {
		i, v, ok := reflect.Select(interrupts)
		switch i {
		case caseShouldStop: // <-task.ShouldStop(ctx)
			return
		case casePending: // j := <-s.pending:
			j := v.Interface().(*job)
			// TODO: Check whether the task was already scheduled.
			// If so, adjust priorites to the min, execute once and broadcast
			// results.
			addJob(j)
		default: // precondition
			binLock.RLock()
			for _, b := range bins {
				if b.interrupt == interrupts[i] {
					if ok {
						// Received a value on an open chan.
						// Once the predicate has passed, it must always pass.
						b.interrupt.Chan = reflect.ValueOf(task.FiredSignal)
					}
					readyChan <- b
				}
			}

			// Rebuild interrupts.
			interrupts = interrupts[:casePreconditions]
			for _, b := range bins {
				if !b.isReady() {
					interrupts = append(interrupts, b.interrupt)
				}
			}
			binLock.RUnlock()
		}
	}
}

func preconditionChan(p interface{}) reflect.Value {
	switch v := p.(type) {
	case nil:
		p = task.FiredSignal
	case time.Time:
		p = time.After(v.Sub(time.Now()))
	case time.Duration:
		p = time.After(v)
	}
	v := reflect.ValueOf(p)
	if v.Kind() != reflect.Chan {
		panic(fmt.Errorf("Precondition must be a Time, Duration or chan. Got %T", p))
	}
	return v
}

type bin struct {
	batch     Batch
	jobs      []*job
	interrupt reflect.SelectCase
	status    *status.Replay
}

// isReady returns true if the bin is ready to be executed.
func (b *bin) isReady() bool {
	i, _, ok := reflect.Select([]reflect.SelectCase{
		b.interrupt,
		reflect.SelectCase{Dir: reflect.SelectDefault},
	})
	if ok {
		// Received a value on the open chan.
		// Once the predicate has passes, it must always pass.
		b.interrupt.Chan = reflect.ValueOf(task.FiredSignal)
	}
	return i == 0
}

func (b *bin) exec(ctx context.Context, exec Executor) {
	l := make([]Executable, 0, len(b.jobs))
	for _, j := range b.jobs {
		if !j.executable.Cancelled.Fired() {
			l = append(l, j.executable)
		}
	}
	b.status.Start(ctx)
	exec(ctx, b.status, l, b.batch)
	b.status.Finish(ctx)
}

type job struct {
	mutex      sync.Mutex
	executable Executable
	batch      Batch
}
