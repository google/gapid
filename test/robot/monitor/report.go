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
	"github.com/google/gapid/test/robot/report"
)

// Report is the in memory representation/wrapper for a report.Action
type Report struct {
	report.Action
}

// Reports is the type that manages a set of Report objects.
type Reports struct {
	entries []*Report
}

// All returns the complete set of Report objects we have seen so far.
func (r *Reports) All() []*Report {
	return r.entries
}

func (o *DataOwner) updateReport(ctx context.Context, action *report.Action) error {
	o.Write(func(data *Data) {
		entry, _ := data.Reports.FindOrCreate(ctx, action)
		entry.Action = *action
	})
	return nil
}

// Find searches the reports for the one that matches the supplied action.
// See worker.EquivalentAction for more information about how actions are compared.
func (r *Reports) Find(ctx context.Context, action *report.Action) *Report {
	for _, entry := range r.entries {
		if worker.EquivalentAction(&entry.Action, action) {
			return entry
		}
	}
	return nil
}

// FindOrCreate returns the report that matches the supplied aciton if it exists, if not
// it creates a new report object, and returns it.
// It does not register the newly created report object for you, that will happen only if
// a call is made to trigger the action on the report service.
func (r *Reports) FindOrCreate(ctx context.Context, action *report.Action) (*Report, bool) {
	entry := r.Find(ctx, action)
	if entry != nil {
		return entry, true
	}
	entry = &Report{Action: *action}
	r.entries = append(r.entries, entry)
	return entry, false
}
