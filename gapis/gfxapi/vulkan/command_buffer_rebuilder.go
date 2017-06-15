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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
)

func rebuildCmdBeginRenderPass(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdBeginRenderPassData) (func(), atom.Atom) {

	clearValues := make([]VkClearValue, len(d.ClearValues))
	for i := 0; i < len(d.ClearValues); i++ {
		clearValues[i] = d.ClearValues[uint32(i)]
	}

	clearValuesData := atom.Must(atom.AllocData(ctx, s, clearValues))

	begin := VkRenderPassBeginInfo{
		VkStructureType_VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
		NewVoidᶜᵖ(memory.Nullptr),
		d.RenderPass,
		d.Framebuffer,
		d.RenderArea,
		uint32(len(clearValues)),
		NewVkClearValueᶜᵖ(clearValuesData.Ptr()),
	}
	beginData := atom.Must(atom.AllocData(ctx, s, begin))

	return func() {
			clearValuesData.Free()
			beginData.Free()
		}, NewVkCmdBeginRenderPass(
			commandBuffer,
			beginData.Ptr(),
			d.Contents).AddRead(beginData.Data()).AddRead(clearValuesData.Data())
}

func rebuildCmdEndRenderPass(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdEndRenderPassData) (func(), atom.Atom) {
	return func() {}, NewVkCmdEndRenderPass(commandBuffer)
}

func rebuildCmdNextSubpass(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdNextSubpassData) (func(), atom.Atom) {
	return func() {}, NewVkCmdNextSubpass(commandBuffer, d.Contents)
}

func rebuildCmdBindPipeline(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdBindPipelineData) (func(), atom.Atom) {

	return func() {}, NewVkCmdBindPipeline(commandBuffer,
		d.PipelineBindPoint, d.Pipeline)
}

func rebuildCmdBindIndexBuffer(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdBindIndexBufferData) (func(), atom.Atom) {

	return func() {}, NewVkCmdBindIndexBuffer(commandBuffer,
		d.Buffer, d.Offset, d.IndexType)
}

func rebuildCmdDrawIndexed(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdDrawIndexedData) (func(), atom.Atom) {

	return func() {}, NewVkCmdDrawIndexed(commandBuffer, d.IndexCount,
		d.InstanceCount, d.FirstIndex, d.VertexOffset, d.FirstInstance)
}

func rebuildCmdBindDescriptorSets(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdBindDescriptorSetsData) (func(), atom.Atom) {

	descriptorSets := make([]VkDescriptorSet, len(d.DescriptorSets))
	dynamicOffsets := make([]uint32, len(d.DynamicOffsets))

	for i := 0; i < len(d.DescriptorSets); i++ {
		descriptorSets[i] = d.DescriptorSets[uint32(i)]
	}

	for i := 0; i < len(d.DynamicOffsets); i++ {
		dynamicOffsets[i] = d.DynamicOffsets[uint32(i)]
	}

	descriptorSetData := atom.Must(atom.AllocData(ctx, s, descriptorSets))
	dynamicOffsetData := atom.Must(atom.AllocData(ctx, s, dynamicOffsets))

	return func() {
			descriptorSetData.Free()
			dynamicOffsetData.Free()
		}, NewVkCmdBindDescriptorSets(commandBuffer,
			d.PipelineBindPoint,
			d.Layout,
			d.FirstSet,
			uint32(len(d.DescriptorSets)),
			descriptorSetData.Ptr(),
			uint32(len(d.DynamicOffsets)),
			dynamicOffsetData.Ptr(),
		).AddRead(
			dynamicOffsetData.Data(),
		).AddRead(descriptorSetData.Data())
}

func rebuildCmdBindVertexBuffers(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdBindVertexBuffersData) (func(), atom.Atom) {

	buffers := make([]VkBuffer, len(d.Buffers))
	offsets := make([]VkDeviceSize, len(d.Offsets))

	for i := 0; i < len(d.Buffers); i++ {
		buffers[i] = d.Buffers[uint32(i)]
	}

	for i := 0; i < len(d.Offsets); i++ {
		offsets[i] = d.Offsets[uint32(i)]
	}

	bufferData := atom.Must(atom.AllocData(ctx, s, buffers))
	offsetData := atom.Must(atom.AllocData(ctx, s, offsets))

	return func() {
			bufferData.Free()
			offsetData.Free()
		}, NewVkCmdBindVertexBuffers(commandBuffer,
			d.FirstBinding,
			d.BindingCount,
			bufferData.Ptr(),
			offsetData.Ptr(),
		).AddRead(offsetData.Data()).AddRead(bufferData.Data())
}

