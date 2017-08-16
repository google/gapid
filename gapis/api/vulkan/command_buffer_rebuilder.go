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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

// unpackMap takes a dense map of u32 -> structure, flattens the map into
// a slice, allocates the appropriate data and returns it as well as the
// lenth of the map
func unpackMap(ctx context.Context, s *api.State, m interface{}) (api.AllocResult, uint32) {
	u32Type := reflect.TypeOf(uint32(0))
	t := reflect.TypeOf(m)
	if t.Kind() != reflect.Map || t.Key() != u32Type {
		panic("Expecting a map of u32 -> structures")
	}

	mv := reflect.ValueOf(m)

	sl := reflect.MakeSlice(reflect.SliceOf(t.Elem()), mv.Len(), mv.Len())
	for i := 0; i < mv.Len(); i++ {
		v := mv.MapIndex(reflect.ValueOf(uint32(i)))
		sl.Index(i).Set(v)
	}
	return s.AllocDataOrPanic(ctx, sl.Interface()), uint32(mv.Len())
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
	s *api.State) (VkCommandBuffer, []api.Cmd, []func()) {

	x := make([]api.Cmd, 0)
	cleanup := make([]func(), 0)
	// DestroyResourcesAtEndOfFrame will handle this actually removing the
	// command buffer. We have no way to handle WHEN this will be done

	modelCmdBufObj := GetState(s).CommandBuffers[modelCmdBuf]

	newCmdBufId := VkCommandBuffer(
		newUnusedID(true,
			func(x uint64) bool {
				_, ok := GetState(s).CommandBuffers[VkCommandBuffer(x)]
				return ok
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

func rebuildCmdBeginRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBeginRenderPassData) (func(), api.Cmd) {

	clearValues := make([]VkClearValue, len(d.ClearValues))
	for i := 0; i < len(d.ClearValues); i++ {
		clearValues[i] = d.ClearValues[uint32(i)]
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
			d.Contents).AddRead(beginData.Data()).AddRead(clearValuesData.Data())
}

func rebuildCmdEndRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdEndRenderPassData) (func(), api.Cmd) {
	return func() {}, cb.VkCmdEndRenderPass(commandBuffer)
}

func rebuildCmdNextSubpass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdNextSubpassData) (func(), api.Cmd) {
	return func() {}, cb.VkCmdNextSubpass(commandBuffer, d.Contents)
}

func rebuildCmdBindPipeline(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBindPipelineData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdBindPipeline(commandBuffer,
		d.PipelineBindPoint, d.Pipeline)
}

func rebuildCmdBindIndexBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBindIndexBufferData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdBindIndexBuffer(commandBuffer,
		d.Buffer, d.Offset, d.IndexType)
}

func rebuildCmdSetLineWidth(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetLineWidthData) (func(), api.Cmd) {
	return func() {}, cb.VkCmdSetLineWidth(commandBuffer, d.LineWidth)

}

func rebuildCmdBindDescriptorSets(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBindDescriptorSetsData) (func(), api.Cmd) {

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
		).AddRead(descriptorSetData.Data())
}

func rebuildCmdBindVertexBuffers(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBindVertexBuffersData) (func(), api.Cmd) {

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
		).AddRead(offsetData.Data()).AddRead(bufferData.Data())
}

func rebuildCmdWaitEvents(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdWaitEventsData) (func(), api.Cmd) {

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
		).AddRead(eventData.Data()).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data())
}

func rebuildCmdPipelineBarrier(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdPipelineBarrierData) (func(), api.Cmd) {

	memoryBarrierData, memoryBarrierCount := unpackMap(ctx, s, d.MemoryBarriers)
	bufferMemoryBarrierData, bufferMemoryBarrierCount := unpackMap(ctx, s, d.BufferMemoryBarriers)
	imageMemoryBarrierData, imageMemoryBarrierCount := unpackMap(ctx, s, d.ImageMemoryBarriers)

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
		).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data())
}

