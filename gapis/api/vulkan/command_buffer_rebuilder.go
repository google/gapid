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

// unpackMapWithAllocator takes a dense map of u32 -> structure, flattens the
// map into a slice, allocates the appropriate data using a custom provided
// allocation function and returns it as well as the length of the map.
func unpackMapWithAllocator(alloc func(v ...interface{}) api.AllocResult, m interface{}) (api.AllocResult, uint32) {
	u32Type := reflect.TypeOf(uint32(0))
	d := dictionary.From(m)
	if d == nil || d.KeyTy() != u32Type {
		msg := fmt.Sprintf("Expecting a map of u32 -> structures: got %T", m)
		panic(msg)
	}

	sl := reflect.MakeSlice(reflect.SliceOf(d.ValTy()), d.Len(), d.Len())
	for _, e := range dictionary.Entries(d) {
		i := e.K.(uint32)
		v := reflect.ValueOf(e.V)
		sl.Index(int(i)).Set(v)
	}

	return alloc(sl.Interface()), uint32(d.Len())
}

// unpackMap takes a dense map of u32 -> structure, flattens the map into
// a slice, allocates the appropriate data and returns it as well as the
// length of the map.
func unpackMap(ctx context.Context, s *api.GlobalState, m interface{}) (api.AllocResult, uint32) {
	return unpackMapWithAllocator(func(v ...interface{}) api.AllocResult {
		return s.AllocDataOrPanic(ctx, v...)
	}, m)
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

	a := s.Arena // TODO: Should this be a seperate temporary arena?

	x := make([]api.Cmd, 0)
	cleanup := make([]func(), 0)
	// DestroyResourcesAtEndOfFrame will handle this actually removing the
	// command buffer. We have no way to handle WHEN this will be done

	modelCmdBufObj := GetState(s).CommandBuffers().Get(modelCmdBuf)

	newCmdBufID := VkCommandBuffer(
		newUnusedID(true, func(x uint64) bool {
			return GetState(s).CommandBuffers().Contains(VkCommandBuffer(x))
		}))
	allocate := NewVkCommandBufferAllocateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		0, // pNext
		modelCmdBufObj.Pool(),  // commandPool
		modelCmdBufObj.Level(), // level
		1, // commandBufferCount
	)
	allocateData := s.AllocDataOrPanic(ctx, allocate)
	cleanup = append(cleanup, func() { allocateData.Free() })

	newCmdBufData := s.AllocDataOrPanic(ctx, newCmdBufID)
	cleanup = append(cleanup, func() { newCmdBufData.Free() })

	x = append(x,
		cb.VkAllocateCommandBuffers(modelCmdBufObj.Device(),
			allocateData.Ptr(), newCmdBufData.Ptr(), VkResult_VK_SUCCESS,
		).AddRead(allocateData.Data()).AddWrite(newCmdBufData.Data()))

	beginInfo := NewVkCommandBufferBeginInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr),
	)
	if bi := modelCmdBufObj.BeginInfo(); bi.Inherited() {
		inheritanceInfo := NewVkCommandBufferInheritanceInfo(a,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_INHERITANCE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			bi.InheritedRenderPass(),
			bi.InheritedSubpass(),
			bi.InheritedFramebuffer(),
			bi.InheritedOcclusionQuery(),
			bi.InheritedQueryFlags(),
			bi.InheritedPipelineStatsFlags(),
		)
		inheritanceInfoData := s.AllocDataOrPanic(ctx, inheritanceInfo)
		cleanup = append(cleanup, func() { inheritanceInfoData.Free() })
		beginInfo.SetPInheritanceInfo(NewVkCommandBufferInheritanceInfoᶜᵖ(inheritanceInfoData.Ptr()))
	}
	beginInfoData := s.AllocDataOrPanic(ctx, beginInfo)
	cleanup = append(cleanup, func() { beginInfoData.Free() })
	x = append(x,
		cb.VkBeginCommandBuffer(newCmdBufID, beginInfoData.Ptr(), VkResult_VK_SUCCESS).AddRead(beginInfoData.Data()))
	return newCmdBufID, x, cleanup
}

func rebuildVkCmdBeginRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdBeginRenderPassArgsʳ) (func(), api.Cmd, error) {

	a := s.Arena // TODO: Should this be a seperate temporary arena?

	if !GetState(s).RenderPasses().Contains(d.RenderPass()) {
		return nil, nil, fmt.Errorf("Cannot find Renderpass %v", d.RenderPass())
	}
	if !GetState(s).Framebuffers().Contains(d.Framebuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Framebuffer %v", d.Framebuffer())
	}

	clearValues := make([]VkClearValue, d.ClearValues().Len())
	for i := range clearValues {
		clearValues[i] = d.ClearValues().Get(uint32(i))
	}

	clearValuesData := s.AllocDataOrPanic(ctx, clearValues)

	begin := NewVkRenderPassBeginInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO, // sType
		0,                                        // pNext
		d.RenderPass(),                           // renderPass
		d.Framebuffer(),                          // framebuffer
		d.RenderArea(),                           // renderArea
		uint32(len(clearValues)),                 // clearValueCount
		NewVkClearValueᶜᵖ(clearValuesData.Ptr()), // pClearValues
	)
	beginData := s.AllocDataOrPanic(ctx, begin)

	return func() {
			clearValuesData.Free()
			beginData.Free()
		}, cb.VkCmdBeginRenderPass(
			commandBuffer,
			beginData.Ptr(),
			d.Contents()).AddRead(beginData.Data()).AddRead(clearValuesData.Data()), nil
}

func rebuildVkCmdEndRenderPass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdEndRenderPassArgsʳ) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdEndRenderPass(commandBuffer), nil
}

func rebuildVkCmdNextSubpass(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdNextSubpassArgsʳ) (func(), api.Cmd, error) {
	return func() {}, cb.VkCmdNextSubpass(commandBuffer, d.Contents()), nil
}

func rebuildVkCmdBindPipeline(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdBindPipelineArgsʳ) (func(), api.Cmd, error) {

	pipeline := d.Pipeline()
	if !GetState(s).ComputePipelines().Contains(pipeline) &&
		!GetState(s).GraphicsPipelines().Contains(pipeline) {
		return nil, nil, fmt.Errorf("Cannot find Pipeline %v", pipeline)
	}
	return func() {}, cb.VkCmdBindPipeline(commandBuffer,
		d.PipelineBindPoint(), pipeline), nil
}

func rebuildVkCmdBindIndexBuffer(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdBindIndexBufferArgsʳ) (func(), api.Cmd, error) {

	buffer := d.Buffer()
	if !GetState(s).Buffers().Contains(buffer) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", buffer)
	}
	return func() {}, cb.VkCmdBindIndexBuffer(commandBuffer,
		buffer, d.Offset(), d.IndexType()), nil
}

func rebuildVkCmdSetLineWidth(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetLineWidthArgsʳ) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdSetLineWidth(commandBuffer, d.LineWidth()), nil
}

