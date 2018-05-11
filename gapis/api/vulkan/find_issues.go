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

package vulkan

import (
	"context"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

// findIssues is a command transform that detects issues when replaying the
// stream of commands. Any issues that are found are written to all the chans in
// the slice out. Once the last issue is sent (if any) all the chans in out are
// closed.
// NOTE: right now this transform is just used to close chans passed in requests.
type findIssues struct {
	state  *api.GlobalState
	issues []replay.Issue
	res    []replay.Result
}

func newFindIssues(ctx context.Context, c *capture.Capture, oldState *api.GlobalState) *findIssues {
	t := &findIssues{
		state: c.NewUninitializedStateWithAllocator(ctx, oldState.Allocator),
	}
	t.state.OnError = func(err interface{}) {
		if issue, ok := err.(replay.Issue); ok {
			t.issues = append(t.issues, issue)
		}
	}
	return t
}

// reportTo adds r to the list of issue listeners.
func (t *findIssues) reportTo(r replay.Result) { t.res = append(t.res, r) }

func (t *findIssues) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	ctx = log.Enter(ctx, "findIssues")
	mutateErr := cmd.Mutate(ctx, id, t.state, nil /* no builder */)
	if mutateErr != nil {
		// Ignore since downstream transform layers can only consume valid commands
		return
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *findIssues) Flush(ctx context.Context, out transform.Writer) {
	cb := CommandBuilder{Thread: 0, Arena: t.state.Arena}
	out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		// Since the PostBack function is called before the replay target has actually arrived at the post command,
		// we need to actually write some data here. r.Uint32() is what actually waits for the replay target to have
		// posted the data in question. If we did not do this, we would shut-down the replay as soon as the second-to-last
		// Post had occurred, which may not be anywhere near the end of the stream.
		code := uint32(0xe11de11d)
		b.Push(value.U32(code))
		b.Post(b.Buffer(1), 4, func(r binary.Reader, err error) {
			if err != nil {
				t.res = nil
				log.E(ctx, "Flush did not get expected EOS code: '%v'", err)
				return
			}
			if r.Uint32() != code {
				log.E(ctx, "Flush did not get expected EOS code")
				return
			}
			for _, res := range t.res {
				res(t.issues, nil)
			}
		})
		return nil
	}))
}
