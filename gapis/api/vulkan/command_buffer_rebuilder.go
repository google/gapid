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
	"fmt"
	"reflect"

	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

// unpackMap takes a dense map of u32 -> structure, flattens the map into
// a slice, allocates the appropriate data and returns it as well as the
// length of the map.
func unpackMap(ctx context.Context, s *api.GlobalState, m interface{}) (api.AllocResult, uint32) {
	u32Type := reflect.TypeOf(uint32(0))
	d := dictionary.From(m)
	if d == nil || d.KeyTy() != u32Type {
		msg := fmt.Sprintf("Expecing a map of u32 -> structures: got %T", m)
		panic(msg)
	}

	sl := reflect.MakeSlice(reflect.SliceOf(d.ValTy()), d.Len(), d.Len())
	for _, e := range d.Entries() {
		i := e.K.(uint32)
		v := reflect.ValueOf(e.V)
		sl.Index(int(i)).Set(v)
	}
	return s.AllocDataOrPanic(ctx, sl.Interface()), uint32(d.Len())
}

// allocateNewCmdBufFromExistingOneAndBegin takes an existing VkCommandBuffer
// and allocate then begin a new one with the same allocation/inheritance and
// begin info. It returns the new allocated and began VkCommandBuffer, the new
// commands added to roll out the allocation and command buffer begin, and the
// clean up functions to recycle the data.
func allocateNewCmdBufFromExistingOneAndBegin(
	ctx context.Context,
	cb CommandBuilder,
	modelCmdBuf VkCommandBuffer,
	s *api.GlobalState) (VkCommandBuffer, []api.Cmd, []func()) {

	x := make([]api.Cmd, 0)
	cleanup := make([]func(), 0)
	// DestroyResourcesAtEndOfFrame will handle this actually removing the
	// command buffer. We have no way to handle WHEN this will be done

	modelCmdBufObj := GetState(s).CommandBuffers.Get(modelCmdBuf)

	newCmdBufId := VkCommandBuffer(
		newUnusedID(true,
			func(x uint64) bool {
				return GetState(s).CommandBuffers.Contains(VkCommandBuffer(x))
			}))
	allocate := VkCommandBufferAllocateInfo{
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		modelCmdBufObj.Pool,
		modelCmdBufObj.Level,
		uint32(1),
	}
	allocateData := s.AllocDataOrPanic(ctx, allocate)
	cleanup = append(cleanup, func() { allocateData.Free() })

	newCmdBufData := s.AllocDataOrPanic(ctx, newCmdBufId)
	cleanup = append(cleanup, func() { newCmdBufData.Free() })

	x = append(x,
		cb.VkAllocateCommandBuffers(modelCmdBufObj.Device,
			allocateData.Ptr(), newCmdBufData.Ptr(), VkResult_VK_SUCCESS,
		).AddRead(allocateData.Data()).AddWrite(newCmdBufData.Data()))

	beginInfo := VkCommandBufferBeginInfo{
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr),
	}
	if modelCmdBufObj.BeginInfo.Inherited {
		inheritanceInfo := VkCommandBufferInheritanceInfo{
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_INHERITANCE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			modelCmdBufObj.BeginInfo.InheritedRenderPass,
			modelCmdBufObj.BeginInfo.InheritedSubpass,
			modelCmdBufObj.BeginInfo.InheritedFramebuffer,
			modelCmdBufObj.BeginInfo.InheritedOcclusionQuery,
			modelCmdBufObj.BeginInfo.InheritedQueryFlags,
			modelCmdBufObj.BeginInfo.InheritedPipelineStatsFlags,
		}
		inheritanceInfoData := s.AllocDataOrPanic(ctx, inheritanceInfo)
		cleanup = append(cleanup, func() { inheritanceInfoData.Free() })
		beginInfo.PInheritanceInfo = NewVkCommandBufferInheritanceInfoᶜᵖ(inheritanceInfoData.Ptr())
	}
	beginInfoData := s.AllocDataOrPanic(ctx, beginInfo)
	cleanup = append(cleanup, func() { beginInfoData.Free() })
	x = append(x,
		cb.VkBeginCommandBuffer(newCmdBufId, beginInfoData.Ptr(), VkResult_VK_SUCCESS).AddRead(beginInfoData.Data()))
	return newCmdBufId, x, cleanup
}

func rebuildVkCmdBeginRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBeginRenderPassArgs) (func(), api.Cmd, error) {
	if !GetState(s).RenderPasses.Contains(d.RenderPass) {
		return nil, nil, fmt.Errorf("Cannot find Renderpass %v", d.RenderPass)
	}
	if !GetState(s).Framebuffers.Contains(d.Framebuffer) {
		return nil, nil, fmt.Errorf("Cannot find Framebuffer %v", d.Framebuffer)
	}

	clearValues := make([]VkClearValue, d.ClearValues.Len())
	for i := 0; i < d.ClearValues.Len(); i++ {
		clearValues[i] = d.ClearValues.Get(uint32(i))
	}

	clearValuesData := s.AllocDataOrPanic(ctx, clearValues)

	begin := VkRenderPassBeginInfo{
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		d.RenderPass,
		d.Framebuffer,
		d.RenderArea,
		uint32(len(clearValues)),
		NewVkClearValueᶜᵖ(clearValuesData.Ptr()),
	}
	beginData := s.AllocDataOrPanic(ctx, begin)

	return func() {
			clearValuesData.Free()
			beginData.Free()
		}, cb.VkCmdBeginRenderPass(
			commandBuffer,
			beginData.Ptr(),
			d.Contents).AddRead(beginData.Data()).AddRead(clearValuesData.Data()), nil
}

func rebuildVkCmdEndRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdEndRenderPassArgs) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdEndRenderPass(commandBuffer), nil
}

func rebuildVkCmdNextSubpass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdNextSubpassArgs) (func(), api.Cmd, error) {
	return func() {}, cb.VkCmdNextSubpass(commandBuffer, d.Contents), nil
}

func rebuildVkCmdBindPipeline(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBindPipelineArgs) (func(), api.Cmd, error) {
	if !GetState(s).ComputePipelines.Contains(d.Pipeline) &&
		!GetState(s).GraphicsPipelines.Contains(d.Pipeline) {
		return nil, nil, fmt.Errorf("Cannot find Pipeline %v", d.Pipeline)
	}
	return func() {}, cb.VkCmdBindPipeline(commandBuffer,
		d.PipelineBindPoint, d.Pipeline), nil
}

func rebuildVkCmdBindIndexBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBindIndexBufferArgs) (func(), api.Cmd, error) {
	if !GetState(s).Buffers.Contains(d.Buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer)
	}
	return func() {}, cb.VkCmdBindIndexBuffer(commandBuffer,
		d.Buffer, d.Offset, d.IndexType), nil
}

func rebuildVkCmdSetLineWidth(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetLineWidthArgs) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdSetLineWidth(commandBuffer, d.LineWidth), nil
}

func rebuildVkCmdBindDescriptorSets(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBindDescriptorSetsArgs) (func(), api.Cmd, error) {

	for i := uint32(0); i < uint32(len(*d.DescriptorSets.Map)); i++ {
		if !GetState(s).DescriptorSets.Contains(d.DescriptorSets.Get(i)) {
			return nil, nil, fmt.Errorf("Cannot find DescriptorSet %v", d.DescriptorSets.Get(i))
		}
	}

	descriptorSetData, descriptorSetCount := unpackMap(ctx, s, d.DescriptorSets)
	dynamicOffsetData, dynamicOffsetCount := unpackMap(ctx, s, d.DynamicOffsets)

	return func() {
			descriptorSetData.Free()
			dynamicOffsetData.Free()
		}, cb.VkCmdBindDescriptorSets(commandBuffer,
			d.PipelineBindPoint,
			d.Layout,
			d.FirstSet,
			descriptorSetCount,
			descriptorSetData.Ptr(),
			dynamicOffsetCount,
			dynamicOffsetData.Ptr(),
		).AddRead(
			dynamicOffsetData.Data(),
		).AddRead(descriptorSetData.Data()), nil
}

func rebuildVkCmdBindVertexBuffers(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBindVertexBuffersArgs) (func(), api.Cmd, error) {

	for i := uint32(0); i < uint32(len(*d.Buffers.Map)); i++ {
		if !GetState(s).Buffers.Contains(d.Buffers.Get(i)) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffers.Get(i))
		}
	}

	bufferData, _ := unpackMap(ctx, s, d.Buffers)
	offsetData, _ := unpackMap(ctx, s, d.Offsets)

	return func() {
			bufferData.Free()
			offsetData.Free()
		}, cb.VkCmdBindVertexBuffers(commandBuffer,
			d.FirstBinding,
			d.BindingCount,
			bufferData.Ptr(),
			offsetData.Ptr(),
		).AddRead(offsetData.Data()).AddRead(bufferData.Data()), nil
}

func rebuildVkCmdWaitEvents(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdWaitEventsArgs) (func(), api.Cmd, error) {

	for i := uint32(0); i < uint32(len(*d.Events.Map)); i++ {
		if !GetState(s).Events.Contains(d.Events.Get(i)) {
			return nil, nil, fmt.Errorf("Cannot find Event %v", d.Events.Get(i))
		}
	}

	for i := uint32(0); i < uint32(len(*d.BufferMemoryBarriers.Map)); i++ {
		if !GetState(s).Buffers.Contains(d.BufferMemoryBarriers.Get(i).Buffer) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.BufferMemoryBarriers.Get(i).Buffer)
		}
	}

	for i := uint32(0); i < uint32(len(*d.ImageMemoryBarriers.Map)); i++ {
		if !GetState(s).Images.Contains(d.ImageMemoryBarriers.Get(i).Image) {
			return nil, nil, fmt.Errorf("Cannot find Event %v", d.ImageMemoryBarriers.Get(i).Image)
		}
	}

	eventData, eventCount := unpackMap(ctx, s, d.Events)
	memoryBarrierData, memoryBarrierCount := unpackMap(ctx, s, d.MemoryBarriers)
	bufferMemoryBarrierData, bufferMemoryBarrierCount := unpackMap(ctx, s, d.BufferMemoryBarriers)
	imageMemoryBarrierData, imageMemoryBarrierCount := unpackMap(ctx, s, d.ImageMemoryBarriers)

	return func() {
			eventData.Free()
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, cb.VkCmdWaitEvents(commandBuffer,
			eventCount,
			eventData.Ptr(),
			d.SrcStageMask,
			d.DstStageMask,
			memoryBarrierCount,
			memoryBarrierData.Ptr(),
			bufferMemoryBarrierCount,
			bufferMemoryBarrierData.Ptr(),
			imageMemoryBarrierCount,
			imageMemoryBarrierData.Ptr(),
		).AddRead(eventData.Data()).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data()), nil
}

