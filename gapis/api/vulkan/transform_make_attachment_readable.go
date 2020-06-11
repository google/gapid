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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

type makeAttachmentReadable2 struct {
	imagesOnly  bool
	allocations *allocationTracker
}

func newMakeAttachmentReadable2(imagesOnly bool) *makeAttachmentReadable2 {
	return &makeAttachmentReadable2{
		imagesOnly:  imagesOnly,
		allocations: nil,
	}
}

func (attachmentTransform *makeAttachmentReadable2) RequiresAccurateState() bool {
	return false
}

func (attachmentTransform *makeAttachmentReadable2) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	attachmentTransform.allocations = NewAllocationTracker(inputState)
	return inputCommands, nil
}

func (attachmentTransform *makeAttachmentReadable2) EndTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (attachmentTransform *makeAttachmentReadable2) ClearTransformResources(ctx context.Context) {
	// Melih TODO: This transform seems to be never releasing the allocations
	// Check if it's intended
	// b/158597615
	// attachmentTransform.allocations.FreeAllocations()
}

func (attachmentTransform *makeAttachmentReadable2) TransformCommand(ctx context.Context, id api.CmdID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for i, cmd := range inputCommands {
		cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

		var modifiedCmd api.Cmd
		modifiedCmd = nil

		if createImageCmd, ok := cmd.(*VkCreateImage); ok {
			modifiedCmd = attachmentTransform.makeImageReadable(ctx, inputState, createImageCmd)
		} else if createSwapchainCmd, ok := cmd.(*VkCreateSwapchainKHR); ok {
			modifiedCmd = attachmentTransform.makeSwapchainReadable(ctx, inputState, createSwapchainCmd)
		} else if createRenderPassCmd, ok := cmd.(*VkCreateRenderPass); ok && !attachmentTransform.imagesOnly {
			modifiedCmd = attachmentTransform.makeRenderPassReadable(ctx, inputState, createRenderPassCmd)
		} else if enumeratePhysicalDevicesCmd, ok := cmd.(*VkEnumeratePhysicalDevices); ok && !attachmentTransform.imagesOnly {
			modifiedCmd = attachmentTransform.makePhysicalDevicesReadable(ctx, inputState, id, enumeratePhysicalDevicesCmd)
		} else if createBufferCmd, ok := cmd.(*VkCreateBuffer); ok {
			modifiedCmd = attachmentTransform.makeBufferReadable(ctx, inputState, createBufferCmd)
		}

		if modifiedCmd != nil {
			inputCommands[i] = modifiedCmd
		}
	}

	return inputCommands, nil
}

