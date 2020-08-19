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

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/vulkan/vulkan_pb"
)

type FenceState struct {
	fences   []VkFence
	statuses []uint32
}

func init() {
	protoconv.Register(
		func(ctx context.Context, o *FenceState) (*vulkan_pb.FenceState, error) {
			fs := &vulkan_pb.FenceState{
				Fences:   []uint64{},
				Statuses: []uint32{},
			}
			for i := 0; i < len(o.fences); i++ {
				fs.Fences = append(fs.Fences, uint64(o.fences[i]))
			}
			fs.Statuses = append(fs.Statuses, o.statuses...)
			return fs, nil
		}, func(ctx context.Context, p *vulkan_pb.FenceState) (*FenceState, error) {
			fs := &FenceState{
				[]VkFence{},
				[]uint32{},
			}
			for i := 0; i < len(p.Fences); i++ {
				fs.fences = append(fs.fences, VkFence(p.Fences[i]))
			}
			fs.statuses = append(fs.statuses, p.Statuses...)
			return fs, nil
		})
}

func findFenceState(extras *api.CmdExtras) *FenceState {
	for _, e := range extras.All() {
		if res, ok := e.(*FenceState); ok {
			return res
		}
	}
	return nil
}