func rebuildCmdWaitEvents(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdWaitEventsData) (func(), atom.Atom) {

	events := make([]VkEvent, len(d.Events))
	memoryBarriers := make([]VkMemoryBarrier, len(d.MemoryBarriers))
	bufferMemoryBarriers := make([]VkBufferMemoryBarrier, len(d.BufferMemoryBarriers))
	imageMemoryBarriers := make([]VkImageMemoryBarrier, len(d.ImageMemoryBarriers))

	for i := 0; i < len(d.Events); i++ {
		events[i] = d.Events[uint32(i)]
	}

	for i := 0; i < len(d.MemoryBarriers); i++ {
		memoryBarriers[i] = d.MemoryBarriers[uint32(i)]
	}

	for i := 0; i < len(d.BufferMemoryBarriers); i++ {
		bufferMemoryBarriers[i] = d.BufferMemoryBarriers[uint32(i)]
	}

	for i := 0; i < len(d.ImageMemoryBarriers); i++ {
		imageMemoryBarriers[i] = d.ImageMemoryBarriers[uint32(i)]
	}

	eventData := atom.Must(atom.AllocData(ctx, s, events))
	memoryBarrierData := atom.Must(atom.AllocData(ctx, s, memoryBarriers))
	bufferMemoryBarrierData := atom.Must(atom.AllocData(ctx, s, bufferMemoryBarriers))
	imageMemoryBarrierData := atom.Must(atom.AllocData(ctx, s, imageMemoryBarriers))

	return func() {
			eventData.Free()
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, NewVkCmdWaitEvents(commandBuffer,
			uint32(len(events)),
			eventData.Ptr(),
			d.SrcStageMask,
			d.DstStageMask,
			uint32(len(memoryBarriers)),
			memoryBarrierData.Ptr(),
			uint32(len(bufferMemoryBarriers)),
			bufferMemoryBarrierData.Ptr(),
			uint32(len(imageMemoryBarriers)),
			imageMemoryBarrierData.Ptr(),
		).AddRead(eventData.Data()).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data())
}

func rebuildCmdPipelineBarrier(
	ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	d *RecreateCmdPipelineBarrierData) (func(), atom.Atom) {

	memoryBarriers := make([]VkMemoryBarrier, len(d.MemoryBarriers))
	bufferMemoryBarriers := make([]VkBufferMemoryBarrier, len(d.BufferMemoryBarriers))
	imageMemoryBarriers := make([]VkImageMemoryBarrier, len(d.ImageMemoryBarriers))

	for i := 0; i < len(d.MemoryBarriers); i++ {
		memoryBarriers[i] = d.MemoryBarriers[uint32(i)]
	}

	for i := 0; i < len(d.BufferMemoryBarriers); i++ {
		bufferMemoryBarriers[i] = d.BufferMemoryBarriers[uint32(i)]
	}

	for i := 0; i < len(d.ImageMemoryBarriers); i++ {
		imageMemoryBarriers[i] = d.ImageMemoryBarriers[uint32(i)]
	}

	memoryBarrierData := atom.Must(atom.AllocData(ctx, s, memoryBarriers))
	bufferMemoryBarrierData := atom.Must(atom.AllocData(ctx, s, bufferMemoryBarriers))
	imageMemoryBarrierData := atom.Must(atom.AllocData(ctx, s, imageMemoryBarriers))

	return func() {
			memoryBarrierData.Free()
			bufferMemoryBarrierData.Free()
			imageMemoryBarrierData.Free()
		}, NewVkCmdPipelineBarrier(commandBuffer,
			d.SrcStageMask,
			d.DstStageMask,
			d.DependencyFlags,
			uint32(len(memoryBarriers)),
			memoryBarrierData.Ptr(),
			uint32(len(bufferMemoryBarriers)),
			bufferMemoryBarrierData.Ptr(),
			uint32(len(imageMemoryBarriers)),
			imageMemoryBarrierData.Ptr(),
		).AddRead(memoryBarrierData.Data()).AddRead(bufferMemoryBarrierData.Data()).AddRead(imageMemoryBarrierData.Data())
}

// AddCommand recreates the command defined by recreateInfo and places it
// into the given command buffer. It returns the atoms that it
// had to create in order to satisfy the command. It also returns a function
// to clean up the data that was allocated during the creation.
func AddCommand(ctx context.Context,
	commandBuffer VkCommandBuffer,
	s *gfxapi.State,
	rebuildInfo interface{}) (func(), atom.Atom) {
	switch t := rebuildInfo.(type) {
	case *RecreateCmdBeginRenderPassData:
		return rebuildCmdBeginRenderPass(ctx, commandBuffer, s, t)
	case *RecreateCmdEndRenderPassData:
		return rebuildCmdEndRenderPass(ctx, commandBuffer, s, t)
	case *RecreateCmdNextSubpassData:
		return rebuildCmdNextSubpass(ctx, commandBuffer, s, t)
	case *RecreateCmdBindPipelineData:
		return rebuildCmdBindPipeline(ctx, commandBuffer, s, t)
	case *RecreateCmdBindDescriptorSetsData:
		return rebuildCmdBindDescriptorSets(ctx, commandBuffer, s, t)
	case *RecreateCmdBindVertexBuffersData:
		return rebuildCmdBindVertexBuffers(ctx, commandBuffer, s, t)
	case *RecreateCmdBindIndexBufferData:
		return rebuildCmdBindIndexBuffer(ctx, commandBuffer, s, t)
	case *RecreateCmdDrawIndexedData:
		return rebuildCmdDrawIndexed(ctx, commandBuffer, s, t)
	case *RecreateCmdPipelineBarrierData:
		return rebuildCmdPipelineBarrier(ctx, commandBuffer, s, t)
	case *RecreateCmdWaitEventsData:
		return rebuildCmdWaitEvents(ctx, commandBuffer, s, t)
	default:
		x := fmt.Sprintf("Should not reach here: %T", t)
		panic(x)
	}
}
