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

package replay

import (
	"context"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

// EndOfReplay is a transform that causes a post back at the end of the replay. It can be used to
// ensure that GAPIS waits for a replay to finish, if there are no postbacks, or no pending
// postbacks in the replay.
type EndOfReplay struct {
	res []Result
}

// AddResult adds the given replay result listener to this transform.
func (t *EndOfReplay) AddResult(r Result) {
	t.res = append(t.res, r)
}

func (t *EndOfReplay) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *EndOfReplay) Flush(ctx context.Context, out transform.Writer) error {
	t.AddNotifyInstruction(ctx, out, func() interface{} { return nil })
	return nil
}

func (t *EndOfReplay) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *EndOfReplay) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *EndOfReplay) BuffersCommands() bool                              { return false }

// AddNotifyInstruction adds the instruction to the replay stream that will notify GAPIS that the
// replay has finished. It should be the end (or very near the end) of the replay stream and thus
// be called from a transform Flush.
func (t *EndOfReplay) AddNotifyInstruction(ctx context.Context, out transform.Writer, result func() interface{}) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{T: 0, F: func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		// Since the PostBack function is called before the replay target has actually arrived at the post command,
		// we need to actually write some data here. r.Uint32() is what actually waits for the replay target to have
		// posted the data in question. If we did not do this, we would shut-down the replay as soon as the second-to-last
		// Post had occurred, which may not be anywhere near the end of the stream.
		code := uint32(0x6060fa51)
		b.Push(value.U32(code))
		b.Post(b.Buffer(1), 4, func(r binary.Reader, err error) {
			for _, res := range t.res {
				res.Do(func() (interface{}, error) {
					if err != nil {
						return nil, log.Err(ctx, err, "Flush did not get expected EOS code: '%v'")
					}
					if r.Uint32() != code {
						return nil, log.Err(ctx, nil, "Flush did not get expected EOS code")
					}
					return result(), nil
				})
			}
		})
		return nil
	}})
}
