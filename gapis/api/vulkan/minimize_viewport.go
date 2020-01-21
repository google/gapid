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

// minimizeViewport returns a transform that sets viewport sizes to 1x1
func minimizeViewport(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "Minimize viewport")
	return transform.Transform("Minimize viewport", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {

		const width = 1
		const height = 1

		s := out.State()
		l := s.MemoryLayout
		a := s.Arena
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: a}

		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		switch cmd := cmd.(type) {
		case *VkCreateGraphicsPipelines:
			createInfoCount := uint64(cmd.CreateInfoCount())
			createInfos := cmd.PCreateInfos().Slice(0, createInfoCount, l).MustRead(ctx, cmd, s, nil)
			viewportStateCreateInfosData := make([]api.AllocResult, createInfoCount)
			viewportDatas := make([]api.AllocResult, 0)
			scissorDatas := make([]api.AllocResult, 0)

			for i := uint64(0); i < createInfoCount; i++ {
				viewportStateCreateInfo := createInfos[i].PViewportState().MustRead(ctx, cmd, s, nil)

				viewportCount := uint64(viewportStateCreateInfo.ViewportCount())
				oldViewports := viewportStateCreateInfo.PViewports().Slice(0, viewportCount, l)
				newViewports := make([]VkViewport, viewportCount)
				for j := uint64(0); j < viewportCount; j++ {
					viewport := oldViewports.Index(j).MustRead(ctx, cmd, s, nil)[0]
					viewport.SetWidth(width)
					viewport.SetHeight(height)
					newViewports[j] = viewport
				}

				viewportDatas := append(viewportDatas, s.AllocDataOrPanic(ctx, newViewports))
				defer viewportDatas[i].Free()

				scissorCount := uint64(viewportStateCreateInfo.ScissorCount())
				oldScissors := viewportStateCreateInfo.PScissors().Slice(0, scissorCount, l)
				newScissors := make([]VkRect2D, scissorCount)
				for j := uint64(0); j < scissorCount; j++ {
					scissor := oldScissors.Index(j).MustRead(ctx, cmd, s, nil)[0]
					scissor.SetOffset(NewVkOffset2D(a, 0, 0))
					scissor.SetExtent(NewVkExtent2D(a, width, height))
					newScissors[j] = scissor
				}

				scissorDatas := append(scissorDatas, s.AllocDataOrPanic(ctx, newScissors))
				defer scissorDatas[i].Free()

				viewportStateCreateInfo.SetPViewports(NewVkViewportᶜᵖ(viewportDatas[i].Ptr()))
				viewportStateCreateInfo.SetPScissors(NewVkRect2Dᶜᵖ(scissorDatas[i].Ptr()))
				viewportStateCreateInfosData[i] = s.AllocDataOrPanic(ctx, viewportStateCreateInfo)
				defer viewportStateCreateInfosData[i].Free()

				createInfos[i].SetPViewportState(NewVkPipelineViewportStateCreateInfoᶜᵖ(viewportStateCreateInfosData[i].Ptr()))
			}

			createInfosData := s.AllocDataOrPanic(ctx, createInfos)
			defer createInfosData.Free()

			newCmd := cb.VkCreateGraphicsPipelines(
				cmd.Device(),
				cmd.PipelineCache(),
				cmd.CreateInfoCount(),
				createInfosData.Ptr(),
				cmd.PAllocator(),
				cmd.PPipelines(),
				cmd.Result(),
			).AddRead(
				createInfosData.Data(),
			)

			for _, vd := range viewportDatas {
				newCmd.AddRead(vd.Data())
			}
			for _, sd := range scissorDatas {
				newCmd.AddRead(sd.Data())
			}
			for _, vps := range viewportStateCreateInfosData {
				newCmd.AddRead(vps.Data())
			}
			for _, r := range cmd.Extras().Observations().Reads {
				newCmd.AddRead(r.Range, r.ID)
			}
			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}

			out.MutateAndWrite(ctx, id, newCmd)
		case *VkCmdBeginRenderPass:
			beginInfo := cmd.PRenderPassBegin().MustRead(ctx, cmd, s, nil)

			beginInfo.SetRenderArea(NewVkRect2D(a,
				NewVkOffset2D(a, 0, 0),
				NewVkExtent2D(a, width, height),
			))

			beginInfoData := s.AllocDataOrPanic(ctx, beginInfo)
			defer beginInfoData.Free()

			newCmd := cb.VkCmdBeginRenderPass(cmd.commandBuffer,
				beginInfoData.Ptr(),
				cmd.Contents(),
			).AddRead(beginInfoData.Data())

			out.MutateAndWrite(ctx, id, newCmd)
		case *VkCmdSetViewport:
			viewportCount := uint64(cmd.viewportCount)
			oldViewports := cmd.PViewports().Slice(0, viewportCount, l)
			newViewports := make([]VkViewport, viewportCount)

			for i := uint64(0); i < viewportCount; i++ {
				viewport := oldViewports.Index(i).MustRead(ctx, cmd, s, nil)[0]
				viewport.SetWidth(width)
				viewport.SetHeight(height)
				newViewports[i] = viewport
			}

			newViewportDatas := s.AllocDataOrPanic(ctx, newViewports)
			defer newViewportDatas.Free()

			newCmd := cb.VkCmdSetViewport(cmd.commandBuffer,
				cmd.FirstViewport(),
				uint32(viewportCount),
				newViewportDatas.Ptr()).AddRead(newViewportDatas.Data())

			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			out.MutateAndWrite(ctx, id, newCmd)
		case *VkCmdSetScissor:
			scissorCount := uint64(cmd.scissorCount)
			newScissors := make([]VkRect2D, scissorCount)

			for i := uint64(0); i < scissorCount; i++ {
				newScissors[i] = NewVkRect2D(a,
					NewVkOffset2D(a, 0, 0),
					NewVkExtent2D(a, width, height),
				)
			}

			newScissorsData := s.AllocDataOrPanic(ctx, newScissors)
			defer newScissorsData.Free()

			newCmd := cb.VkCmdSetScissor(cmd.commandBuffer,
				cmd.FirstScissor(),
				uint32(scissorCount),
				newScissorsData.Ptr(),
			).AddRead(newScissorsData.Data())

			out.MutateAndWrite(ctx, id, newCmd)
		default:
			out.MutateAndWrite(ctx, id, cmd)
		}
		return nil
	})
}
