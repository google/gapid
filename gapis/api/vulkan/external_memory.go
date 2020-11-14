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
	"math"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/vulkan/vulkan_pb"
	"github.com/google/gapid/gapis/memory"
)

type ExternalBufferData struct {
	Buffer             VkBuffer
	BufferOffset       VkDeviceSize
	DataOffset         VkDeviceSize
	Size               VkDeviceSize
	SubmitIndex        uint32
	CommandBufferIndex uint32
}

type ExternalImageDataRange struct {
	DataOffset  VkDeviceSize
	Subresource VkImageSubresourceLayers
}

type ExternalImageData struct {
	Image              VkImage
	BarrierRange       VkImageSubresourceRange
	OldLayout          VkImageLayout
	NewLayout          VkImageLayout
	SubmitIndex        uint32
	CommandBufferIndex uint32
	CopyRanges         []ExternalImageDataRange
}

type ExternalMemoryData struct {
	ID      id.ID
	Size    VkDeviceSize
	Buffers []ExternalBufferData
	Images  []ExternalImageData
}

// ExternalMemoryData returns a pointer to the ExternalMemoryData structure in the
// CmdExtras, or nil if there are no observations in the CmdExtras.
func GetExternalMemoryData(e *api.CmdExtras) *ExternalMemoryData {
	for _, o := range e.All() {
		if o, ok := o.(*ExternalMemoryData); ok {
			return o
		}
	}
	return nil
}

type externalMemoryCommandBuffer struct {
	commandBuffer        VkCommandBuffer
	stagingCommandBuffer VkCommandBuffer
	buffers              []ExternalBufferData
	images               []ExternalImageData
}

type externalMemorySubmitInfo struct {
	submitInfo     VkSubmitInfo
	commandBuffers []externalMemoryCommandBuffer
}

type externalMemoryStaging struct {
	h *vkQueueSubmitHijack

	queueFamilyIndex uint32
	device           VkDevice

	stagingBufferSize VkDeviceSize
	stagingMemorySize VkDeviceSize
	stagingData       id.ID

	submits []externalMemorySubmitInfo

	stagingBuffer        VkBuffer
	stagingMemory        VkDeviceMemory
	stagingCommandPool   VkCommandPool
	stagingCommandBuffer VkCommandBuffer
}

func (e *externalMemoryStaging) initialize(h *vkQueueSubmitHijack, externalData *ExternalMemoryData) {
	e.h = h
	e.stagingData = externalData.ID
	e.stagingBufferSize = externalData.Size
	e.stagingMemorySize = 2 * e.stagingBufferSize

	queueObj := e.h.c.Queues().Get(h.get().Queue())
	e.device = queueObj.Device()
	e.queueFamilyIndex = queueObj.Family()

	submitInfos := e.h.submitInfos()
	e.submits = make([]externalMemorySubmitInfo, len(submitInfos))

	for i, submitInfo := range submitInfos {
		e.submits[i].submitInfo = submitInfo
		e.submits[i].commandBuffers = make([]externalMemoryCommandBuffer, submitInfo.CommandBufferCount())
		commandBufferCount := uint64(submitInfo.CommandBufferCount())
		commandBuffers := submitInfo.PCommandBuffers().Slice(0, commandBufferCount, e.h.s.MemoryLayout).MustRead(e.h.ctx, e.h.origSubmit, e.h.s, nil)
		for j, commandBuffer := range commandBuffers {
			e.submits[i].commandBuffers[j].commandBuffer = commandBuffer
		}
	}

	for _, extBuf := range externalData.Buffers {
		buffers := &e.submits[extBuf.SubmitIndex].commandBuffers[extBuf.CommandBufferIndex].buffers
		*buffers = append(*buffers, extBuf)
	}

	for _, extImg := range externalData.Images {
		images := &e.submits[extImg.SubmitIndex].commandBuffers[extImg.CommandBufferIndex].images
		*images = append(*images, extImg)
	}
}

