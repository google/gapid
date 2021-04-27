// Copyright (C) 2021 Google Inc.
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
	"bytes"
	"context"
	"errors"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

// externalMemory is a transform that will transform commands using external
// memory to internal memory, so that they can be replayed correctly.
type externalMemory struct {
	allocations    *allocationTracker
	externalImages map[VkImage]struct{}
}

func newExternalMemory() *externalMemory {
	return &externalMemory{
		externalImages: map[VkImage]struct{}{},
	}
}

func (e *externalMemory) RequiresAccurateState() bool {
	return false
}

func (e *externalMemory) RequiresInnerStateMutation() bool {
	return false
}

func (e *externalMemory) SetInnerStateMutationFunction(sm transform.StateMutator) {
	// This transform does not require inner state mutation.
}

func (e *externalMemory) BeginTransform(ctx context.Context, g *api.GlobalState) error {
	e.allocations = NewAllocationTracker(g)
	return nil
}

func (e *externalMemory) EndTransform(ctx context.Context, g *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (e *externalMemory) ClearTransformResources(ctx context.Context) {
	e.allocations.FreeAllocations()
}

func (e *externalMemory) TransformCommand(ctx context.Context, id transform.CommandID, cmds []api.Cmd, g *api.GlobalState) ([]api.Cmd, error) {
	var err error
	var out []api.Cmd
	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case *VkCreateImage:
			cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
			cmd, err = e.vkCreateImage(ctx, g, c)
		case *VkCreateImageView:
			cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
			cmd, err = e.vkCreateImageView(ctx, g, c)
		case *VkAllocateMemory:
			cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
			cmd, err = e.vkAllocateMemory(ctx, g, c)
		case *VkGetPhysicalDeviceImageFormatProperties2:
			// TODO: figure out why this crashes the replay.
			cmd = nil
		case *VkCmdPipelineBarrier:
			cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
			cmd, err = e.vkCmdPipelineBarrier(ctx, g, c)
		}

		if err != nil {
			return nil, err
		} else if cmd != nil {
			out = append(out, cmd)
		}
	}
	return out, nil
}

func (e *externalMemory) vkCreateImage(ctx context.Context, g *api.GlobalState, cmd *VkCreateImage) (api.Cmd, error) {
	newCmd, updated, err := filterPNext(ctx, g, cmd, NewVulkanStructHeaderᵖ(cmd.PCreateInfo()), func(h VulkanStructHeader) bool {
		s := h.SType()
		return s == VkStructureType_VK_STRUCTURE_TYPE_EXTERNAL_MEMORY_IMAGE_CREATE_INFO ||
			s == VkStructureType_VK_STRUCTURE_TYPE_EXTERNAL_FORMAT_ANDROID
	})
	if err != nil || !updated {
		return cmd, err
	}

	cmd = newCmd.(*VkCreateImage)

	// Apply our override, so we don't undo it.
	cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
	info, err := cmd.PCreateInfo().Read(ctx, cmd, g, nil)
	if err != nil {
		return cmd, err
	}
	if info.Fmt() == VkFormat_VK_FORMAT_UNDEFINED {
		// The spec allows undefined "external" formats. Change it to RGBA.
		info.SetFmt(VkFormat_VK_FORMAT_R8G8B8A8_UNORM)
		addOverrideObservation(ctx, g, cmd, cmd.PCreateInfo(), info)

		cmd.Extras().Observations().ApplyWrites(g.Memory.ApplicationPool())
		image, err := cmd.PImage().Read(ctx, cmd, g, nil)
		if err != nil {
			return cmd, err
		}
		e.externalImages[image] = struct{}{}
	}

	return cmd, nil
}

func (e *externalMemory) vkCreateImageView(ctx context.Context, g *api.GlobalState, cmd *VkCreateImageView) (api.Cmd, error) {
	info, err := cmd.PCreateInfo().Read(ctx, cmd, g, nil)
	if err != nil {
		return cmd, err
	}
	if _, ok := e.externalImages[info.Image()]; ok {
		// This is an image that was created with an external format, that we turned
		// into an RGBA image. Remove the conversion sampler info from the image view.
		newCmd, _, err := filterPNext(ctx, g, cmd, NewVulkanStructHeaderᵖ(cmd.PCreateInfo()), func(h VulkanStructHeader) bool {
			return h.SType() == VkStructureType_VK_STRUCTURE_TYPE_SAMPLER_YCBCR_CONVERSION_INFO
		})
		if err != nil {
			return cmd, err
		}

		cmd = newCmd.(*VkCreateImageView)

		// Apply our override, so we don't undo it and re-read the info.
		cmd.Extras().Observations().ApplyReads(g.Memory.ApplicationPool())
		if info, err = cmd.PCreateInfo().Read(ctx, cmd, g, nil); err != nil {
			return cmd, err
		}
		info.SetFmt(VkFormat_VK_FORMAT_R8G8B8A8_UNORM)
		addOverrideObservation(ctx, g, cmd, cmd.PCreateInfo(), info)
	}
	return cmd, nil
}

