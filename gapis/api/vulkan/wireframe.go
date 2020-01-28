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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// wireframe returns a transform that set all the graphics pipeline to be
// created with rasterization polygon mode == VK_POLYGON_MODE_LINE
func wireframe(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Wireframe")
	return transform.Transform("Wireframe", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {
		s := out.State()
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		switch cmd := cmd.(type) {
		case *VkCreateGraphicsPipelines:
			count := uint64(cmd.CreateInfoCount())
			infos := cmd.PCreateInfos().Slice(0, count, l)
			newInfos := make([]VkGraphicsPipelineCreateInfo, count)
			newRasterStateDatas := make([]api.AllocResult, count)
			for i := uint64(0); i < count; i++ {
				info := infos.Index(i).MustRead(ctx, cmd, s, nil)[0]
				rasterState := info.PRasterizationState().MustRead(ctx, cmd, s, nil)
				rasterState.SetPolygonMode(VkPolygonMode_VK_POLYGON_MODE_LINE)
				newRasterStateDatas[i] = s.AllocDataOrPanic(ctx, rasterState)
				info.SetPRasterizationState(NewVkPipelineRasterizationStateCreateInfoᶜᵖ(newRasterStateDatas[i].Ptr()))
				newInfos[i] = info
			}
			newInfosData := s.AllocDataOrPanic(ctx, newInfos)
			newCmd := cb.VkCreateGraphicsPipelines(cmd.Device(),
				cmd.PipelineCache(), cmd.CreateInfoCount(), newInfosData.Ptr(),
				cmd.PAllocator(), cmd.PPipelines(), cmd.Result()).AddRead(newInfosData.Data())
			for _, r := range newRasterStateDatas {
				newCmd.AddRead(r.Data())
			}
			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			return out.MutateAndWrite(ctx, id, newCmd)
		default:
			return out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
