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
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

// Actions is a struct to manage a persistent set of actions.
type Actions struct {
	mu         sync.Mutex
	ledger     record.Ledger
	onChange   event.Broadcast
	entries    []Action
	byID       map[string]Action
	nullAction Action
	nullTask   Task
}

// Action is the interface to the specific action type for a given of the Actions store.
type Action interface {
	proto.Message
	// Init is called as you add input and target device pairs to the manager.
	// It is invoked with the action id to use, the input the manager was given, and
	// the worker that the task is going to be posted to.
	Init(id string, input Input, w *job.Worker)
	// JobID must return the id that was handed to Init.
	JobID() string
	// JobHost must return the host of the worker that was handed to Init.
	JobHost() string
	// JobTarget must return the target of the worker that was handed to Init.
	JobTarget() string
	// JobInput must return the input that was handed to Init.
	JobInput() Input
}

func (a *Actions) init(ctx context.Context, library record.Library, op job.Operation, nullAction Action, nullTask Task) error {
	ctx = log.V{"operation": op}.Bind(ctx)
	a.nullAction = nullAction
	a.nullTask = nullTask
	a.byID = map[string]Action{}
	name := strings.ToLower(op.String())
	ledger, err := library.Open(ctx, name+"-actions", nullAction)
	if err != nil {
		return err
	}
	a.ledger = ledger
	apply := event.AsHandler(ctx, a.apply)
	if err := ledger.Read(ctx, apply); err != nil {
		return err
	}
	ledger.Watch(ctx, apply)
	return nil
}

// apply is called with items coming out of the ledger
// it should be called with the mutation lock already held.
func (a *Actions) apply(ctx context.Context, item Action) error {
	id := item.JobID()
	entry := a.byID[id]
	if entry == nil {
		entry = item
		a.entries = append(a.entries, entry)
		a.byID[id] = entry
	} else {
		proto.Merge(entry, item)
	}
	a.onChange.Send(ctx, entry)
	return nil
}

func (a *Actions) do(ctx context.Context, w *Worker, input Input) (string, error) {
	action, err := func() (Action, error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		action := proto.Clone(a.nullAction).(Action)
		action.Init(id.Unique().String(), input, w.Info)
		err := a.ledger.Add(ctx, action)
		if err != nil {
			return nil, err
		}
		return action, nil
	}()
	if err != nil {
		return "", err
	}
	task := proto.Clone(a.nullTask).(Task)
	task.Init(action.JobID(), input, w.Info)
	if err := w.send(ctx, task); err != nil {
		return "", err
	}
	return action.JobID(), nil
}

// update an action, and return the merged action.
func (a *Actions) update(ctx context.Context, action Action) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ledger.Add(ctx, action)
}

// ByID returns the action with the specified id
func (a *Actions) ByID(ctx context.Context, id string) Action {
	return a.byID[id]
}

// ByInput finds the action that matches the given input
func (a *Actions) ByInput(ctx context.Context, input proto.Message) Action {
	for _, action := range a.entries {
		if proto.Equal(action.JobInput(), input) {
			return action
		}
	}
	return nil
}

// Search runs the query for each entry in the action list, and hands the matches to the action handler.
func (a *Actions) Search(ctx context.Context, query *search.Query, handler interface{}) error {
	filter := eval.Filter(ctx, query, reflect.TypeOf(a.nullAction), event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, a.entries)
	if query.Monitor {
		return event.Monitor(ctx, &a.mu, a.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

// EquivalentAction returns true if an action is the same task being performed on the same devices.
func EquivalentAction(a, b Action) bool {
	if !proto.Equal(a.JobInput(), b.JobInput()) {
		return false
	}
	if a.JobHost() != b.JobHost() {
		return false
	}
	if a.JobTarget() != b.JobTarget() {
		return false
	}
	return true
}
