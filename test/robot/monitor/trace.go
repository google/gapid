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

package monitor

import (
	"context"

	"github.com/google/gapid/test/robot/job/worker"
	"github.com/google/gapid/test/robot/trace"
)

// Trace is the in memory representation/wrapper for a trace.Action
type Trace struct {
	trace.Action
}

// Traces is the type that manages a set of Trace objects.
type Traces struct {
	entries []*Trace
}

// All returns the complete set of Trace objects we have seen so far.
func (t *Traces) All() []*Trace {
	return t.entries
}

// MatchPackage returns the set of Trace objects that were traced with a specific package.
func (t *Traces) MatchPackage(p *Package) []*Trace {
	result := []*Trace{}
	for _, trace := range t.entries {
		if trace.Input.Package == p.Id {
			result = append(result, trace)
		}
	}
	return result
}

func (o *DataOwner) updateTrace(ctx context.Context, action *trace.Action) error {
	o.Write(func(data *Data) {
		entry, _ := data.Traces.FindOrCreate(ctx, action)
		entry.Action = *action
	})
	return nil
}

// Find searches the traces for the one that matches the supplied action.
// See worker.EquivalentAction for more information about how actions are compared.
func (t *Traces) Find(ctx context.Context, action *trace.Action) *Trace {
	for _, entry := range t.entries {
		if worker.EquivalentAction(&entry.Action, action) {
			return entry
		}
	}
	return nil
}

// FindOrCreate returns the trace that matches the supplied aciton if it exists, if not
// it creates a new trace object, and returns it.
// It does not register the newly created trace object for you, that will happen only if
// a call is made to trigger the action on the trace service.
func (t *Traces) FindOrCreate(ctx context.Context, action *trace.Action) (*Trace, bool) {
	entry := t.Find(ctx, action)
	if entry != nil {
		return entry, true
	}
	entry = &Trace{Action: *action}
	t.entries = append(t.entries, entry)
	return entry, false
}