func rebuildVkCmdBindDescriptorSets(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdBindDescriptorSetsArgsʳ) (func(), api.Cmd, error) {

	for i, c := 0, d.DescriptorSets().Len(); i < c; i++ {
		ds := d.DescriptorSets().Get(uint32(i))
		if !GetState(s).DescriptorSets().Contains(ds) {
			return nil, nil, fmt.Errorf("Cannot find DescriptorSet %v", ds)
		}
	}

	descriptorSetData, descriptorSetCount := unpackMap(ctx, s, d.DescriptorSets())
	dynamicOffsetData, dynamicOffsetCount := unpackMap(ctx, s, d.DynamicOffsets())

	return func() {
			descriptorSetData.Free()
			dynamicOffsetData.Free()
		}, cb.VkCmdBindDescriptorSets(commandBuffer,
			d.PipelineBindPoint(),
			d.Layout(),
			d.FirstSet(),
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
	d VkCmdBindVertexBuffersArgsʳ) (func(), api.Cmd, error) {

	for i, c := 0, d.Buffers().Len(); i < c; i++ {
		buf := d.Buffers().Get(uint32(i))
		if !GetState(s).Buffers().Contains(buf) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", buf)
		}
	}

	bufferData, _ := unpackMap(ctx, s, d.Buffers())
	offsetData, _ := unpackMap(ctx, s, d.Offsets())

	return func() {
			bufferData.Free()
			offsetData.Free()
		}, cb.VkCmdBindVertexBuffers(commandBuffer,
			d.FirstBinding(),
			d.BindingCount(),
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
	d VkCmdWaitEventsArgsʳ) (func(), api.Cmd, error) {

	for i, c := 0, d.Events().Len(); i < c; i++ {
		evt := d.Events().Get(uint32(i))
		if !GetState(s).Events().Contains(evt) {
			return nil, nil, fmt.Errorf("Cannot find Event %v", evt)
		}
	}

	for i, c := 0, d.BufferMemoryBarriers().Len(); i < c; i++ {
		buf := d.BufferMemoryBarriers().Get(uint32(i)).Buffer()
		if !GetState(s).Buffers().Contains(buf) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", buf)
		}
	}

	for i, c := 0, d.ImageMemoryBarriers().Len(); i < c; i++ {
		img := d.ImageMemoryBarriers().Get(uint32(i)).Image()
		if !GetState(s).Images().Contains(img) {
			return nil, nil, fmt.Errorf("Cannot find Event %v", img)
		}
	}

	eventData, eventCount := unpackMap(ctx, s, d.Events())
	memoryBarrierData, memoryBarrierCount := unpackMap(ctx, s, d.MemoryBarriers())
	bufferMemoryBarrierData, bufferMemoryBarrierCount := unpackMap(ctx, s, d.BufferMemoryBarriers())
	imageMemoryBarrierData, imageMemoryBarrierCount := unpackMap(ctx, s, d.ImageMemoryBarriers())

	return func() {
			eventData.Free()
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, cb.VkCmdWaitEvents(commandBuffer,
			eventCount,
			eventData.Ptr(),
			d.SrcStageMask(),
			d.DstStageMask(),
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
	d VkCmdPipelineBarrierArgsʳ) (func(), api.Cmd, error) {

	memoryBarrierData, memoryBarrierCount := unpackMap(ctx, s, d.MemoryBarriers())
	bufferMemoryBarrierData, bufferMemoryBarrierCount := unpackMap(ctx, s, d.BufferMemoryBarriers())
	imageMemoryBarrierData, imageMemoryBarrierCount := unpackMap(ctx, s, d.ImageMemoryBarriers())

	for i, c := 0, d.BufferMemoryBarriers().Len(); i < c; i++ {
		buf := d.BufferMemoryBarriers().Get(uint32(i)).Buffer()
		if !GetState(s).Buffers().Contains(buf) {
			return nil, nil, fmt.Errorf("Cannot find Buffer %v", buf)
		}
	}

	for i, c := 0, d.ImageMemoryBarriers().Len(); i < c; i++ {
		img := d.ImageMemoryBarriers().Get(uint32(i)).Image()
		if !GetState(s).Images().Contains(img) {
			return nil, nil, fmt.Errorf("Cannot find Image %v", img)
		}
	}

	return func() {
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, cb.VkCmdPipelineBarrier(commandBuffer,
			d.SrcStageMask(),
			d.DstStageMask(),
			d.DependencyFlags(),
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
	d VkCmdBeginQueryArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).QueryPools().Contains(d.QueryPool()) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool())
	}

	return func() {}, cb.VkCmdBeginQuery(commandBuffer, d.QueryPool(),
		d.Query(), d.Flags()), nil
}

func rebuildVkCmdBlitImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdBlitImageArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Images().Contains(d.SrcImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage())
	}

	if !GetState(s).Images().Contains(d.DstImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage())
	}

	blitData, blitCount := unpackMap(ctx, s, d.Regions())

	return func() {
			blitData.Free()
		}, cb.VkCmdBlitImage(commandBuffer,
			d.SrcImage(),
			d.SrcImageLayout(),
			d.DstImage(),
			d.DstImageLayout(),
			blitCount,
			blitData.Ptr(),
			d.Filter(),
		).AddRead(blitData.Data()), nil
}

