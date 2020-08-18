// Copyright (C) 2020 Google Inc.
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
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
)

var _ transform2.Transform = &endOfReplay{}

// endOfReplay is a transform that causes a post back at the end of the replay. It can be used to
// ensure that GAPIS waits for a replay to finish, if there are no postbacks, or no pending
// postbacks in the replay.
type endOfReplay struct {
	res []replay.Result
}

func newEndOfReplay() *endOfReplay {
	return &endOfReplay{
		res: []replay.Result{},
	}
}

// AddResult adds the given replay result listener to this transform.
func (endTransform *endOfReplay) AddResult(r replay.Result) {
	endTransform.res = append(endTransform.res, r)
}

func (endTransform *endOfReplay) RequiresAccurateState() bool {
	return false
}

func (endTransform *endOfReplay) RequiresInnerStateMutation() bool {
	return false
}

func (endTransform *endOfReplay) SetInnerStateMutationFunction(mutator transform2.StateMutator) {
	// This transform do not require inner state mutation
}

func (endTransform *endOfReplay) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (endTransform *endOfReplay) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}

func (endTransform *endOfReplay) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return []api.Cmd{endTransform.CreateNotifyInstruction(ctx, defaultNotifyFunction)}, nil
}

func (endTransform *endOfReplay) TransformCommand(ctx context.Context, id transform2.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func defaultNotifyFunction() interface{} {
	return nil
}

// CreateNotifyInstruction creates a command that adds an instruction to the replay stream
// that will notify GAPIS that the replay has finished.
// It should be the end (or very near the end) of the replay stream and thus
// be called from an EndTransform.
func (endTransform *endOfReplay) CreateNotifyInstruction(ctx context.Context, result func() interface{}) api.Cmd {
	return replay.Custom{T: 0, F: func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		// Since the PostBack function is called before the replay target has actually arrived at the post command,
		// we need to actually write some data here. r.Uint32() is what actually waits for the replay target to have
		// posted the data in question. If we did not do this, we would shut-down the replay as soon as the second-to-last
		// Post had occurred, which may not be anywhere near the end of the stream.
		kEosCode := uint32(0x6060fa51)
		b.Push(value.U32(kEosCode))
		b.Post(b.Buffer(1), 4, func(r binary.Reader, err error) {
			for _, res := range endTransform.res {
				res.Do(func() (interface{}, error) {
					if err != nil {
						return nil, log.Err(ctx, err, "Flush did not get expected EOS code: '%v'")
					}
					if r.Uint32() != kEosCode {
						return nil, log.Err(ctx, nil, "Flush did not get expected EOS code")
					}
					return result(), nil
				})
			}
		})
		return nil
	}}
}