func (e *externalMemory) vkAllocateMemory(ctx context.Context, g *api.GlobalState, cmd *VkAllocateMemory) (api.Cmd, error) {
	found := false
	var buffer VkBuffer
	var image VkImage
	err := forEachPNext(ctx, g, cmd, NewVulkanStructHeaderᵖ(cmd.PAllocateInfo()), func(ptr VulkanStructHeaderᵖ, h VulkanStructHeader) error {
		switch h.SType() {
		case VkStructureType_VK_STRUCTURE_TYPE_IMPORT_ANDROID_HARDWARE_BUFFER_INFO_ANDROID:
			found = true
		case VkStructureType_VK_STRUCTURE_TYPE_MEMORY_DEDICATED_ALLOCATE_INFO_KHR:
			ext, err := VkMemoryDedicatedAllocationInfoKHRᵖ(ptr).Read(ctx, cmd, g, nil)
			if err != nil {
				return err
			}
			buffer = ext.Buffer()
			image = ext.Image()
		}
		return nil
	})
	if err != nil || !found {
		return cmd, err
	}

	if buffer != 0 {
		// TODO confirm that buffer size is ok here.
		newCmd, _, err := filterPNext(ctx, g, cmd, NewVulkanStructHeaderᵖ(cmd.PAllocateInfo()), func(h VulkanStructHeader) bool {
			return h.SType() == VkStructureType_VK_STRUCTURE_TYPE_IMPORT_ANDROID_HARDWARE_BUFFER_INFO_ANDROID
		})
		return newCmd, err
	} else if image != 0 {
		// Replace the VkAllocateMemory command with the synthetic replayAllocateImageMemory command.
		s := GetState(g)
		device := s.Devices().Get(cmd.Device())
		physDevice := s.PhysicalDevices().Get(device.PhysicalDevice())

		memProps := e.allocations.AllocDataOrPanic(ctx, physDevice.MemoryProperties())

		cb := CommandBuilder{Thread: cmd.Thread()}
		newCmd := cb.ReplayAllocateImageMemory(
			cmd.Device(),
			memProps.Ptr(),
			image,
			cmd.PMemory(),
			VkResult_VK_SUCCESS,
		).AddRead(
			memProps.Data(),
		)

		// Copy the write observations over.
		oldObs := cmd.Extras().Observations()
		newObs := newCmd.Extras().Observations()
		newObs.Writes = append(newObs.Writes, oldObs.Writes...)

		return newCmd, nil
	} else {
		// This is against the spec, AHB allocations have to be dedicated.
		return cmd, errors.New("Non dedicated AHB backed allocation")
	}
}