func rebuildVkCmdClearAttachments(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdClearAttachmentsArgsʳ) (func(), api.Cmd, error) {

	clearAttachmentData, clearCount := unpackMap(ctx, s, d.Attachments())
	rectData, rectCount := unpackMap(ctx, s, d.Rects())

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
	d VkCmdClearColorImageArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Images().Contains(d.Image()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.Image())
	}

	colorData := s.AllocDataOrPanic(ctx, d.Color())

	rangeData, rangeCount := unpackMap(ctx, s, d.Ranges())

	return func() {
			colorData.Free()
			rangeData.Free()
		}, cb.VkCmdClearColorImage(commandBuffer,
			d.Image(),
			d.ImageLayout(),
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
	d VkCmdClearDepthStencilImageArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Images().Contains(d.Image()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.Image())
	}

	depthStencilData := s.AllocDataOrPanic(ctx, d.DepthStencil())

	rangeData, rangeCount := unpackMap(ctx, s, d.Ranges())

	return func() {
			depthStencilData.Free()
			rangeData.Free()
		}, cb.VkCmdClearDepthStencilImage(commandBuffer,
			d.Image(),
			d.ImageLayout(),
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
	d VkCmdCopyBufferArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Buffers().Contains(d.SrcBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.SrcBuffer())
	}
	if !GetState(s).Buffers().Contains(d.DstBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer())
	}

	regionData, regionCount := unpackMap(ctx, s, d.CopyRegions())

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBuffer(commandBuffer,
			d.SrcBuffer(),
			d.DstBuffer(),
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
	d VkCmdCopyBufferToImageArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Buffers().Contains(d.SrcBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.SrcBuffer())
	}
	if !GetState(s).Images().Contains(d.DstImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage())
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions())

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyBufferToImage(commandBuffer,
			d.SrcBuffer(),
			d.DstImage(),
			d.Layout(),
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
	d VkCmdCopyImageArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Images().Contains(d.SrcImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage())
	}
	if !GetState(s).Images().Contains(d.DstImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage())
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions())

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyImage(commandBuffer,
			d.SrcImage(),
			d.SrcImageLayout(),
			d.DstImage(),
			d.DstImageLayout(),
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
	d VkCmdCopyImageToBufferArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Images().Contains(d.SrcImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage())
	}
	if !GetState(s).Buffers().Contains(d.DstBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer())
	}
	regionData, regionCount := unpackMap(ctx, s, d.Regions())

	return func() {
			regionData.Free()
		}, cb.VkCmdCopyImageToBuffer(commandBuffer,
			d.SrcImage(),
			d.SrcImageLayout(),
			d.DstBuffer(),
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
	d VkCmdCopyQueryPoolResultsArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools().Contains(d.QueryPool()) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool())
	}
	if !GetState(s).Buffers().Contains(d.DstBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.DstBuffer())
	}
	return func() {}, cb.VkCmdCopyQueryPoolResults(commandBuffer,
		d.QueryPool(),
		d.FirstQuery(),
		d.QueryCount(),
		d.DstBuffer(),
		d.DstOffset(),
		d.Stride(),
		d.Flags(),
	), nil
}

func rebuildVkCmdDispatch(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDispatchArgsʳ) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDispatch(commandBuffer,
		d.GroupCountX(),
		d.GroupCountY(),
		d.GroupCountZ(),
	), nil
}

func rebuildVkCmdDispatchIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDispatchIndirectArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Buffers().Contains(d.Buffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer())
	}
	return func() {}, cb.VkCmdDispatchIndirect(commandBuffer,
		d.Buffer(),
		d.Offset(),
	), nil
}

func rebuildVkCmdDraw(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDrawArgsʳ) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDraw(commandBuffer,
		d.VertexCount(),
		d.InstanceCount(),
		d.FirstVertex(),
		d.FirstInstance(),
	), nil
}

func rebuildVkCmdDrawIndexed(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDrawIndexedArgsʳ) (func(), api.Cmd, error) {

	return func() {}, cb.VkCmdDrawIndexed(commandBuffer, d.IndexCount(),
		d.InstanceCount(), d.FirstIndex(), d.VertexOffset(), d.FirstInstance()), nil
}

func rebuildVkCmdDrawIndexedIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDrawIndexedIndirectArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Buffers().Contains(d.Buffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer())
	}
	return func() {}, cb.VkCmdDrawIndexedIndirect(commandBuffer,
		d.Buffer(),
		d.Offset(),
		d.DrawCount(),
		d.Stride(),
	), nil
}

func rebuildVkCmdDrawIndirect(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDrawIndirectArgsʳ) (func(), api.Cmd, error) {

	if !GetState(s).Buffers().Contains(d.Buffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer())
	}
	return func() {}, cb.VkCmdDrawIndirect(commandBuffer,
		d.Buffer(),
		d.Offset(),
		d.DrawCount(),
		d.Stride(),
	), nil
}

func rebuildVkCmdEndQuery(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdEndQueryArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools().Contains(d.QueryPool()) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool())
	}
	return func() {}, cb.VkCmdEndQuery(commandBuffer,
		d.QueryPool(),
		d.Query(),
	), nil
}

func rebuildVkCmdExecuteCommands(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdExecuteCommandsArgsʳ) (func(), api.Cmd, error) {

	for i, c := 0, d.CommandBuffers().Len(); i < c; i++ {
		buf := d.CommandBuffers().Get(uint32(i))
		if !GetState(s).CommandBuffers().Contains(buf) {
			return nil, nil, fmt.Errorf("Cannot find CommandBuffer %v", buf)
		}
	}

	commandBufferData, commandBufferCount := unpackMap(ctx, s, d.CommandBuffers())

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
	d VkCmdFillBufferArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Buffers().Contains(d.Buffer()) {
		return nil, nil, fmt.Errorf("Cannot find Buffer %v", d.Buffer())
	}
	return func() {
		}, cb.VkCmdFillBuffer(commandBuffer,
			d.Buffer(),
			d.DstOffset(),
			d.Size(),
			d.Data(),
		), nil
}

