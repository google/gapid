// Copyright (C) 2019 Google Inc.
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

// simplifySampling returns a transform that turns off anisotropic filtering
// and sets filtering modes to nearest neighbor
func simplifySampling(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Simplify sampling")
	return transform.Transform("Simplify sampling", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {

		s := out.State()
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		switch cmd := cmd.(type) {
		case *VkCreateSampler:
			cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

			samplerCreateInfo := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
			samplerCreateInfo.SetAnisotropyEnable(VkBool32(0))
			samplerCreateInfo.SetMinFilter(VkFilter_VK_FILTER_NEAREST)
			samplerCreateInfo.SetMagFilter(VkFilter_VK_FILTER_NEAREST)

			samplerCreateInfoData := s.AllocDataOrPanic(ctx, samplerCreateInfo)
			defer samplerCreateInfoData.Free()

			newCmd := cb.VkCreateSampler(cmd.Device(),
				samplerCreateInfoData.Ptr(),
				cmd.PAllocator(),
				cmd.PSampler(),
				VkResult_VK_SUCCESS,
			).AddRead(samplerCreateInfoData.Data())

			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			return out.MutateAndWrite(ctx, id, newCmd)
		default:
			return out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