func rebuildCmdBeginQuery(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBeginQueryData) (func(), api.Cmd) {
	return func() {}, cb.VkCmdBeginQuery(commandBuffer, d.QueryPool,
		d.Query, d.Flags)
}

func rebuildCmdBlitImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdBlitImageData) (func(), api.Cmd) {

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
		).AddRead(blitData.Data())
}

func rebuildCmdClearAttachments(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdClearAttachmentsData) (func(), api.Cmd) {

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
		).AddRead(clearAttachmentData.Data()).AddRead(rectData.Data())
}

func rebuildCmdClearColorImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdClearColorImageData) (func(), api.Cmd) {

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
		).AddRead(colorData.Data()).AddRead(rangeData.Data())
}

func rebuildCmdClearDepthStencilImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdClearDepthStencilImageData) (func(), api.Cmd) {

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
		).AddRead(depthStencilData.Data()).AddRead(rangeData.Data())
}

func rebuildCmdCopyBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdCopyBufferData) (func(), api.Cmd) {

	regionData, regionCount := unpackMap(ctx, s, d.CopyRegions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBuffer(commandBuffer,
			d.SrcBuffer,
			d.DstBuffer,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data())
}

func rebuildCmdCopyBufferToImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCopyBufferToImageData) (func(), api.Cmd) {

	regionData, regionCount := unpackMap(ctx, s, d.Regions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBufferToImage(commandBuffer,
			d.SrcBuffer,
			d.DstImage,
			d.Layout,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data())
}

func rebuildCmdCopyImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdCopyImageData) (func(), api.Cmd) {

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
		).AddRead(regionData.Data())
}

func rebuildCmdCopyImageToBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCopyImageToBufferData) (func(), api.Cmd) {

	regionData, regionCount := unpackMap(ctx, s, d.Regions)

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyImageToBuffer(commandBuffer,
			d.SrcImage,
			d.SrcImageLayout,
			d.DstBuffer,
			regionCount,
			regionData.Ptr(),
		).AddRead(regionData.Data())
}

func rebuildCmdCopyQueryPoolResults(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdCopyQueryPoolResultsData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdCopyQueryPoolResults(commandBuffer,
		d.QueryPool,
		d.FirstQuery,
		d.QueryCount,
		d.DstBuffer,
		d.DstOffset,
		d.Stride,
		d.Flags,
	)
}

func rebuildCmdDispatch(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDispatchData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDispatch(commandBuffer,
		d.GroupCountX,
		d.GroupCountY,
		d.GroupCountZ,
	)
}

func rebuildCmdDispatchIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDispatchIndirectData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDispatchIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
	)
}

func rebuildCmdDraw(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDrawData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDraw(commandBuffer,
		d.VertexCount,
		d.InstanceCount,
		d.FirstVertex,
		d.FirstInstance,
	)
}

func rebuildCmdDrawIndexed(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDrawIndexedData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDrawIndexed(commandBuffer, d.IndexCount,
		d.InstanceCount, d.FirstIndex, d.VertexOffset, d.FirstInstance)
}

func rebuildCmdDrawIndexedIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDrawIndexedIndirectData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDrawIndexedIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
		d.DrawCount,
		d.Stride,
	)
}

func rebuildCmdDrawIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdDrawIndirectData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdDrawIndirect(commandBuffer,
		d.Buffer,
		d.Offset,
		d.DrawCount,
		d.Stride,
	)
}

func rebuildCmdEndQuery(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdEndQueryData) (func(), api.Cmd) {

	return func() {}, cb.VkCmdEndQuery(commandBuffer,
		d.QueryPool,
		d.Query,
	)
}

func rebuildCmdExecuteCommands(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdExecuteCommandsData) (func(), api.Cmd) {

	commandBufferData, commandBufferCount := unpackMap(ctx, s, d.CommandBuffers)

	return func() {
			commandBufferData.Free()
		}, cb.VkCmdExecuteCommands(commandBuffer,
			commandBufferCount,
			commandBufferData.Ptr(),
		).AddRead(commandBufferData.Data())
}

func rebuildCmdFillBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdFillBufferData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdFillBuffer(commandBuffer,
			d.Buffer,
			d.DstOffset,
			d.Size,
			d.Data,
		)
}

func rebuildCmdPushConstants(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdPushConstantsDataExpanded) (func(), api.Cmd) {

	data := s.AllocDataOrPanic(ctx, d.Data)

	return func() {
			data.Free()
		}, cb.VkCmdPushConstants(commandBuffer,
			d.Layout,
			d.StageFlags,
			d.Offset,
			d.Size,
			data.Ptr(),
		).AddRead(data.Data())
}

func rebuildCmdResetQueryPool(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdResetQueryPoolData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdResetQueryPool(commandBuffer,
			d.QueryPool,
			d.FirstQuery,
			d.QueryCount,
		)
}

func rebuildCmdResolveImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdResolveImageData) (func(), api.Cmd) {

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
		).AddRead(resolveData.Data())
}

func rebuildCmdSetBlendConstants(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetBlendConstantsData) (func(), api.Cmd) {

	var constants F32ː4ᵃ
	constants[0] = d.R
	constants[1] = d.G
	constants[2] = d.B
	constants[3] = d.A

	return func() {
		}, cb.VkCmdSetBlendConstants(commandBuffer,
			constants,
		)
}

func rebuildCmdSetDepthBias(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetDepthBiasData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetDepthBias(commandBuffer,
			d.DepthBiasConstantFactor,
			d.DepthBiasClamp,
			d.DepthBiasSlopeFactor,
		)
}

func rebuildCmdSetDepthBounds(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetDepthBoundsData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetDepthBounds(commandBuffer,
			d.MinDepthBounds,
			d.MaxDepthBounds,
		)
}

func rebuildCmdSetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetEventData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetEvent(commandBuffer,
			d.Event,
			d.StageMask,
		)
}

func rebuildCmdResetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdResetEventData) (func(), api.Cmd) {
	return func() {
		}, cb.VkCmdResetEvent(commandBuffer,
			d.Event,
			d.StageMask,
		)
}

func rebuildCmdSetScissor(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetScissorData) (func(), api.Cmd) {

	scissorData, scissorCount := unpackMap(ctx, s, d.Scissors)

	return func() {
			scissorData.Free()
		}, cb.VkCmdSetScissor(commandBuffer,
			d.FirstScissor,
			scissorCount,
			scissorData.Ptr(),
		).AddRead(scissorData.Data())
}

func rebuildCmdSetStencilCompareMask(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetStencilCompareMaskData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetStencilCompareMask(commandBuffer,
			d.FaceMask,
			d.CompareMask,
		)
}

func rebuildCmdSetStencilReference(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetStencilReferenceData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetStencilReference(commandBuffer,
			d.FaceMask,
			d.Reference,
		)
}

func rebuildCmdSetStencilWriteMask(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetStencilWriteMaskData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdSetStencilWriteMask(commandBuffer,
			d.FaceMask,
			d.WriteMask,
		)
}

func rebuildCmdSetViewport(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdSetViewportData) (func(), api.Cmd) {

	viewportData, viewportCount := unpackMap(ctx, s, d.Viewports)

	return func() {
			viewportData.Free()
		}, cb.VkCmdSetViewport(commandBuffer,
			d.FirstViewport,
			viewportCount,
			viewportData.Ptr(),
		).AddRead(viewportData.Data())
}

func rebuildCmdUpdateBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdUpdateBufferData) (func(), api.Cmd) {

	data := s.AllocDataOrPanic(ctx, d.Data)

	return func() {
			data.Free()
		}, cb.VkCmdUpdateBuffer(commandBuffer,
			d.DstBuffer,
			d.DstOffset,
			d.DataSize,
			data.Ptr(),
		).AddRead(data.Data())
}