func rebuildVkCmdPushConstants(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdPushConstantsArgsʳ) (func(), api.Cmd, error) {

	dat := d.Data().MustRead(ctx, nil, r, nil, nil)
	data := s.AllocDataOrPanic(ctx, dat)

	return func() {
			data.Free()
		}, cb.VkCmdPushConstants(commandBuffer,
			d.Layout(),
			d.StageFlags(),
			d.Offset(),
			d.Size(),
			data.Ptr(),
		).AddRead(data.Data()), nil
}

func rebuildVkCmdResetQueryPool(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdResetQueryPoolArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools().Contains(d.QueryPool()) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool())
	}
	return func() {
		}, cb.VkCmdResetQueryPool(commandBuffer,
			d.QueryPool(),
			d.FirstQuery(),
			d.QueryCount(),
		), nil
}

func rebuildVkCmdResolveImage(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdResolveImageArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Images().Contains(d.SrcImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.SrcImage())
	}
	if !GetState(s).Images().Contains(d.DstImage()) {
		return nil, nil, fmt.Errorf("Cannot find Image %v", d.DstImage())
	}
	resolveData, resolveCount := unpackMap(ctx, s, d.ResolveRegions())

	return func() {
			resolveData.Free()
		}, cb.VkCmdResolveImage(commandBuffer,
			d.SrcImage(),
			d.SrcImageLayout(),
			d.DstImage(),
			d.DstImageLayout(),
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
	d VkCmdSetBlendConstantsArgsʳ) (func(), api.Cmd, error) {

	constants := NewF32ː4ᵃ(s.Arena, d.R(), d.G(), d.B(), d.A())

	return func() {}, cb.VkCmdSetBlendConstants(commandBuffer, constants), nil
}

func rebuildVkCmdSetDepthBias(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetDepthBiasArgsʳ) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetDepthBias(commandBuffer,
			d.DepthBiasConstantFactor(),
			d.DepthBiasClamp(),
			d.DepthBiasSlopeFactor(),
		), nil
}

func rebuildVkCmdSetDepthBounds(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetDepthBoundsArgsʳ) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetDepthBounds(commandBuffer,
			d.MinDepthBounds(),
			d.MaxDepthBounds(),
		), nil
}

func rebuildVkCmdSetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetEventArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Events().Contains(d.Event()) {
		return nil, nil, fmt.Errorf("Cannot find Event %v", d.Event())
	}
	return func() {
		}, cb.VkCmdSetEvent(commandBuffer,
			d.Event(),
			d.StageMask(),
		), nil
}

func rebuildVkCmdResetEvent(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdResetEventArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Events().Contains(d.Event()) {
		return nil, nil, fmt.Errorf("Cannot find Event %v", d.Event())
	}
	return func() {
		}, cb.VkCmdResetEvent(commandBuffer,
			d.Event(),
			d.StageMask(),
		), nil
}

func rebuildVkCmdSetScissor(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetScissorArgsʳ) (func(), api.Cmd, error) {

	scissorData, scissorCount := unpackMap(ctx, s, d.Scissors())

	return func() {
			scissorData.Free()
		}, cb.VkCmdSetScissor(commandBuffer,
			d.FirstScissor(),
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
	d VkCmdSetStencilCompareMaskArgsʳ) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilCompareMask(commandBuffer,
			d.FaceMask(),
			d.CompareMask(),
		), nil
}

func rebuildVkCmdSetStencilReference(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetStencilReferenceArgsʳ) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilReference(commandBuffer,
			d.FaceMask(),
			d.Reference(),
		), nil
}

func rebuildVkCmdSetStencilWriteMask(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetStencilWriteMaskArgsʳ) (func(), api.Cmd, error) {

	return func() {
		}, cb.VkCmdSetStencilWriteMask(commandBuffer,
			d.FaceMask(),
			d.WriteMask(),
		), nil
}

