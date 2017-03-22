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

package worker

import (
	"context"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/job"
)

// Worker is the representation of a live object that can perform actions.
type Worker struct {
	lock sync.Mutex
	// Info is the persistable information about a worker.
	Info    *job.Worker
	handler event.Handler
	tasks   chan taskEntry
}

// Workers is the non persisted list of live workers.
type Workers struct {
	jobs    job.Manager
	entries []*Worker
	op      job.Operation
}

// Input is the interface that must be implemented by anything that wants to be
// the input to an action.
type Input interface {
	proto.Message
}

// Task is the interface to something that can be sent to a worker.
// It should always contain the inputs to the action, and the id of the action to attach results to.
type Task interface {
	proto.Message
	// Init is called as you add input and target device pairs to the manager.
	// It is invoked with the action id to post results to, the input the manager was given, and
	// the worker that the task is going to be posted to.
	Init(id string, input Input, w *job.Worker)
}

type taskEntry struct {
	task   Task
	result chan error
}

func (w *Workers) init(ctx context.Context, jobs job.Manager, op job.Operation) {
	w.jobs = jobs
	w.op = op
}

// Register is called to add a new worker to the active set.
// It takes the host and target device for the worker, which may be the same, and a handler that will
// be passed all the task objects that are sent to the worker.
func (w *Workers) Register(ctx context.Context, host *device.Instance, target *device.Instance, handler interface{}) error {
	info, err := w.jobs.GetWorker(ctx, host, target, w.op)
	if err != nil {
		return err
	}
	entry := &Worker{
		Info:    info,
		handler: event.AsHandler(ctx, handler),
		tasks:   make(chan taskEntry),
	}
	go entry.run(ctx)
	w.entries = append(w.entries, entry)
	return nil
}

// Find searches the live worker set to find the one that is managing a given target device.
func (w *Workers) Find(ctx context.Context, device string) *Worker {
	for _, entry := range w.entries {
		if entry.Info.Target == device {
			return entry
		}
	}
	return nil
}

func (w *Worker) run(ctx context.Context) {
	for e := range w.tasks {
		e.result <- w.handler(ctx, e.task)
		close(e.result)
	}
}

func (w *Worker) send(ctx context.Context, task Task) error {
	if w == nil {
		return log.Err(ctx, nil, "No worker for device")
	}
	result := make(chan error)
	w.lock.Lock()
	if w.tasks != nil {
		w.tasks <- taskEntry{task: task, result: result}
	}
	w.lock.Unlock()
	err := <-result
	if err != nil {
		w.lock.Lock()
		if w.tasks != nil {
			close(w.tasks)
			w.tasks = nil
		}
		w.lock.Unlock()
	}
	return err
}