func rebuildVkCmdPipelineBarrier(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdPipelineBarrierArgs) (func(), api.Cmd, error) {

	memoryBarrierData, memoryBarrierCount := unpackMap(ctx, s, d.MemoryBarriers)
	bufferMemoryBarrierData, bufferMemoryBarrierCount := unpackMap(ctx, s, d.BufferMemoryBarriers)
	imageMemoryBarrierData, imageMemoryBarrierCount := unpackMap(ctx, s, d.ImageMemoryBarriers)

	for i := uint32(0); i < uint32(len(*d.BufferMemoryBarriers.Map)); i++ {
		if !GetState(s).Buffers.Contains(d.BufferMemoryBarriers.Get(i).Buffer) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.BufferMemoryBarriers.Get(i).Buffer)
		}
	}

	for i := uint32(0); i < uint32(len(*d.ImageMemoryBarriers.Map)); i++ {
		if !GetState(s).Images.Contains(d.ImageMemoryBarriers.Get(i).Image) {
			return nil, nil, fmt.Errorf("Cannot find Image %v", d.ImageMemoryBarriers.Get(i).Image)
		}
	}

	return func() {
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, cb.VkCmdPipelineBarrier(commandBuffer,
			d.SrcStageMask,
			d.DstStageMask,
			d.DependencyFlags,
			memoryBarrierCount,
			memoryBarrierData.Ptr(),
			bufferMemoryBarrierCount,
			bufferMemoryBarrierData.Ptr(),
			imageMemoryBarrierCount,
			imageMemoryBarrierData.Ptr(),
		).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data()), nil
}

func rebuildVkCmdBeginQuery(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBeginQueryArgs) (func(), api.Cmd, error) {

	if !GetState(s).QueryPools.Contains(d.QueryPool) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool)
	}

	return func() {}, cb.VkCmdBeginQuery(commandBuffer, d.QueryPool,
		d.Query, d.Flags), nil
}

func rebuildVkCmdBlitImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdBlitImageArgs) (func(), api.Cmd, error) {

	if !GetState(s).Images.Contains(d.SrcImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage)
	}

	if !GetState(s).Images.Contains(d.DstImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage)
	}

	blitData, blitCount := unpackMap(ctx, s, d.Regions)

	return func() {
			blitData.Free()
		}, cb.VkCmdBlitImage(commandBuffer,
			d.SrcImage,
			d.SrcImageLayout,
			d.DstImage,
			d.DstImageLayout,
			blitCount,
			blitData.Ptr(),
			d.Filter,
		).AddRead(blitData.Data()), nil
}

func rebuildVkCmdClearAttachments(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdClearAttachmentsArgs) (func(), api.Cmd, error) {

	clearAttachmentData, clearCount := unpackMap(ctx, s, d.Attachments)
	rectData, rectCount := unpackMap(ctx, s, d.Rects)

	return func() {
			clearAttachmentData.Free()
			rectData.Free()
		}, cb.VkCmdClearAttachments(commandBuffer,
			clearCount,
			clearAttachmentData.Ptr(),
			rectCount,
			rectData.Ptr(),
		).AddRead(clearAttachmentData.Data()).AddRead(rectData.Data()), nil
}

func rebuildVkCmdClearColorImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdClearColorImageArgs) (func(), api.Cmd, error) {

	if !GetState(s).Images.Contains(d.Image) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.Image)
	}

	colorData := s.AllocDataOrPanic(ctx, d.Color)

	rangeData, rangeCount := unpackMap(ctx, s, d.Ranges)

	return func() {
			colorData.Free()
			rangeData.Free()
		}, cb.VkCmdClearColorImage(commandBuffer,
			d.Image,
			d.ImageLayout,
			colorData.Ptr(),
			rangeCount,
			rangeData.Ptr(),
		).AddRead(colorData.Data()).AddRead(rangeData.Data()), nil
}

func rebuildVkCmdClearDepthStencilImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdClearDepthStencilImageArgs) (func(), api.Cmd, error) {

	if !GetState(s).Images.Contains(d.Image) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.Image)
	}

	depthStencilData := s.AllocDataOrPanic(ctx, d.DepthStencil)

	rangeData, rangeCount := unpackMap(ctx, s, d.Ranges)

	return func() {
			depthStencilData.Free()
			rangeData.Free()
		}, cb.VkCmdClearDepthStencilImage(commandBuffer,
			d.Image,
			d.ImageLayout,
			depthStencilData.Ptr(),
			rangeCount,
			rangeData.Ptr(),
		).AddRead(depthStencilData.Data()).AddRead(rangeData.Data()), nil
}

func rebuildVkCmdCopyBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdCopyBufferArgs) (func(), api.Cmd, error) {

	if !GetState(s).Buffers.Contains(d.SrcBuffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.SrcBuffer)
	}
	if !GetState(s).Buffers.Contains(d.DstBuffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer)
	}

	regionData, regionCount := unpackMap(ctx, s, d.CopyRegions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBuffer(commandBuffer,
			d.SrcBuffer,
			d.DstBuffer,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data()), nil
}

func rebuildVkCmdCopyBufferToImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdCopyBufferToImageArgs) (func(), api.Cmd, error) {
	if !GetState(s).Buffers.Contains(d.SrcBuffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.SrcBuffer)
	}
	if !GetState(s).Images.Contains(d.DstImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage)
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBufferToImage(commandBuffer,
			d.SrcBuffer,
			d.DstImage,
			d.Layout,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data()), nil
}

func rebuildVkCmdCopyImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdCopyImageArgs) (func(), api.Cmd, error) {
	if !GetState(s).Images.Contains(d.SrcImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage)
	}
	if !GetState(s).Images.Contains(d.DstImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage)
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyImage(commandBuffer,
			d.SrcImage,
			d.SrcImageLayout,
			d.DstImage,
			d.DstImageLayout,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data()), nil
}

func rebuildVkCmdCopyImageToBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdCopyImageToBufferArgs) (func(), api.Cmd, error) {
	if !GetState(s).Images.Contains(d.SrcImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage)
	}
	if !GetState(s).Buffers.Contains(d.DstBuffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer)
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyImageToBuffer(commandBuffer,
			d.SrcImage,
			d.SrcImageLayout,
			d.DstBuffer,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data()), nil
}

func rebuildVkCmdCopyQueryPoolResults(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdCopyQueryPoolResultsArgs) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools.Contains(d.QueryPool) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool)
	}
	if !GetState(s).Buffers.Contains(d.DstBuffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer)
	}
	return func() {}, cb.VkCmdCopyQueryPoolResults(commandBuffer,
		d.QueryPool,
		d.FirstQuery,
		d.QueryCount,
		d.DstBuffer,
		d.DstOffset,
		d.Stride,
		d.Flags,
	), nil
}

func rebuildVkCmdDispatch(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDispatchArgs) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDispatch(commandBuffer,
		d.GroupCountX,
		d.GroupCountY,
		d.GroupCountZ,
	), nil
}

func rebuildVkCmdDispatchIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDispatchIndirectArgs) (func(), api.Cmd, error) {

	if !GetState(s).Buffers.Contains(d.Buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer)
	}
	return func() {}, cb.VkCmdDispatchIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
	), nil
}

func rebuildVkCmdDraw(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDrawArgs) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDraw(commandBuffer,
		d.VertexCount,
		d.InstanceCount,
		d.FirstVertex,
		d.FirstInstance,
	), nil
}

func rebuildVkCmdDrawIndexed(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDrawIndexedArgs) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDrawIndexed(commandBuffer, d.IndexCount,
		d.InstanceCount, d.FirstIndex, d.VertexOffset, d.FirstInstance), nil
}

func rebuildVkCmdDrawIndexedIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDrawIndexedIndirectArgs) (func(), api.Cmd, error) {
	if !GetState(s).Buffers.Contains(d.Buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer)
	}
	return func() {}, cb.VkCmdDrawIndexedIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
		d.DrawCount,
		d.Stride,
	), nil
}

func rebuildVkCmdDrawIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDrawIndirectArgs) (func(), api.Cmd, error) {

	if !GetState(s).Buffers.Contains(d.Buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer)
	}
	return func() {}, cb.VkCmdDrawIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
		d.DrawCount,
		d.Stride,
	), nil
}

func rebuildVkCmdEndQuery(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdEndQueryArgs) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools.Contains(d.QueryPool) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool)
	}
	return func() {}, cb.VkCmdEndQuery(commandBuffer,
		d.QueryPool,
		d.Query,
	), nil
}

func rebuildVkCmdExecuteCommands(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdExecuteCommandsArgs) (func(), api.Cmd, error) {

	for i := uint32(0); i < uint32(len(*d.CommandBuffers.Map)); i++ {
		if !GetState(s).CommandBuffers.Contains(d.CommandBuffers.Get(i)) {
			return nil, nil, fmt.Errorf("Cannot find CommandBuffer %v", d.CommandBuffers.Get(i))
		}
	}

	commandBufferData, commandBufferCount := unpackMap(ctx, s, d.CommandBuffers)

	return func() {
			commandBufferData.Free()
		}, cb.VkCmdExecuteCommands(commandBuffer,
			commandBufferCount,
			commandBufferData.Ptr(),
		).AddRead(commandBufferData.Data()), nil
}

func rebuildVkCmdFillBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdFillBufferArgs) (func(), api.Cmd, error) {
	if !GetState(s).Buffers.Contains(d.Buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer)
	}
	return func() {
		}, cb.VkCmdFillBuffer(commandBuffer,
			d.Buffer,
			d.DstOffset,
			d.Size,
			d.Data,
		), nil
}