func rebuildVkCmdSetViewport(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdSetViewportArgsʳ) (func(), api.Cmd, error) {

	viewportData, viewportCount := unpackMap(ctx, s, d.Viewports())

	return func() {
			viewportData.Free()
		}, cb.VkCmdSetViewport(commandBuffer,
			d.FirstViewport(),
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
	d VkCmdUpdateBufferArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).Buffers().Contains(d.DstBuffer()) {
		return nil, nil, fmt.Errorf("Cannot find buffer %v", d.DstBuffer())
	}

	dat := d.Data().MustRead(ctx, nil, r, nil, nil)
	data := s.AllocDataOrPanic(ctx, dat)

	return func() {
			data.Free()
		}, cb.VkCmdUpdateBuffer(commandBuffer,
			d.DstBuffer(),
			d.DstOffset(),
			d.DataSize(),
			data.Ptr(),
		).AddRead(data.Data()), nil
}

func rebuildVkCmdWriteTimestamp(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdWriteTimestampArgsʳ) (func(), api.Cmd, error) {
	if !GetState(s).QueryPools().Contains(d.QueryPool()) {
		return nil, nil, fmt.Errorf("Cannot find QueryPool %v", d.QueryPool())
	}
	return func() {
		}, cb.VkCmdWriteTimestamp(commandBuffer,
			d.PipelineStage(),
			d.QueryPool(),
			d.Query(),
		), nil
}

func rebuildVkCmdDebugMarkerBeginEXT(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDebugMarkerBeginEXTArgsʳ) (func(), api.Cmd, error) {

	a := s.Arena // TODO: Should this be a seperate temporary arena?

	markerNameData := s.AllocDataOrPanic(ctx, d.MarkerName())
	color := NewF32ː4ᵃ(a,
		d.Color().Get(0),
		d.Color().Get(1),
		d.Color().Get(2),
		d.Color().Get(3),
	)
	markerInfoData := s.AllocDataOrPanic(ctx,
		NewVkDebugMarkerMarkerInfoEXT(a,
			VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
			NewVoidᶜᵖ(memory.Nullptr),
			NewCharᶜᵖ(markerNameData.Ptr()),
			color,
		))
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
	d VkCmdDebugMarkerEndEXTArgsʳ) (func(), api.Cmd, error) {
	return func() {}, cb.VkCmdDebugMarkerEndEXT(commandBuffer), nil
}

func rebuildVkCmdDebugMarkerInsertEXT(
	ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	d VkCmdDebugMarkerInsertEXTArgsʳ) (func(), api.Cmd, error) {

	a := s.Arena // TODO: Should this be a seperate temporary arena?

	markerNameData := s.AllocDataOrPanic(ctx, d.MarkerName())
	color := NewF32ː4ᵃ(a, d.Color().Get(0), d.Color().Get(1), d.Color().Get(2), d.Color().Get(3))
	markerInfoData := s.AllocDataOrPanic(ctx,
		NewVkDebugMarkerMarkerInfoEXT(a,
			VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
			NewVoidᶜᵖ(memory.Nullptr),
			NewCharᶜᵖ(markerNameData.Ptr()),
			color,
		))
	return func() {
			markerNameData.Free()
			markerInfoData.Free()
		}, cb.VkCmdDebugMarkerInsertEXT(commandBuffer, markerInfoData.Ptr()).AddRead(
			markerNameData.Data()).AddRead(markerInfoData.Data()), nil
}

