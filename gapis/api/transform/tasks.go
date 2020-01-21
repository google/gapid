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

package transform

import (
	"context"
	"sort"

	"github.com/google/gapid/gapis/api"
)

type task struct {
	at   api.CmdID
	work func(ctx context.Context, w Writer)
}

var _ Transformer = (*Tasks)(nil)

// Tasks is a Transformer that calls functions when the specified command is
// reached or passed.
type Tasks struct {
	tasks  []task
	sorted bool
}

// Add adds the job to be invoked when the command with the specified id is
// reached or passed.
func (t *Tasks) Add(at api.CmdID, work func(context.Context, Writer)) {
	t.tasks = append(t.tasks, task{at, work})
	t.sorted = false
}

func (t *Tasks) sort() {
	if !t.sorted {
		sort.Slice(t.tasks, func(i, j int) bool { return t.tasks[i].at < t.tasks[j].at })
		t.sorted = true
	}
}

func (t *Tasks) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) error {
	if id.IsReal() {
		t.sort()
		for len(t.tasks) > 0 && t.tasks[0].at < id {
			t.tasks[0].work(ctx, out)
			t.tasks = t.tasks[1:]
		}
	}
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *Tasks) Flush(ctx context.Context, out Writer) error {
	t.sort()
	for _, task := range t.tasks {
		task.work(ctx, out)
	}
	t.tasks = nil
	return nil
}

func (t *Tasks) PreLoop(ctx context.Context, output Writer)  {}
func (t *Tasks) PostLoop(ctx context.Context, output Writer) {}
func (t *Tasks) BuffersCommands() bool                       { return false }