func rebuildVkCmdPushConstants(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdPushConstantsArgs) (func(), api.Cmd, error) {

	dat := d.Data.MustRead(ctx, nil, r, nil)
	data := s.AllocDataOrPanic(ctx, dat)

	return func() {
			data.Free()
		}, cb.VkCmdPushConstants(commandBuffer,
			d.Layout,
			d.StageFlags,
			d.Offset,
			d.Size,
			data.Ptr(),
		).AddRead(data.Data()), nil
}

func rebuildVkCmdResetQueryPool(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdResetQueryPoolArgs) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools.Contains(d.QueryPool) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool)
	}
	return func() {
		}, cb.VkCmdResetQueryPool(commandBuffer,
			d.QueryPool,
			d.FirstQuery,
			d.QueryCount,
		), nil
}

func rebuildVkCmdResolveImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdResolveImageArgs) (func(), api.Cmd, error) {
	if !GetState(s).Images.Contains(d.SrcImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage)
	}
	if !GetState(s).Images.Contains(d.DstImage) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage)
	}
	resolveData, resolveCount := unpackMap(ctx, s, d.ResolveRegions)

	return func() {
			resolveData.Free()
		}, cb.VkCmdResolveImage(commandBuffer,
			d.SrcImage,
			d.SrcImageLayout,
			d.DstImage,
			d.DstImageLayout,
			resolveCount,
			resolveData.Ptr(),
		).AddRead(resolveData.Data()), nil
}

func rebuildVkCmdSetBlendConstants(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetBlendConstantsArgs) (func(), api.Cmd, error) {

	var constants F32ː4ᵃ
	constants[0] = d.R
	constants[1] = d.G
	constants[2] = d.B
	constants[3] = d.A

	return func() {
		}, cb.VkCmdSetBlendConstants(commandBuffer,
			constants,
		), nil
}

func rebuildVkCmdSetDepthBias(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetDepthBiasArgs) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetDepthBias(commandBuffer,
			d.DepthBiasConstantFactor,
			d.DepthBiasClamp,
			d.DepthBiasSlopeFactor,
		), nil
}

func rebuildVkCmdSetDepthBounds(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetDepthBoundsArgs) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetDepthBounds(commandBuffer,
			d.MinDepthBounds,
			d.MaxDepthBounds,
		), nil
}

func rebuildVkCmdSetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetEventArgs) (func(), api.Cmd, error) {
	if !GetState(s).Events.Contains(d.Event) {
		return nil, nil, fmt.Errorf("Cannot find Event %v", d.Event)
	}
	return func() {
		}, cb.VkCmdSetEvent(commandBuffer,
			d.Event,
			d.StageMask,
		), nil
}

func rebuildVkCmdResetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdResetEventArgs) (func(), api.Cmd, error) {
	if !GetState(s).Events.Contains(d.Event) {
		return nil, nil, fmt.Errorf("Cannot find Event %v", d.Event)
	}
	return func() {
		}, cb.VkCmdResetEvent(commandBuffer,
			d.Event,
			d.StageMask,
		), nil
}

func rebuildVkCmdSetScissor(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetScissorArgs) (func(), api.Cmd, error) {

	scissorData, scissorCount := unpackMap(ctx, s, d.Scissors)

	return func() {
			scissorData.Free()
		}, cb.VkCmdSetScissor(commandBuffer,
			d.FirstScissor,
			scissorCount,
			scissorData.Ptr(),
		).AddRead(scissorData.Data()), nil
}

func rebuildVkCmdSetStencilCompareMask(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetStencilCompareMaskArgs) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilCompareMask(commandBuffer,
			d.FaceMask,
			d.CompareMask,
		), nil
}

func rebuildVkCmdSetStencilReference(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetStencilReferenceArgs) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilReference(commandBuffer,
			d.FaceMask,
			d.Reference,
		), nil
}

func rebuildVkCmdSetStencilWriteMask(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetStencilWriteMaskArgs) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilWriteMask(commandBuffer,
			d.FaceMask,
			d.WriteMask,
		), nil
}

func rebuildVkCmdSetViewport(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdSetViewportArgs) (func(), api.Cmd, error) {

	viewportData, viewportCount := unpackMap(ctx, s, d.Viewports)

	return func() {
			viewportData.Free()
		}, cb.VkCmdSetViewport(commandBuffer,
			d.FirstViewport,
			viewportCount,
			viewportData.Ptr(),
		).AddRead(viewportData.Data()), nil
}

func rebuildVkCmdUpdateBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdUpdateBufferArgs) (func(), api.Cmd, error) {
	if !GetState(s).Buffers.Contains(d.DstBuffer) {
		return nil, nil, fmt.Errorf("Cannot find buffer %v", d.DstBuffer)
	}

	dat := d.Data.MustRead(ctx, nil, r, nil)
	data := s.AllocDataOrPanic(ctx, dat)

	return func() {
			data.Free()
		}, cb.VkCmdUpdateBuffer(commandBuffer,
			d.DstBuffer,
			d.DstOffset,
			d.DataSize,
			data.Ptr(),
		).AddRead(data.Data()), nil
}