func (e *externalMemory) vkCmdPipelineBarrier(ctx context.Context, g *api.GlobalState, cmd *VkCmdPipelineBarrier) (api.Cmd, error) {
	buffersToFix := map[memory.Pointer]VkBufferMemoryBarrier{}
	if numBuffers := cmd.BufferMemoryBarrierCount(); numBuffers > 0 {
		base := cmd.PBufferMemoryBarriers()
		buffers, err := base.Slice(0, uint64(numBuffers), g.MemoryLayout).Read(ctx, cmd, g, nil)
		if err != nil {
			return cmd, err
		}

		for i, buffer := range buffers {
			if isExternalQueue(buffer.SrcQueueFamilyIndex()) || isExternalQueue(buffer.DstQueueFamilyIndex()) {
				buffersToFix[base.Offset(uint64(i)*base.ElementSize(g.MemoryLayout))] = buffer
			}
		}
	}

	imagesToFix := map[memory.Pointer]VkImageMemoryBarrier{}
	if numImages := cmd.ImageMemoryBarrierCount(); numImages > 0 {
		base := cmd.PImageMemoryBarriers()
		images, err := base.Slice(0, uint64(numImages), g.MemoryLayout).Read(ctx, cmd, g, nil)
		if err != nil {
			return cmd, err
		}

		for i, image := range images {
			if isExternalQueue(image.SrcQueueFamilyIndex()) || isExternalQueue(image.DstQueueFamilyIndex()) {
				imagesToFix[base.Offset(uint64(i)*base.ElementSize(g.MemoryLayout))] = image
			}
		}
	}

	if len(buffersToFix) == 0 && len(imagesToFix) == 0 {
		return cmd, nil
	}

	newCmd := cmd.Clone()
	newCmd.Extras().CloneObservations()

	for ptr, buffer := range buffersToFix {
		if isExternalQueue(buffer.SrcQueueFamilyIndex()) {
			buffer.SetSrcQueueFamilyIndex(buffer.DstQueueFamilyIndex())
		} else {
			buffer.SetDstQueueFamilyIndex(buffer.SrcQueueFamilyIndex())
		}
		if err := addOverrideObservation(ctx, g, newCmd, ptr, buffer); err != nil {
			return cmd, err
		}
	}

	for ptr, image := range imagesToFix {
		if isExternalQueue(image.SrcQueueFamilyIndex()) {
			image.SetSrcQueueFamilyIndex(image.DstQueueFamilyIndex())
		} else {
			image.SetDstQueueFamilyIndex(image.SrcQueueFamilyIndex())
		}
		if err := addOverrideObservation(ctx, g, newCmd, ptr, image); err != nil {
			return cmd, err
		}
	}

	return newCmd, nil
}

// forEachPNext loops over the pNext chain starting at root and calls the given callback for each node.
func forEachPNext(ctx context.Context, g *api.GlobalState, cmd api.Cmd, root VulkanStructHeaderᵖ, cb func(ptr VulkanStructHeaderᵖ, h VulkanStructHeader) error) error {
	for node := root; !node.IsNullptr(); {
		header, err := node.Read(ctx, cmd, g, nil)
		if err != nil {
			return err
		}
		if err = cb(node, header); err != nil {
			return err
		}
		node = VulkanStructHeaderᵖ(header.PNext())
	}
	return nil
}

// filterPNext removes any node from the pNext chain where the filter callback returned true.
// Returns a cloned command and true, if a change was made, otherwhise the original command and false.
func filterPNext(ctx context.Context, g *api.GlobalState, cmd api.Cmd, root VulkanStructHeaderᵖ, filter func(h VulkanStructHeader) bool) (api.Cmd, bool, error) {
	prev := VulkanStructHeaderᵖ(0)
	prevHeader := VulkanStructHeader{}
	newNext := map[VulkanStructHeaderᵖ]VulkanStructHeader{}
	err := forEachPNext(ctx, g, cmd, root, func(ptr VulkanStructHeaderᵖ, header VulkanStructHeader) error {
		if ptr != root && filter(header) {
			prevHeader.SetPNext(header.PNext())
			newNext[prev] = prevHeader
		} else {
			prev = ptr
			prevHeader = header
		}
		return nil
	})
	if err != nil {
		return cmd, false, err
	}

	if len(newNext) == 0 {
		return cmd, false, nil
	}

	newCmd := cmd.Clone()
	newCmd.Extras().CloneObservations()
	for ptr, header := range newNext {
		// Create a new memory blob that will override the previous struct header.
		if err := addOverrideObservation(ctx, g, newCmd, ptr, header); err != nil {
			return cmd, false, err
		}
	}

	return newCmd, true, nil
}

func addOverrideObservation(ctx context.Context, g *api.GlobalState, cmd api.Cmd, ptr memory.Pointer, v interface{}) error {
	buf := &bytes.Buffer{}
	enc := memory.NewEncoder(endian.Writer(buf, g.MemoryLayout.GetEndian()), g.MemoryLayout)
	memory.Write(enc, v)
	id, err := database.Store(ctx, buf.Bytes())
	if err != nil {
		return err
	}

	cmd.Extras().GetOrAppendObservations().AddRead(
		memory.Range{Base: ptr.Address(), Size: uint64(len(buf.Bytes()))}, id)
	return nil
}

func isExternalQueue(qf uint32) bool {
	return qf == VK_QUEUE_FAMILY_EXTERNAL || qf == VK_QUEUE_FAMILY_FOREIGN_EXT
}
