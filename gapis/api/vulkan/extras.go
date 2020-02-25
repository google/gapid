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

type SemaphoreState struct {
	semaphores []VkSemaphore
	values     []uint64
}

func init() {
	protoconv.Register(
		func(ctx context.Context, o *SemaphoreState) (*vulkan_pb.SemaphoreState, error) {
			ss := &vulkan_pb.SemaphoreState{
				Semaphores: []uint64{},
				Values:     []uint64{},
			}
			for i := 0; i < len(o.semaphores); i++ {
				ss.Semaphores = append(ss.Semaphores, uint64(o.semaphores[i]))
			}
			ss.Values = append(ss.Values, o.values...)
			return ss, nil
		}, func(ctx context.Context, p *vulkan_pb.SemaphoreState) (*SemaphoreState, error) {
			ss := &SemaphoreState{
				[]VkSemaphore{},
				[]uint64{},
			}
			for i := 0; i < len(p.Semaphores); i++ {
				ss.semaphores = append(ss.semaphores, VkSemaphore(p.Semaphores[i]))
			}
			ss.values = append(ss.values, p.Values...)
			return ss, nil
		})
}

func findSemaphoreState(extras *api.CmdExtras) *SemaphoreState {
	for _, e := range extras.All() {
		if res, ok := e.(*SemaphoreState); ok {
			return res
		}
	}
	return nil
}