func (e *externalMemoryStaging) createCommandPool() error {
	commandPool := VkCommandPool(newUnusedID(false, func(x uint64) bool { return e.h.c.CommandPools().Contains(VkCommandPool(x)) }))
	pCommandPool := e.h.mustAllocData(commandPool)
	pCommandPoolCreateInfo := e.h.mustAllocData(NewVkCommandPoolCreateInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		e.queueFamilyIndex, // queueFamilyIndex
	))
	err := e.h.cb.VkCreateCommandPool(
		e.device,
		pCommandPoolCreateInfo.Ptr(),
		memory.Nullptr,
		pCommandPool.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		pCommandPoolCreateInfo.Data(),
	).AddWrite(
		pCommandPool.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return err
	}
	e.stagingCommandPool = commandPool
	return nil
}

func (e *externalMemoryStaging) allocCommandBuffer() (VkCommandBuffer, error) {
	commandBuffer := VkCommandBuffer(newUnusedID(false, func(x uint64) bool { return e.h.c.CommandBuffers().Contains(VkCommandBuffer(x)) }))
	pCommandBuffer := e.h.mustAllocData(commandBuffer)
	pCommandBufferAllocInfo := e.h.mustAllocData(NewVkCommandBufferAllocateInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		e.stagingCommandPool,                                           // commandPool
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
		1, // commandBufferCount
	))

	err := e.h.cb.VkAllocateCommandBuffers(
		e.device,
		pCommandBufferAllocInfo.Ptr(),
		pCommandBuffer.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		pCommandBufferAllocInfo.Data(),
	).AddWrite(
		pCommandBuffer.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return VkCommandBuffer(0), err
	}

	return commandBuffer, nil
}

func (e *externalMemoryStaging) createBuffer() error {
	bufferID := VkBuffer(newUnusedID(false, func(x uint64) bool { ok := e.h.c.Buffers().Contains(VkBuffer(x)); return ok }))
	pBuffer := e.h.mustAllocData(bufferID)
	pCreateInfo := e.h.mustAllocData(NewVkBufferCreateInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                            // pNext
		0,                                                    // flags
		VkDeviceSize(e.stagingBufferSize),                    // size
		VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT), // usage
		VkSharingMode_VK_SHARING_MODE_EXCLUSIVE,                                    // sharingMode
		0,                                                                          // queueFamilyIndexCount
		NewU32ᶜᵖ(memory.Nullptr),                                                   // pQueueFamilyIndices
	))

	err := e.h.cb.VkCreateBuffer(
		e.device,
		pCreateInfo.Ptr(),
		memory.Nullptr,
		pBuffer.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		pCreateInfo.Data(),
	).AddWrite(
		pBuffer.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return err
	}
	e.stagingBuffer = bufferID
	return nil
}

func (e *externalMemoryStaging) allocMemory() error {
	deviceObj := e.h.c.Devices().Get(e.device)
	physicalDeviceObj := e.h.c.PhysicalDevices().Get(deviceObj.PhysicalDevice())
	memProps := physicalDeviceObj.MemoryProperties()

	memoryID := VkDeviceMemory(newUnusedID(false, func(x uint64) bool { ok := e.h.c.DeviceMemories().Contains(VkDeviceMemory(x)); return ok }))
	pStagingMemory := e.h.mustAllocData(memoryID)

	stagingMemoryTypeIndex := uint32(math.MaxUint32)
	for i := uint32(0); i < memProps.MemoryTypeCount(); i++ {
		t := memProps.MemoryTypes().Get(int(i))
		if 0 != (t.PropertyFlags() & VkMemoryPropertyFlags(
			VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT)) {
			stagingMemoryTypeIndex = i
			break
		}
	}

	pAllocInfo := e.h.mustAllocData(NewVkMemoryAllocateInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                              // pNext
		VkDeviceSize(e.stagingMemorySize),                      // allocationSize
		stagingMemoryTypeIndex,                                 // memoryTypeIndex
	))

	err := e.h.cb.VkAllocateMemory(
		e.device,
		pAllocInfo.Ptr(),
		memory.Nullptr,
		pStagingMemory.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		pAllocInfo.Data(),
	).AddWrite(
		pStagingMemory.Data(),
	).mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return err
	}
	e.stagingMemory = memoryID
	return nil
}