func (attachmentTransform *makeAttachmentReadable2) makeImageReadable(ctx context.Context, inputState *api.GlobalState, createImageCmd *VkCreateImage) api.Cmd {
	pinfo := createImageCmd.PCreateInfo()
	info := pinfo.MustRead(ctx, createImageCmd, inputState, nil)

	newUsage, changed := patchImageUsage2(info.Usage())
	if !changed {
		return nil
	}

	device := createImageCmd.Device()
	palloc := memory.Pointer(createImageCmd.PAllocator())
	pimage := memory.Pointer(createImageCmd.PImage())
	result := createImageCmd.Result()

	info.SetUsage(newUsage)
	newInfo := attachmentTransform.allocations.AllocDataOrPanic(ctx, info)
	cb := CommandBuilder{Thread: createImageCmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateImage(device, newInfo.Ptr(), palloc, pimage, result)

	// Carry all non-observation extras through.
	for _, e := range createImageCmd.Extras().All() {
		if _, ok := e.(*api.CmdObservations); !ok {
			newCmd.Extras().Add(e)
		}
	}

	// Carry observations through. We cannot merge these code with the
	// above code for handling extras together since we'd like to change
	// the observations, which are slices.
	observations := createImageCmd.Extras().Observations()
	for _, r := range observations.Reads {
		// TODO: filter out the old VkImageCreateInfo. That should be done via
		// creating new observations for data we are interested from t.state.
		newCmd.AddRead(r.Range, r.ID)
	}
	// Use our new VkImageCreateInfo.
	newCmd.AddRead(newInfo.Data())
	for _, w := range observations.Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}

func (attachmentTransform *makeAttachmentReadable2) makeSwapchainReadable(ctx context.Context, inputState *api.GlobalState, createSwapchainCmd *VkCreateSwapchainKHR) api.Cmd {
	pinfo := createSwapchainCmd.PCreateInfo()
	info := pinfo.MustRead(ctx, createSwapchainCmd, inputState, nil)

	newUsage, changed := patchImageUsage2(info.ImageUsage())
	if !changed {
		return nil
	}

	device := createSwapchainCmd.Device()
	palloc := memory.Pointer(createSwapchainCmd.PAllocator())
	pswapchain := memory.Pointer(createSwapchainCmd.PSwapchain())
	result := createSwapchainCmd.Result()

	info.SetImageUsage(newUsage)
	newInfo := attachmentTransform.allocations.AllocDataOrPanic(ctx, info)
	cb := CommandBuilder{Thread: createSwapchainCmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateSwapchainKHR(device, newInfo.Ptr(), palloc, pswapchain, result)
	for _, e := range createSwapchainCmd.Extras().All() {
		if _, ok := e.(*api.CmdObservations); !ok {
			newCmd.Extras().Add(e)
		}
	}

	observations := createSwapchainCmd.Extras().Observations()
	for _, r := range observations.Reads {
		// TODO: filter out the old VkSwapchainCreateInfoKHR. That should be done via
		// creating new observations for data we are interested from t.state.
		newCmd.AddRead(r.Range, r.ID)
	}
	newCmd.AddRead(newInfo.Data())
	for _, w := range observations.Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}

func (attachmentTransform *makeAttachmentReadable2) makeRenderPassReadable(ctx context.Context, inputState *api.GlobalState, createRenderPassCmd *VkCreateRenderPass) api.Cmd {
	pInfo := createRenderPassCmd.PCreateInfo()
	info := pInfo.MustRead(ctx, createRenderPassCmd, inputState, nil)

	layout := inputState.MemoryLayout
	pAttachments := info.PAttachments()
	attachments := pAttachments.Slice(0, uint64(info.AttachmentCount()), layout).MustRead(ctx, createRenderPassCmd, inputState, nil)
	changed := false
	for i := range attachments {
		if attachments[i].StoreOp() == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE {
			changed = true
			attachments[i].SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
		}
	}

	if !changed {
		return nil
	}

	// Build new attachments data, new create info and new command
	newAttachments := attachmentTransform.allocations.AllocDataOrPanic(ctx, attachments)
	info.SetPAttachments(NewVkAttachmentDescriptionᶜᵖ(newAttachments.Ptr()))
	newInfo := attachmentTransform.allocations.AllocDataOrPanic(ctx, info)
	cb := CommandBuilder{Thread: createRenderPassCmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateRenderPass(createRenderPassCmd.Device(),
		newInfo.Ptr(),
		memory.Pointer(createRenderPassCmd.PAllocator()),
		memory.Pointer(createRenderPassCmd.PRenderPass()),
		createRenderPassCmd.Result())

	// Add back the extras and read/write observations
	for _, e := range createRenderPassCmd.Extras().All() {
		if _, ok := e.(*api.CmdObservations); !ok {
			newCmd.Extras().Add(e)
		}
	}

	for _, r := range createRenderPassCmd.Extras().Observations().Reads {
		newCmd.AddRead(r.Range, r.ID)
	}
	newCmd.AddRead(newInfo.Data()).AddRead(newAttachments.Data())
	for _, w := range createRenderPassCmd.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}

func buildReplayEnumeratePhysicalDevices2(
	ctx context.Context, s *api.GlobalState, cb CommandBuilder, instance VkInstance,
	count uint32, devices []VkPhysicalDevice,
	propertiesInOrder []VkPhysicalDeviceProperties) *ReplayEnumeratePhysicalDevices {
	numDevData := s.AllocDataOrPanic(ctx, count)
	phyDevData := s.AllocDataOrPanic(ctx, devices)
	dids := make([]uint64, 0)
	for i := uint32(0); i < count; i++ {
		dids = append(dids, uint64(
			propertiesInOrder[i].VendorID())<<32|
			uint64(propertiesInOrder[i].DeviceID()))
	}
	devIDData := s.AllocDataOrPanic(ctx, dids)
	return cb.ReplayEnumeratePhysicalDevices(
		instance, numDevData.Ptr(), phyDevData.Ptr(), devIDData.Ptr(),
		VkResult_VK_SUCCESS).AddRead(
		numDevData.Data()).AddRead(phyDevData.Data()).AddRead(devIDData.Data())
}

func (attachmentTransform *makeAttachmentReadable2) makePhysicalDevicesReadable(ctx context.Context, inputState *api.GlobalState, id api.CmdID, enumeratePhysicalDeviceCmd *VkEnumeratePhysicalDevices) api.Cmd {
	if enumeratePhysicalDeviceCmd.PPhysicalDevices() == 0 {
		// Querying for the number of devices.
		// No changes needed here.
		return nil
	}

	layout := inputState.MemoryLayout
	enumeratePhysicalDeviceCmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
	numDev := enumeratePhysicalDeviceCmd.PPhysicalDeviceCount().Slice(0, 1, layout).MustRead(ctx, enumeratePhysicalDeviceCmd, inputState, nil)[0]
	devSlice := enumeratePhysicalDeviceCmd.PPhysicalDevices().Slice(0, uint64(numDev), layout)
	devs := devSlice.MustRead(ctx, enumeratePhysicalDeviceCmd, inputState, nil)
	allProps := externs{ctx, enumeratePhysicalDeviceCmd, id, inputState, nil, nil}.fetchPhysicalDeviceProperties(enumeratePhysicalDeviceCmd.Instance(), devSlice)

	propList := []VkPhysicalDeviceProperties{}
	for _, dev := range devs {
		propList = append(propList, allProps.PhyDevToProperties().Get(dev).Clone(inputState.Arena, api.CloneContext{}))
	}

	cb := CommandBuilder{Thread: enumeratePhysicalDeviceCmd.Thread(), Arena: inputState.Arena}
	newCmd := buildReplayEnumeratePhysicalDevices2(ctx, inputState, cb, enumeratePhysicalDeviceCmd.Instance(), numDev, devs, propList)
	for _, extra := range enumeratePhysicalDeviceCmd.Extras().All() {
		newCmd.Extras().Add(extra)
	}
	return newCmd
}

func (attachmentTransform *makeAttachmentReadable2) makeBufferReadable(ctx context.Context, inputState *api.GlobalState, createBufferCmd *VkCreateBuffer) api.Cmd {
	pinfo := createBufferCmd.PCreateInfo()
	info := pinfo.MustRead(ctx, createBufferCmd, inputState, nil)

	newUsage, changed := patchBufferUsage2(info.Usage())
	if !changed {
		return nil
	}

	info.SetUsage(newUsage)
	newInfo := attachmentTransform.allocations.AllocDataOrPanic(ctx, info)
	cb := CommandBuilder{Thread: createBufferCmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateBuffer(
		createBufferCmd.Device(),
		newInfo.Ptr(),
		createBufferCmd.PAllocator(),
		createBufferCmd.PBuffer(),
		createBufferCmd.Result())

	for _, e := range createBufferCmd.Extras().All() {
		if _, ok := e.(*api.CmdObservations); !ok {
			newCmd.Extras().Add(e)
		}
	}

	observations := createBufferCmd.Extras().Observations()
	for _, r := range observations.Reads {
		newCmd.AddRead(r.Range, r.ID)
	}
	newCmd.AddRead(newInfo.Data())
	for _, w := range observations.Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}

// color/depth/stencil attachment bit.
func patchImageUsage2(usage VkImageUsageFlags) (VkImageUsageFlags, bool) {
	hasBit := func(flag VkImageUsageFlags, bit VkImageUsageFlagBits) bool {
		return (uint32(flag) & uint32(bit)) == uint32(bit)
	}

	if hasBit(usage, VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT) ||
		hasBit(usage, VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT) {
		return VkImageUsageFlags(uint32(usage) | uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT)), true
	}
	return usage, false
}

// Add VK_BUFFER_USAGE_TRANSFER_SRC_BIT to the buffer usage bit.
// TODO(renfeng) using shader to do the copy instead of change the usage bit.
func patchBufferUsage2(usage VkBufferUsageFlags) (VkBufferUsageFlags, bool) {
	hasBit := func(flag VkBufferUsageFlags, bit VkBufferUsageFlagBits) bool {
		return (uint32(flag) & uint32(bit)) == uint32(bit)
	}

	if hasBit(usage, VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT) {
		return usage, false
	}

	return VkBufferUsageFlags(uint32(usage) | uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT)), true
}