// GetCommandArgs takes a command reference and returns the command arguments
// of that recorded command.
func GetCommandArgs(ctx context.Context,
	cr CommandReferenceʳ,
	s *State) interface{} {

	cmds := s.CommandBuffers().Get(cr.Buffer()).BufferCommands()

	switch cr.Type() {
	case CommandType_cmd_vkCmdBeginRenderPass:
		return cmds.VkCmdBeginRenderPass().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdEndRenderPass:
		return cmds.VkCmdEndRenderPass().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdNextSubpass:
		return cmds.VkCmdNextSubpass().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBindPipeline:
		return cmds.VkCmdBindPipeline().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBindDescriptorSets:
		return cmds.VkCmdBindDescriptorSets().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBindVertexBuffers:
		return cmds.VkCmdBindVertexBuffers().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBindIndexBuffer:
		return cmds.VkCmdBindIndexBuffer().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdPipelineBarrier:
		return cmds.VkCmdPipelineBarrier().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdWaitEvents:
		return cmds.VkCmdWaitEvents().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBeginQuery:
		return cmds.VkCmdBeginQuery().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdBlitImage:
		return cmds.VkCmdBlitImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdClearAttachments:
		return cmds.VkCmdClearAttachments().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdClearColorImage:
		return cmds.VkCmdClearColorImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdClearDepthStencilImage:
		return cmds.VkCmdClearDepthStencilImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdCopyBuffer:
		return cmds.VkCmdCopyBuffer().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdCopyBufferToImage:
		return cmds.VkCmdCopyBufferToImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdCopyImage:
		return cmds.VkCmdCopyImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdCopyImageToBuffer:
		return cmds.VkCmdCopyImageToBuffer().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdCopyQueryPoolResults:
		return cmds.VkCmdCopyQueryPoolResults().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDispatch:
		return cmds.VkCmdDispatch().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDispatchIndirect:
		return cmds.VkCmdDispatchIndirect().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDraw:
		return cmds.VkCmdDraw().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDrawIndexed:
		return cmds.VkCmdDrawIndexed().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDrawIndexedIndirect:
		return cmds.VkCmdDrawIndexedIndirect().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDrawIndirect:
		return cmds.VkCmdDrawIndirect().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdEndQuery:
		return cmds.VkCmdEndQuery().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdExecuteCommands:
		return cmds.VkCmdExecuteCommands().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdFillBuffer:
		return cmds.VkCmdFillBuffer().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdPushConstants:
		return cmds.VkCmdPushConstants().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdResetQueryPool:
		return cmds.VkCmdResetQueryPool().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdResolveImage:
		return cmds.VkCmdResolveImage().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetBlendConstants:
		return cmds.VkCmdSetBlendConstants().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetDepthBias:
		return cmds.VkCmdSetDepthBias().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetDepthBounds:
		return cmds.VkCmdSetDepthBounds().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetEvent:
		return cmds.VkCmdSetEvent().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdResetEvent:
		return cmds.VkCmdResetEvent().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetLineWidth:
		return cmds.VkCmdSetLineWidth().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetScissor:
		return cmds.VkCmdSetScissor().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetStencilCompareMask:
		return cmds.VkCmdSetStencilCompareMask().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetStencilReference:
		return cmds.VkCmdSetStencilReference().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetStencilWriteMask:
		return cmds.VkCmdSetStencilWriteMask().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdSetViewport:
		return cmds.VkCmdSetViewport().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdUpdateBuffer:
		return cmds.VkCmdUpdateBuffer().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdWriteTimestamp:
		return cmds.VkCmdWriteTimestamp().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDebugMarkerBeginEXT:
		return cmds.VkCmdDebugMarkerBeginEXT().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDebugMarkerEndEXT:
		return cmds.VkCmdDebugMarkerEndEXT().Get(cr.MapIndex())
	case CommandType_cmd_vkCmdDebugMarkerInsertEXT:
		return cmds.VkCmdDebugMarkerInsertEXT().Get(cr.MapIndex())
	default:
		x := fmt.Sprintf("Should not reach here: %T", cr)
		panic(x)
	}
}