func rebuildVkCmdWriteTimestamp(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdWriteTimestampArgs) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools.Contains(d.QueryPool) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool)
	}
	return func() {
		}, cb.VkCmdWriteTimestamp(commandBuffer,
			d.PipelineStage,
			d.QueryPool,
			d.Query,
		), nil
}

func rebuildVkCmdDebugMarkerBeginEXT(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDebugMarkerBeginEXTArgs) (func(), api.Cmd, error) {
	markerNameData := s.AllocDataOrPanic(ctx, d.MarkerName)
	var color F32ː4ᵃ
	color[0] = d.Color[0]
	color[1] = d.Color[1]
	color[2] = d.Color[2]
	color[3] = d.Color[3]
	markerInfoData := s.AllocDataOrPanic(ctx,
		VkDebugMarkerMarkerInfoEXT{
			SType:       VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
			PNext:       NewVoidᶜᵖ(memory.Nullptr),
			PMarkerName: NewCharᶜᵖ(markerNameData.Ptr()),
			Color:       color,
		})
	return func() {
			markerNameData.Free()
			markerInfoData.Free()
		}, cb.VkCmdDebugMarkerBeginEXT(commandBuffer, markerInfoData.Ptr()).AddRead(
			markerNameData.Data()).AddRead(markerInfoData.Data()), nil
}

func rebuildVkCmdDebugMarkerEndEXT(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDebugMarkerEndEXTArgs) (func(), api.Cmd, error) {
	return func() {}, cb.VkCmdDebugMarkerEndEXT(commandBuffer), nil
}

func rebuildVkCmdDebugMarkerInsertEXT(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d *VkCmdDebugMarkerInsertEXTArgs) (func(), api.Cmd, error) {
	markerNameData := s.AllocDataOrPanic(ctx, d.MarkerName)
	var color F32ː4ᵃ
	color[0] = d.Color[0]
	color[1] = d.Color[1]
	color[2] = d.Color[2]
	color[3] = d.Color[3]
	markerInfoData := s.AllocDataOrPanic(ctx,
		VkDebugMarkerMarkerInfoEXT{
			SType:       VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
			PNext:       NewVoidᶜᵖ(memory.Nullptr),
			PMarkerName: NewCharᶜᵖ(markerNameData.Ptr()),
			Color:       color,
		})
	return func() {
			markerNameData.Free()
			markerInfoData.Free()
		}, cb.VkCmdDebugMarkerInsertEXT(commandBuffer, markerInfoData.Ptr()).AddRead(
			markerNameData.Data()).AddRead(markerInfoData.Data()), nil
}

// GetCommandArgs takes a command reference and returns the command arguments
// of that recorded command.
func GetCommandArgs(ctx context.Context,
	cr CommandReference,
	s *State) interface{} {
	switch cr.Type {
	case CommandType_cmd_vkCmdBeginRenderPass:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBeginRenderPass.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdEndRenderPass:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdEndRenderPass.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdNextSubpass:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdNextSubpass.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBindPipeline:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBindPipeline.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBindDescriptorSets:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBindDescriptorSets.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBindVertexBuffers:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBindVertexBuffers.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBindIndexBuffer:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBindIndexBuffer.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdPipelineBarrier:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdPipelineBarrier.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdWaitEvents:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdWaitEvents.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBeginQuery:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBeginQuery.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdBlitImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdBlitImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdClearAttachments:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdClearAttachments.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdClearColorImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdClearColorImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdClearDepthStencilImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdClearDepthStencilImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdCopyBuffer:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdCopyBuffer.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdCopyBufferToImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdCopyBufferToImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdCopyImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdCopyImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdCopyImageToBuffer:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdCopyImageToBuffer.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdCopyQueryPoolResults:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdCopyQueryPoolResults.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDispatch:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDispatch.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDispatchIndirect:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDispatchIndirect.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDraw:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDraw.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDrawIndexed:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDrawIndexed.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDrawIndexedIndirect:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDrawIndexedIndirect.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDrawIndirect:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDrawIndirect.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdEndQuery:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdEndQuery.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdExecuteCommands:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdExecuteCommands.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdFillBuffer:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdFillBuffer.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdPushConstants:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdPushConstants.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdResetQueryPool:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdResetQueryPool.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdResolveImage:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdResolveImage.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetBlendConstants:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetBlendConstants.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetDepthBias:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetDepthBias.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetDepthBounds:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetDepthBounds.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetEvent:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetEvent.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdResetEvent:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdResetEvent.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetLineWidth:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetLineWidth.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetScissor:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetScissor.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetStencilCompareMask:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetStencilCompareMask.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetStencilReference:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetStencilReference.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetStencilWriteMask:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetStencilWriteMask.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdSetViewport:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdSetViewport.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdUpdateBuffer:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdUpdateBuffer.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdWriteTimestamp:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdWriteTimestamp.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDebugMarkerBeginEXT:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDebugMarkerBeginEXT.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDebugMarkerEndEXT:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDebugMarkerEndEXT.Get(cr.MapIndex)
	case CommandType_cmd_vkCmdDebugMarkerInsertEXT:
		return s.CommandBuffers.Get(cr.Buffer).BufferCommands.VkCmdDebugMarkerInsertEXT.Get(cr.MapIndex)
	default:
		x := fmt.Sprintf("Should not reach here: %T", cr)
		panic(x)
	}
}

