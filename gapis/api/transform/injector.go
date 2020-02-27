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

	"github.com/google/gapid/gapis/api"
)

// Injector is an implementation of Transformer that can inject commands into
// the stream.
type Injector struct {
	injections map[api.CmdID][]api.Cmd
}

// Inject emits cmd after the command with identifier after.
func (t *Injector) Inject(after api.CmdID, cmd api.Cmd) {
	if t.injections == nil {
		t.injections = make(map[api.CmdID][]api.Cmd)
	}
	t.injections[after] = append(t.injections[after], cmd)
}

// Transform implements the Transformer interface.
func (t *Injector) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out Writer) error {
	if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
		return err
	}

	if r, ok := t.injections[id]; ok {
		for _, injection := range r {
			if err := out.MutateAndWrite(ctx, api.CmdNoID, injection); err != nil {
				return err
			}
		}
		delete(t.injections, id)
	}
	return nil
}

// Flush implements the Transformer interface.
func (t *Injector) Flush(ctx context.Context, out Writer) error { return nil }

func (t *Injector) PreLoop(ctx context.Context, output Writer)  {}
func (t *Injector) PostLoop(ctx context.Context, output Writer) {}
func (t *Injector) BuffersCommands() bool                       { return false }