func (e *externalMemoryStaging) bindBufferMemory() error {
	err := e.h.cb.VkBindBufferMemory(
		e.device,
		e.stagingBuffer,
		e.stagingMemory,
		VkDeviceSize(0),
		VkResult_VK_SUCCESS,
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (e *externalMemoryStaging) createResources() error {
	if err := e.createCommandPool(); err != nil {
		return err
	}
	if cmdBuf, err := e.allocCommandBuffer(); err != nil {
		return err
	} else {
		e.stagingCommandBuffer = cmdBuf
	}
	for i := range e.submits {
		for j := range e.submits[i].commandBuffers {
			cmdBuf := &e.submits[i].commandBuffers[j]
			if len(cmdBuf.buffers) > 0 || len(cmdBuf.images) > 0 {
				if stagingCommandBuffer, err := e.allocCommandBuffer(); err != nil {
					return err
				} else {
					cmdBuf.stagingCommandBuffer = stagingCommandBuffer
				}
			}
		}
	}
	if err := e.createBuffer(); err != nil {
		return err
	}
	if err := e.allocMemory(); err != nil {
		return err
	}
	if err := e.bindBufferMemory(); err != nil {
		return err
	}
	return nil
}

func (e *externalMemoryStaging) beginCommandBuffer(commandBuffer VkCommandBuffer) error {
	pBeginInfo := e.h.mustAllocData(NewVkCommandBufferBeginInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,                                         // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                           // pNext
		VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr),                                                 // pInheritanceInfo
	))

	return e.h.cb.VkBeginCommandBuffer(
		commandBuffer,
		pBeginInfo.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		pBeginInfo.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) endCommandBuffer(commandBuffer VkCommandBuffer) error {
	return e.h.cb.VkEndCommandBuffer(
		commandBuffer,
		VkResult_VK_SUCCESS,
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) cmdPipelineBarrier(
	commandBuffer VkCommandBuffer,
	srcStageMask VkPipelineStageFlags,
	dstStageMask VkPipelineStageFlags,
	bufferBarriers []VkBufferMemoryBarrier,
	imageBarriers []VkImageMemoryBarrier) error {
	pBufferBarriers := e.h.mustAllocData(bufferBarriers)
	pImageBarriers := e.h.mustAllocData(imageBarriers)

	return e.h.cb.VkCmdPipelineBarrier(
		commandBuffer,               // commandBuffer
		srcStageMask,                // srcStageMask
		dstStageMask,                // dstStageMask
		0,                           // dependencyFlags
		0,                           // memoryBarrierCount
		memory.Nullptr,              // pMemoryBarriers
		uint32(len(bufferBarriers)), // bufferMemoryBarrierCount
		pBufferBarriers.Ptr(),       // pBufferMemoryBarriers
		uint32(len(imageBarriers)),  // imageMemoryBarrierCount
		pImageBarriers.Ptr(),        // pImageMemoryBarriers
	).AddRead(
		pBufferBarriers.Data(),
	).AddRead(
		pImageBarriers.Data(),
	).mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) cmdCopyBuffer(
	commandBuffer VkCommandBuffer,
	srcBuffer VkBuffer,
	dstBuffer VkBuffer,
	regions []VkBufferCopy) error {
	pRegions := e.h.mustAllocData(regions)

	return e.h.cb.VkCmdCopyBuffer(
		commandBuffer,        // commandBuffer
		srcBuffer,            // srcBuffer
		dstBuffer,            // dstBuffer
		uint32(len(regions)), // regionCount
		pRegions.Ptr(),       // pRegions
	).AddRead(
		pRegions.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) cmdCopyBufferToImage(
	commandBuffer VkCommandBuffer,
	srcBuffer VkBuffer,
	dstImage VkImage,
	regions []VkBufferImageCopy) error {
	pRegions := e.h.mustAllocData(regions)

	return e.h.cb.VkCmdCopyBufferToImage(
		commandBuffer, // commandBuffer
		srcBuffer,     // srcBuffer
		dstImage,      // dstImage
		VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, // dstImageLayout
		uint32(len(regions)), // regionCount
		pRegions.Ptr(),       // pRegions
	).AddRead(
		pRegions.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) recordCommandBuffers() error {
	if err := e.beginCommandBuffer(e.stagingCommandBuffer); err != nil {
		return err
	}
	imageLayoutBarriers := []VkImageMemoryBarrier{}
	for _, submit := range e.submits {
		for _, cmdBuf := range submit.commandBuffers {
			for _, img := range cmdBuf.images {
				imageLayoutBarriers = append(imageLayoutBarriers, NewVkImageMemoryBarrier(
					e.h.s.Arena,
					VkStructureType_VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
					NewVoidᶜᵖ(memory.Nullptr),                              // pNext
					VkAccessFlags(0),                                       // srcAccessMask
					VkAccessFlags(0),                                       // dstAccessMask
					VkImageLayout_VK_IMAGE_LAYOUT_UNDEFINED,                // oldLayout
					VkImageLayout_VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,     // newLayout
					e.queueFamilyIndex,                                     // srcQueueFamilyIndex
					e.queueFamilyIndex,                                     // dstQueueFamilyIndex
					img.Image,                                              // image
					img.BarrierRange,                                       // subresourceRange
				))
			}
		}
	}
	err := e.cmdPipelineBarrier(
		e.stagingCommandBuffer, // commandBuffer
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_HOST_BIT),     // srcStageMask
		VkPipelineStageFlags(VkPipelineStageFlagBits_VK_PIPELINE_STAGE_TRANSFER_BIT), // dstStageMask
		[]VkBufferMemoryBarrier{
			NewVkBufferMemoryBarrier(
				e.h.s.Arena,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,     // sType
				NewVoidᶜᵖ(memory.Nullptr),                                   // pNext
				VkAccessFlags(VkAccessFlagBits_VK_ACCESS_HOST_WRITE_BIT),    // srcAccessMask
				VkAccessFlags(VkAccessFlagBits_VK_ACCESS_TRANSFER_READ_BIT), // dstAccessMask
				e.queueFamilyIndex,  // srcQueueFamilyIndex
				e.queueFamilyIndex,  // dstQueueFamilyIndex
				e.stagingBuffer,     // buffer
				0,                   // offset
				e.stagingBufferSize, // size
			),
		},
		imageLayoutBarriers,
	)
	if err != nil {
		return err
	}
	if err := e.endCommandBuffer(e.stagingCommandBuffer); err != nil {
		return err
	}

	for _, submit := range e.submits {
		for _, cmdBuf := range submit.commandBuffers {
			if cmdBuf.stagingCommandBuffer == VkCommandBuffer(0) {
				continue
			}
			if err := e.beginCommandBuffer(cmdBuf.stagingCommandBuffer); err != nil {
				return err
			}
			for _, buf := range cmdBuf.buffers {
				err := e.cmdCopyBuffer(
					cmdBuf.stagingCommandBuffer, // commandBuffer
					e.stagingBuffer,             // srcBuffer
					buf.Buffer,                  // dstBuffer
					[]VkBufferCopy{
						NewVkBufferCopy(
							e.h.s.Arena,
							buf.DataOffset,   // srcOffset
							buf.BufferOffset, // dstOffset
							buf.Size,         // size
						),
					},
				)
				if err != nil {
					return err
				}
			}
			for _, img := range cmdBuf.images {
				copies := make([]VkBufferImageCopy, 0, len(img.CopyRanges))
				extent := e.h.c.Images().Get(img.Image).Info().Extent()
				offset := NewVkOffset3D(e.h.s.Arena, 0, 0, 0)
				for _, rng := range img.CopyRanges {
					copies = append(copies, NewVkBufferImageCopy(
						e.h.s.Arena,
						rng.DataOffset,  // bufferOffset
						0,               // bufferRowLength
						0,               // bufferImageHeight
						rng.Subresource, // imageSubresource
						offset,          // imageOffset
						extent,          // imageExtent
					))
				}
				err := e.cmdCopyBufferToImage(
					cmdBuf.stagingCommandBuffer, // commandBuffer
					e.stagingBuffer,             // srcBuffer
					img.Image,                   // dstImage
					copies,
				)
				if err != nil {
					return err
				}
			}
			if err := e.endCommandBuffer(cmdBuf.stagingCommandBuffer); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *externalMemoryStaging) mapMemory() (memory.Range, error) {
	VK_WHOLE_SIZE := VkDeviceSize(0xFFFFFFFFFFFFFFFF)

	at := e.h.mustAlloc(uint64(e.stagingMemorySize))
	mappedPointer := e.h.mustAllocData(at.Address())

	err := e.h.cb.VkMapMemory(
		e.device,
		e.stagingMemory,
		0,
		VK_WHOLE_SIZE,
		VkMemoryMapFlags(0),
		mappedPointer.Ptr(),
		VkResult_VK_SUCCESS,
	).AddWrite(
		mappedPointer.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
	if err != nil {
		return memory.Range{}, err
	}
	return at.Range(), nil
}

func (e *externalMemoryStaging) flushMappedMemory(at memory.Range) error {
	VK_WHOLE_SIZE := VkDeviceSize(0xFFFFFFFFFFFFFFFF)

	pRange := e.h.mustAllocData(NewVkMappedMemoryRange(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
		NewVoidᶜᵖ(memory.Nullptr),                             // pNext
		e.stagingMemory,                                       // memory
		VkDeviceSize(0),                                       // offset
		VK_WHOLE_SIZE,                                         // size
	))

	return e.h.cb.VkFlushMappedMemoryRanges(
		e.device,
		1,
		pRange.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		memory.Range{
			Base: at.Base,
			Size: uint64(e.stagingBufferSize),
		},
		e.stagingData,
	).AddRead(
		pRange.Data(),
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) unmapMemory() error {
	return e.h.cb.VkUnmapMemory(
		e.device,
		e.stagingMemory,
	).Mutate(
		e.h.ctx, api.CmdNoID, e.h.s, e.h.b, nil,
	)
}

func (e *externalMemoryStaging) stageData() error {
	at, err := e.mapMemory()
	if err != nil {
		return err
	}
	err = e.flushMappedMemory(at)
	if err != nil {
		return err
	}
	return e.unmapMemory()
}

func (e *externalMemoryStaging) updateCall() {
	newSubmitInfos := make([]VkSubmitInfo, 0, len(e.submits)+1)
	hijack := e.h.hijack()
	pStagingCommandBuffer := e.h.mustAllocData(e.stagingCommandBuffer)
	newSubmitInfos = append(newSubmitInfos, NewVkSubmitInfo(
		e.h.s.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                     // pNext
		0,                                             // waitSemaphoreCount
		NewVkSemaphoreᶜᵖ(memory.Nullptr),              // pWaitSemaphores
		NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr), // pWaitDstStageMask
		1, // commandBufferCount
		NewVkCommandBufferᶜᵖ(pStagingCommandBuffer.Ptr()), // pCommandBuffers
		0,                                // signalSemaphoreCount
		NewVkSemaphoreᶜᵖ(memory.Nullptr), // pSignalSemaphores
	))
	hijack.AddRead(pStagingCommandBuffer.Data())
	for _, submit := range e.submits {
		commandBuffers := make([]VkCommandBuffer, 0, 2*len(submit.commandBuffers))
		for _, cmdBuf := range submit.commandBuffers {
			if cmdBuf.stagingCommandBuffer != VkCommandBuffer(0) {
				commandBuffers = append(commandBuffers, cmdBuf.stagingCommandBuffer)
			}
			commandBuffers = append(commandBuffers, cmdBuf.commandBuffer)
		}
		pCommandBuffers := e.h.mustAllocData(commandBuffers)
		submit.submitInfo.SetCommandBufferCount(uint32(len(commandBuffers)))
		submit.submitInfo.SetPCommandBuffers(NewVkCommandBufferᶜᵖ(pCommandBuffers.Ptr()))
		hijack.AddRead(pCommandBuffers.Data())
		newSubmitInfos = append(newSubmitInfos, submit.submitInfo)
	}
	e.h.setSubmitInfos(newSubmitInfos)
}

func (h *vkQueueSubmitHijack) processExternalMemory() error {
	externalData := GetExternalMemoryData(h.origSubmit.Extras())
	if externalData == nil {
		return nil
	}
	staging := externalMemoryStaging{}
	staging.initialize(h, externalData)
	if err := staging.createResources(); err != nil {
		return err
	}
	if err := staging.recordCommandBuffers(); err != nil {
		return err
	}
	if err := staging.stageData(); err != nil {
		return err
	}
	staging.updateCall()
	return nil
}

func init() {
	protoconv.Register(
		func(ctx context.Context, a *ExternalMemoryData) (*vulkan_pb.ExternalMemoryData, error) {
			resIndex, err := id.GetRemapper(ctx).RemapID(ctx, a.ID)
			if err != nil {
				return nil, err
			}

			res := &vulkan_pb.ExternalMemoryData{
				ResIndex: resIndex,
				ResSize:  int64(a.Size),
			}
			for _, b := range a.Buffers {
				res.Buffers = append(res.Buffers, &vulkan_pb.ExternalBufferData{
					Buffer:             uint64(b.Buffer),
					BufferOffset:       uint64(b.BufferOffset),
					DataOffset:         uint64(b.DataOffset),
					Size:               uint64(b.Size),
					SubmitIndex:        b.SubmitIndex,
					CommandBufferIndex: b.CommandBufferIndex,
				})
			}
			return res, nil
		},
		func(ctx context.Context, from *vulkan_pb.ExternalMemoryData) (*ExternalMemoryData, error) {
			id, err := id.GetRemapper(ctx).RemapIndex(ctx, from.ResIndex)
			if err != nil {
				return nil, err
			}
			o := &ExternalMemoryData{
				ID:   id,
				Size: VkDeviceSize(from.ResSize),
			}
			o.Buffers = make([]ExternalBufferData, 0, len(from.Buffers))
			for _, b := range from.Buffers {
				o.Buffers = append(o.Buffers, ExternalBufferData{
					Buffer:             VkBuffer(b.Buffer),
					BufferOffset:       VkDeviceSize(b.BufferOffset),
					DataOffset:         VkDeviceSize(b.DataOffset),
					Size:               VkDeviceSize(b.Size),
					SubmitIndex:        b.SubmitIndex,
					CommandBufferIndex: b.CommandBufferIndex,
				})
			}

			o.Images = make([]ExternalImageData, 0, len(from.Images))
			a := arena.Get(ctx)
			for _, img := range from.Images {
				barrierRange := NewVkImageSubresourceRange(a,
					VkImageAspectFlags(img.AspectMask), // aspectMask
					img.BaseMipLevel,                   // baseMipLevel
					img.LevelCount,                     // levelCount
					img.BaseArrayLayer,                 // baseArrayLayer
					img.LayerCount,                     // layerCount
				)
				copyRanges := make([]ExternalImageDataRange, 0, len(img.Ranges))
				for _, rng := range img.Ranges {
					subresource := NewVkImageSubresourceLayers(a,
						VkImageAspectFlags(rng.AspectMask), // aspectMask
						rng.MipLevel,                       // mipLevel
						rng.BaseArrayLayer,                 // baseArrayLayer
						rng.LayerCount,                     // layerCount
					)
					copyRanges = append(copyRanges, ExternalImageDataRange{
						DataOffset:  VkDeviceSize(rng.DataOffset),
						Subresource: subresource,
					})
				}
				o.Images = append(o.Images, ExternalImageData{
					Image:              VkImage(img.Image),
					BarrierRange:       barrierRange,
					OldLayout:          VkImageLayout(img.OldLayout),
					NewLayout:          VkImageLayout(img.NewLayout),
					SubmitIndex:        img.SubmitIndex,
					CommandBufferIndex: img.CommandBufferIndex,
					CopyRanges:         copyRanges,
				})
			}
			return o, nil
		},
	)
}