// GetCommandFunction takes a command reference (recorded command buffer
// command) and returns the function which mutates the state as the recorded
// command get executed.
func GetCommandFunction(cr CommandReference) interface{} {
	switch cr.Type {
	case CommandType_cmd_vkCmdBeginRenderPass:
		return subDovkCmdBeginRenderPass
	case CommandType_cmd_vkCmdEndRenderPass:
		return subDovkCmdEndRenderPass
	case CommandType_cmd_vkCmdNextSubpass:
		return subDovkCmdNextSubpass
	case CommandType_cmd_vkCmdBindPipeline:
		return subDovkCmdBindPipeline
	case CommandType_cmd_vkCmdBindDescriptorSets:
		return subDovkCmdBindDescriptorSets
	case CommandType_cmd_vkCmdBindVertexBuffers:
		return subDovkCmdBindVertexBuffers
	case CommandType_cmd_vkCmdBindIndexBuffer:
		return subDovkCmdBindIndexBuffer
	case CommandType_cmd_vkCmdPipelineBarrier:
		return subDovkCmdPipelineBarrier
	case CommandType_cmd_vkCmdWaitEvents:
		return subDovkCmdWaitEvents
	case CommandType_cmd_vkCmdBeginQuery:
		return subDovkCmdBeginQuery
	case CommandType_cmd_vkCmdBlitImage:
		return subDovkCmdBlitImage
	case CommandType_cmd_vkCmdClearAttachments:
		return subDovkCmdClearAttachments
	case CommandType_cmd_vkCmdClearColorImage:
		return subDovkCmdClearColorImage
	case CommandType_cmd_vkCmdClearDepthStencilImage:
		return subDovkCmdClearDepthStencilImage
	case CommandType_cmd_vkCmdCopyBuffer:
		return subDovkCmdCopyBuffer
	case CommandType_cmd_vkCmdCopyBufferToImage:
		return subDovkCmdCopyBufferToImage
	case CommandType_cmd_vkCmdCopyImage:
		return subDovkCmdCopyImage
	case CommandType_cmd_vkCmdCopyImageToBuffer:
		return subDovkCmdCopyImageToBuffer
	case CommandType_cmd_vkCmdCopyQueryPoolResults:
		return subDovkCmdCopyQueryPoolResults
	case CommandType_cmd_vkCmdDispatch:
		return subDovkCmdDispatch
	case CommandType_cmd_vkCmdDispatchIndirect:
		return subDovkCmdDispatchIndirect
	case CommandType_cmd_vkCmdDraw:
		return subDovkCmdDraw
	case CommandType_cmd_vkCmdDrawIndexed:
		return subDovkCmdDrawIndexed
	case CommandType_cmd_vkCmdDrawIndexedIndirect:
		return subDovkCmdDrawIndexedIndirect
	case CommandType_cmd_vkCmdDrawIndirect:
		return subDovkCmdDrawIndirect
	case CommandType_cmd_vkCmdEndQuery:
		return subDovkCmdEndQuery
	case CommandType_cmd_vkCmdExecuteCommands:
		return subDovkCmdExecuteCommands
	case CommandType_cmd_vkCmdFillBuffer:
		return subDovkCmdFillBuffer
	case CommandType_cmd_vkCmdPushConstants:
		return subDovkCmdPushConstants
	case CommandType_cmd_vkCmdResetQueryPool:
		return subDovkCmdResetQueryPool
	case CommandType_cmd_vkCmdResolveImage:
		return subDovkCmdResolveImage
	case CommandType_cmd_vkCmdSetBlendConstants:
		return subDovkCmdSetBlendConstants
	case CommandType_cmd_vkCmdSetDepthBias:
		return subDovkCmdSetDepthBias
	case CommandType_cmd_vkCmdSetDepthBounds:
		return subDovkCmdSetDepthBounds
	case CommandType_cmd_vkCmdSetEvent:
		return subDovkCmdSetEvent
	case CommandType_cmd_vkCmdResetEvent:
		return subDovkCmdResetEvent
	case CommandType_cmd_vkCmdSetLineWidth:
		return subDovkCmdSetLineWidth
	case CommandType_cmd_vkCmdSetScissor:
		return subDovkCmdSetScissor
	case CommandType_cmd_vkCmdSetStencilCompareMask:
		return subDovkCmdSetStencilCompareMask
	case CommandType_cmd_vkCmdSetStencilReference:
		return subDovkCmdSetStencilReference
	case CommandType_cmd_vkCmdSetStencilWriteMask:
		return subDovkCmdSetStencilWriteMask
	case CommandType_cmd_vkCmdSetViewport:
		return subDovkCmdSetViewport
	case CommandType_cmd_vkCmdUpdateBuffer:
		return subDovkCmdUpdateBuffer
	case CommandType_cmd_vkCmdWriteTimestamp:
		return subDovkCmdWriteTimestamp
	case CommandType_cmd_vkCmdDebugMarkerBeginEXT:
		return subDovkCmdDebugMarkerBeginEXT
	case CommandType_cmd_vkCmdDebugMarkerEndEXT:
		return subDovkCmdDebugMarkerEndEXT
	case CommandType_cmd_vkCmdDebugMarkerInsertEXT:
		return subDovkCmdDebugMarkerInsertEXT
	default:
		x := fmt.Sprintf("Should not reach here: %T", cr)
		panic(x)
	}
}