func rebuildCmdWriteTimestamp(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	d *RecreateCmdWriteTimestampData) (func(), api.Cmd) {

	return func() {
		}, cb.VkCmdWriteTimestamp(commandBuffer,
			d.PipelineStage,
			d.QueryPool,
			d.Query,
		)
}

// AddCommand recreates the command defined by recreateInfo and places it
// into the given command buffer. It returns the atoms that it
// had to create in order to satisfy the command. It also returns a function
// to clean up the data that was allocated during the creation.
func AddCommand(ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	s *api.State,
	rebuildInfo interface{}) (func(), api.Cmd) {
	switch t := rebuildInfo.(type) {
	case *RecreateCmdBeginRenderPassData:
		return rebuildCmdBeginRenderPass(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdEndRenderPassData:
		return rebuildCmdEndRenderPass(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdNextSubpassData:
		return rebuildCmdNextSubpass(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBindPipelineData:
		return rebuildCmdBindPipeline(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBindDescriptorSetsData:
		return rebuildCmdBindDescriptorSets(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBindVertexBuffersData:
		return rebuildCmdBindVertexBuffers(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBindIndexBufferData:
		return rebuildCmdBindIndexBuffer(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdPipelineBarrierData:
		return rebuildCmdPipelineBarrier(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdWaitEventsData:
		return rebuildCmdWaitEvents(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBeginQueryData:
		return rebuildCmdBeginQuery(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdBlitImageData:
		return rebuildCmdBlitImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdClearAttachmentsData:
		return rebuildCmdClearAttachments(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdClearColorImageData:
		return rebuildCmdClearColorImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdClearDepthStencilImageData:
		return rebuildCmdClearDepthStencilImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdCopyBufferData:
		return rebuildCmdCopyBuffer(ctx, cb, commandBuffer, s, t)
	case *RecreateCopyBufferToImageData:
		return rebuildCmdCopyBufferToImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdCopyImageData:
		return rebuildCmdCopyImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCopyImageToBufferData:
		return rebuildCmdCopyImageToBuffer(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdCopyQueryPoolResultsData:
		return rebuildCmdCopyQueryPoolResults(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDispatchData:
		return rebuildCmdDispatch(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDispatchIndirectData:
		return rebuildCmdDispatchIndirect(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDrawData:
		return rebuildCmdDraw(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDrawIndexedData:
		return rebuildCmdDrawIndexed(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDrawIndexedIndirectData:
		return rebuildCmdDrawIndexedIndirect(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdDrawIndirectData:
		return rebuildCmdDrawIndirect(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdEndQueryData:
		return rebuildCmdEndQuery(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdExecuteCommandsData:
		return rebuildCmdExecuteCommands(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdFillBufferData:
		return rebuildCmdFillBuffer(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdPushConstantsDataExpanded:
		return rebuildCmdPushConstants(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdResetQueryPoolData:
		return rebuildCmdResetQueryPool(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdResolveImageData:
		return rebuildCmdResolveImage(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetBlendConstantsData:
		return rebuildCmdSetBlendConstants(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetDepthBiasData:
		return rebuildCmdSetDepthBias(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetDepthBoundsData:
		return rebuildCmdSetDepthBounds(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetEventData:
		return rebuildCmdSetEvent(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdResetEventData:
		return rebuildCmdResetEvent(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetLineWidthData:
		return rebuildCmdSetLineWidth(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetScissorData:
		return rebuildCmdSetScissor(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetStencilCompareMaskData:
		return rebuildCmdSetStencilCompareMask(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetStencilReferenceData:
		return rebuildCmdSetStencilReference(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetStencilWriteMaskData:
		return rebuildCmdSetStencilWriteMask(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdSetViewportData:
		return rebuildCmdSetViewport(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdUpdateBufferData:
		return rebuildCmdUpdateBuffer(ctx, cb, commandBuffer, s, t)
	case *RecreateCmdWriteTimestampData:
		return rebuildCmdWriteTimestamp(ctx, cb, commandBuffer, s, t)
	default:
		x := fmt.Sprintf("Should not reach here: %T", t)
		panic(x)
	}
}