// GetCommandFunction takes a command reference (recorded command buffer
// command) and returns the function which mutates the state as the recorded
// command get executed.
func GetCommandFunction(cr *CommandReference) interface{} {
	switch cr.Type() {
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
// into the given command buffer. It returns the commands that it
// had to create in order to satisfy the command. It also returns a function
// to clean up the data that was allocated during the creation.
func AddCommand(ctx context.Context,
	cb CommandBuilder,
	commandBuffer VkCommandBuffer,
	r *api.GlobalState,
	s *api.GlobalState,
	rebuildInfo interface{}) (func(), api.Cmd, error) {

	switch t := rebuildInfo.(type) {
	case VkCmdBeginRenderPassArgsʳ:
		return rebuildVkCmdBeginRenderPass(ctx, cb, commandBuffer, r, s, t)
	case VkCmdEndRenderPassArgsʳ:
		return rebuildVkCmdEndRenderPass(ctx, cb, commandBuffer, r, s, t)
	case VkCmdNextSubpassArgsʳ:
		return rebuildVkCmdNextSubpass(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBindPipelineArgsʳ:
		return rebuildVkCmdBindPipeline(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBindDescriptorSetsArgsʳ:
		return rebuildVkCmdBindDescriptorSets(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBindVertexBuffersArgsʳ:
		return rebuildVkCmdBindVertexBuffers(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBindIndexBufferArgsʳ:
		return rebuildVkCmdBindIndexBuffer(ctx, cb, commandBuffer, r, s, t)
	case VkCmdPipelineBarrierArgsʳ:
		return rebuildVkCmdPipelineBarrier(ctx, cb, commandBuffer, r, s, t)
	case VkCmdWaitEventsArgsʳ:
		return rebuildVkCmdWaitEvents(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBeginQueryArgsʳ:
		return rebuildVkCmdBeginQuery(ctx, cb, commandBuffer, r, s, t)
	case VkCmdBlitImageArgsʳ:
		return rebuildVkCmdBlitImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdClearAttachmentsArgsʳ:
		return rebuildVkCmdClearAttachments(ctx, cb, commandBuffer, r, s, t)
	case VkCmdClearColorImageArgsʳ:
		return rebuildVkCmdClearColorImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdClearDepthStencilImageArgsʳ:
		return rebuildVkCmdClearDepthStencilImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdCopyBufferArgsʳ:
		return rebuildVkCmdCopyBuffer(ctx, cb, commandBuffer, r, s, t)
	case VkCmdCopyBufferToImageArgsʳ:
		return rebuildVkCmdCopyBufferToImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdCopyImageArgsʳ:
		return rebuildVkCmdCopyImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdCopyImageToBufferArgsʳ:
		return rebuildVkCmdCopyImageToBuffer(ctx, cb, commandBuffer, r, s, t)
	case VkCmdCopyQueryPoolResultsArgsʳ:
		return rebuildVkCmdCopyQueryPoolResults(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDispatchArgsʳ:
		return rebuildVkCmdDispatch(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDispatchIndirectArgsʳ:
		return rebuildVkCmdDispatchIndirect(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDrawArgsʳ:
		return rebuildVkCmdDraw(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDrawIndexedArgsʳ:
		return rebuildVkCmdDrawIndexed(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDrawIndexedIndirectArgsʳ:
		return rebuildVkCmdDrawIndexedIndirect(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDrawIndirectArgsʳ:
		return rebuildVkCmdDrawIndirect(ctx, cb, commandBuffer, r, s, t)
	case VkCmdEndQueryArgsʳ:
		return rebuildVkCmdEndQuery(ctx, cb, commandBuffer, r, s, t)
	case VkCmdExecuteCommandsArgsʳ:
		return rebuildVkCmdExecuteCommands(ctx, cb, commandBuffer, r, s, t)
	case VkCmdFillBufferArgsʳ:
		return rebuildVkCmdFillBuffer(ctx, cb, commandBuffer, r, s, t)
	case VkCmdPushConstantsArgsʳ:
		return rebuildVkCmdPushConstants(ctx, cb, commandBuffer, r, s, t)
	case VkCmdResetQueryPoolArgsʳ:
		return rebuildVkCmdResetQueryPool(ctx, cb, commandBuffer, r, s, t)
	case VkCmdResolveImageArgsʳ:
		return rebuildVkCmdResolveImage(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetBlendConstantsArgsʳ:
		return rebuildVkCmdSetBlendConstants(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetDepthBiasArgsʳ:
		return rebuildVkCmdSetDepthBias(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetDepthBoundsArgsʳ:
		return rebuildVkCmdSetDepthBounds(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetEventArgsʳ:
		return rebuildVkCmdSetEvent(ctx, cb, commandBuffer, r, s, t)
	case VkCmdResetEventArgsʳ:
		return rebuildVkCmdResetEvent(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetLineWidthArgsʳ:
		return rebuildVkCmdSetLineWidth(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetScissorArgsʳ:
		return rebuildVkCmdSetScissor(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetStencilCompareMaskArgsʳ:
		return rebuildVkCmdSetStencilCompareMask(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetStencilReferenceArgsʳ:
		return rebuildVkCmdSetStencilReference(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetStencilWriteMaskArgsʳ:
		return rebuildVkCmdSetStencilWriteMask(ctx, cb, commandBuffer, r, s, t)
	case VkCmdSetViewportArgsʳ:
		return rebuildVkCmdSetViewport(ctx, cb, commandBuffer, r, s, t)
	case VkCmdUpdateBufferArgsʳ:
		return rebuildVkCmdUpdateBuffer(ctx, cb, commandBuffer, r, s, t)
	case VkCmdWriteTimestampArgsʳ:
		return rebuildVkCmdWriteTimestamp(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDebugMarkerBeginEXTArgsʳ:
		return rebuildVkCmdDebugMarkerBeginEXT(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDebugMarkerEndEXTArgsʳ:
		return rebuildVkCmdDebugMarkerEndEXT(ctx, cb, commandBuffer, r, s, t)
	case VkCmdDebugMarkerInsertEXTArgsʳ:
		return rebuildVkCmdDebugMarkerInsertEXT(ctx, cb, commandBuffer, r, s, t)
	default:
		x := fmt.Sprintf("Should not reach here: %T", t)
		panic(x)
	}
}