// AddCommand recreates the command defined by recreateInfo and places it
// into the given command buffer. It returns the atoms that it
// had to create in order to satisfy the command. It also returns a function
// to clean up the data that was allocated during the creation.
func AddCommand(ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	rebuildInfo interface{}) (func(), api.Cmd, error) {
	switch t := rebuildInfo.(type) {
	case *VkCmdBeginRenderPassArgs:
		return rebuildVkCmdBeginRenderPass(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdEndRenderPassArgs:
		return rebuildVkCmdEndRenderPass(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdNextSubpassArgs:
		return rebuildVkCmdNextSubpass(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBindPipelineArgs:
		return rebuildVkCmdBindPipeline(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBindDescriptorSetsArgs:
		return rebuildVkCmdBindDescriptorSets(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBindVertexBuffersArgs:
		return rebuildVkCmdBindVertexBuffers(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBindIndexBufferArgs:
		return rebuildVkCmdBindIndexBuffer(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdPipelineBarrierArgs:
		return rebuildVkCmdPipelineBarrier(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdWaitEventsArgs:
		return rebuildVkCmdWaitEvents(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBeginQueryArgs:
		return rebuildVkCmdBeginQuery(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdBlitImageArgs:
		return rebuildVkCmdBlitImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdClearAttachmentsArgs:
		return rebuildVkCmdClearAttachments(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdClearColorImageArgs:
		return rebuildVkCmdClearColorImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdClearDepthStencilImageArgs:
		return rebuildVkCmdClearDepthStencilImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdCopyBufferArgs:
		return rebuildVkCmdCopyBuffer(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdCopyBufferToImageArgs:
		return rebuildVkCmdCopyBufferToImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdCopyImageArgs:
		return rebuildVkCmdCopyImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdCopyImageToBufferArgs:
		return rebuildVkCmdCopyImageToBuffer(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdCopyQueryPoolResultsArgs:
		return rebuildVkCmdCopyQueryPoolResults(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDispatchArgs:
		return rebuildVkCmdDispatch(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDispatchIndirectArgs:
		return rebuildVkCmdDispatchIndirect(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDrawArgs:
		return rebuildVkCmdDraw(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDrawIndexedArgs:
		return rebuildVkCmdDrawIndexed(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDrawIndexedIndirectArgs:
		return rebuildVkCmdDrawIndexedIndirect(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDrawIndirectArgs:
		return rebuildVkCmdDrawIndirect(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdEndQueryArgs:
		return rebuildVkCmdEndQuery(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdExecuteCommandsArgs:
		return rebuildVkCmdExecuteCommands(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdFillBufferArgs:
		return rebuildVkCmdFillBuffer(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdPushConstantsArgs:
		return rebuildVkCmdPushConstants(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdResetQueryPoolArgs:
		return rebuildVkCmdResetQueryPool(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdResolveImageArgs:
		return rebuildVkCmdResolveImage(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetBlendConstantsArgs:
		return rebuildVkCmdSetBlendConstants(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetDepthBiasArgs:
		return rebuildVkCmdSetDepthBias(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetDepthBoundsArgs:
		return rebuildVkCmdSetDepthBounds(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetEventArgs:
		return rebuildVkCmdSetEvent(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdResetEventArgs:
		return rebuildVkCmdResetEvent(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetLineWidthArgs:
		return rebuildVkCmdSetLineWidth(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetScissorArgs:
		return rebuildVkCmdSetScissor(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetStencilCompareMaskArgs:
		return rebuildVkCmdSetStencilCompareMask(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetStencilReferenceArgs:
		return rebuildVkCmdSetStencilReference(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetStencilWriteMaskArgs:
		return rebuildVkCmdSetStencilWriteMask(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdSetViewportArgs:
		return rebuildVkCmdSetViewport(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdUpdateBufferArgs:
		return rebuildVkCmdUpdateBuffer(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdWriteTimestampArgs:
		return rebuildVkCmdWriteTimestamp(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDebugMarkerBeginEXTArgs:
		return rebuildVkCmdDebugMarkerBeginEXT(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDebugMarkerEndEXTArgs:
		return rebuildVkCmdDebugMarkerEndEXT(ctx, cb, commandBuffer, r, s, t)
	case *VkCmdDebugMarkerInsertEXTArgs:
		return rebuildVkCmdDebugMarkerInsertEXT(ctx, cb, commandBuffer, r, s, t)

	default:
		x := fmt.Sprintf("Should not reach here: %T", t)
		panic(x)
	}
}
