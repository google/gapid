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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

type sliceCommandMapper struct {
	subCommandIndicesMap *map[api.CommandSubmissionKey][]uint64
	submissionCount      uint64
}

func (t *sliceCommandMapper) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	ctx = log.Enter(ctx, "Slice Command Mapper")
	s := out.State()

	switch cmd := cmd.(type) {
	case *VkQueueSubmit:
		if id.IsReal() {
			submitInfoCount := cmd.SubmitCount()
			submitInfos := cmd.pSubmits.Slice(0, uint64(submitInfoCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
			for i := uint32(0); i < submitInfoCount; i++ {
				si := submitInfos[i]
				cmdBufferCount := si.CommandBufferCount()
				cmdBuffers := si.PCommandBuffers().Slice(0, uint64(cmdBufferCount), s.MemoryLayout).MustRead(ctx, cmd, s, nil)
				for j := uint32(0); j < cmdBufferCount; j++ {
					commandIndex := []uint64{uint64(id), uint64(i), uint64(j), 0}
					key := api.CommandSubmissionKey{SubmissionOrder: t.submissionCount, CommandBuffer: int64(cmdBuffers[j])}
					(*t.subCommandIndicesMap)[key] = commandIndex
				}
			}
			t.submissionCount++
		}
		return out.MutateAndWrite(ctx, id, cmd)
	default:
		return out.MutateAndWrite(ctx, id, cmd)
	}
	return nil
}

func (t *sliceCommandMapper) PreLoop(ctx context.Context, out transform.Writer) {
	out.NotifyPreLoop(ctx)
}
func (t *sliceCommandMapper) PostLoop(ctx context.Context, out transform.Writer) {
	out.NotifyPostLoop(ctx)
}
func (t *sliceCommandMapper) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *sliceCommandMapper) BuffersCommands() bool {
	return false
}

func newSliceCommandMapper(subCommandIndicesMap *map[api.CommandSubmissionKey][]uint64) *sliceCommandMapper {
	return &sliceCommandMapper{subCommandIndicesMap: subCommandIndicesMap, submissionCount: 0}
}
