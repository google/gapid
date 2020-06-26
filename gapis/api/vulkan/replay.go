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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/commandGenerator"
	"github.com/google/gapid/gapis/api/controlFlowGenerator"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/trace"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(API{})
	_ = replay.QueryFramebufferAttachment(API{})
	_ = replay.Support(API{})
	_ = replay.QueryTimestamps(API{})
	_ = replay.Profiler(API{})
)

// GetReplayPriority returns a uint32 representing the preference for
// replaying this trace on the given device.
// A lower number represents a higher priority, and Zero represents
// an inability for the trace to be replayed on the given device.
func (a API) GetReplayPriority(ctx context.Context, i *device.Instance, h *capture.Header) uint32 {
	devConf := i.GetConfiguration()
	devAbis := devConf.GetABIs()
	devVkDriver := devConf.GetDrivers().GetVulkan()
	traceVkDriver := h.GetDevice().GetConfiguration().GetDrivers().GetVulkan()

	if traceVkDriver == nil {
		log.E(ctx, "Vulkan trace does not contain VulkanDriver info.")
		return 0
	}

	// The device does not support Vulkan
	if devVkDriver == nil {
		return 0
	}

	for _, abi := range devAbis {
		// Memory layout must match.
		if !abi.GetMemoryLayout().SameAs(h.GetABI().GetMemoryLayout()) {
			continue
		}
		// If there is no physical devices, the trace must not contain
		// vkCreateInstance, any ABI compatible Vulkan device should be able to
		// replay.
		if len(traceVkDriver.GetPhysicalDevices()) == 0 {
			return 1
		}
		// Requires same vendor, device and version of API.
		for _, devPhyInfo := range devVkDriver.GetPhysicalDevices() {
			for _, tracePhyInfo := range traceVkDriver.GetPhysicalDevices() {
				// TODO: More sophisticated rules
				if devPhyInfo.GetVendorId() != tracePhyInfo.GetVendorId() {
					continue
				}
				if devPhyInfo.GetDeviceId() != tracePhyInfo.GetDeviceId() {
					continue
				}
				// Ignore the API patch level (bottom 12 bits) when comparing the API version.
				if (devPhyInfo.GetApiVersion() & ^uint32(0xfff)) != (tracePhyInfo.GetApiVersion() & ^uint32(0xfff)) {
					continue
				}
				return 1
			}
		}
	}
	return 0
}

// makeAttachementReadable is a transformation marking all color/depth/stencil
// attachment images created via vkCreateImage commands as readable (by patching
// the transfer src bit).
type makeAttachementReadable struct {
	imagesOnly bool
}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	startScope                api.CmdID
	endScope                  api.CmdID
	subindices                string // drawConfig needs to be comparable, so we cannot use a slice
	drawMode                  path.DrawMode
	disableReplayOptimization bool
}

type imgRes struct {
	img *image.Data // The image data.
	err error       // The error that occurred generating the image.
}

// framebufferRequest requests a postback of a framebuffer's attachment.
type framebufferRequest struct {
	after            []uint64
	width, height    uint32
	attachment       api.FramebufferAttachmentType
	framebufferIndex uint32
	out              chan imgRes
	wireframeOverlay bool
	displayToSurface bool
}

// color/depth/stencil attachment bit.
func patchImageUsage(usage VkImageUsageFlags) (VkImageUsageFlags, bool) {
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
func patchBufferusage(usage VkBufferUsageFlags) (VkBufferUsageFlags, bool) {
	hasBit := func(flag VkBufferUsageFlags, bit VkBufferUsageFlagBits) bool {
		return (uint32(flag) & uint32(bit)) == uint32(bit)
	}

	if hasBit(usage, VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT) {
		return usage, false
	}

	return VkBufferUsageFlags(uint32(usage) | uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT)), true
}

func (t *makeAttachementReadable) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	s := out.State()
	l := s.MemoryLayout
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
	cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

	if image, ok := cmd.(*VkCreateImage); ok {
		pinfo := image.PCreateInfo()
		info := pinfo.MustRead(ctx, image, s, nil)

		if newUsage, changed := patchImageUsage(info.Usage()); changed {
			device := image.Device()
			palloc := memory.Pointer(image.PAllocator())
			pimage := memory.Pointer(image.PImage())
			result := image.Result()

			info.SetUsage(newUsage)
			newInfo := s.AllocDataOrPanic(ctx, info)
			newCmd := cb.VkCreateImage(device, newInfo.Ptr(), palloc, pimage, result)
			// Carry all non-observation extras through.
			for _, e := range image.Extras().All() {
				if _, ok := e.(*api.CmdObservations); !ok {
					newCmd.Extras().Add(e)
				}
			}
			// Carry observations through. We cannot merge these code with the
			// above code for handling extras together since we'd like to change
			// the observations, which are slices.
			observations := image.Extras().Observations()
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
			return out.MutateAndWrite(ctx, id, newCmd)
		}
	} else if swapchain, ok := cmd.(*VkCreateSwapchainKHR); ok {
		pinfo := swapchain.PCreateInfo()
		info := pinfo.MustRead(ctx, swapchain, s, nil)

		if newUsage, changed := patchImageUsage(info.ImageUsage()); changed {
			device := swapchain.Device()
			palloc := memory.Pointer(swapchain.PAllocator())
			pswapchain := memory.Pointer(swapchain.PSwapchain())
			result := swapchain.Result()

			info.SetImageUsage(newUsage)
			newInfo := s.AllocDataOrPanic(ctx, info)
			newCmd := cb.VkCreateSwapchainKHR(device, newInfo.Ptr(), palloc, pswapchain, result)
			for _, e := range swapchain.Extras().All() {
				if _, ok := e.(*api.CmdObservations); !ok {
					newCmd.Extras().Add(e)
				}
			}
			observations := swapchain.Extras().Observations()
			for _, r := range observations.Reads {
				// TODO: filter out the old VkSwapchainCreateInfoKHR. That should be done via
				// creating new observations for data we are interested from t.state.
				newCmd.AddRead(r.Range, r.ID)
			}
			newCmd.AddRead(newInfo.Data())
			for _, w := range observations.Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			return out.MutateAndWrite(ctx, id, newCmd)
		}
	} else if createRenderPass, ok := cmd.(*VkCreateRenderPass); ok && !t.imagesOnly {
		pInfo := createRenderPass.PCreateInfo()
		info := pInfo.MustRead(ctx, createRenderPass, s, nil)
		pAttachments := info.PAttachments()
		attachments := pAttachments.Slice(0, uint64(info.AttachmentCount()), l).MustRead(ctx, createRenderPass, s, nil)
		changed := false
		for i := range attachments {
			if attachments[i].StoreOp() == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE {
				changed = true
				attachments[i].SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
			}
		}
		// Returns if no attachment description needs to be changed
		if !changed {
			return out.MutateAndWrite(ctx, id, cmd)
		}
		// Build new attachments data, new create info and new command
		newAttachments := s.AllocDataOrPanic(ctx, attachments)
		info.SetPAttachments(NewVkAttachmentDescriptionᶜᵖ(newAttachments.Ptr()))
		newInfo := s.AllocDataOrPanic(ctx, info)
		newCmd := cb.VkCreateRenderPass(createRenderPass.Device(),
			newInfo.Ptr(),
			memory.Pointer(createRenderPass.PAllocator()),
			memory.Pointer(createRenderPass.PRenderPass()),
			createRenderPass.Result())
		// Add back the extras and read/write observations
		for _, e := range createRenderPass.Extras().All() {
			if _, ok := e.(*api.CmdObservations); !ok {
				newCmd.Extras().Add(e)
			}
		}
		for _, r := range createRenderPass.Extras().Observations().Reads {
			newCmd.AddRead(r.Range, r.ID)
		}
		newCmd.AddRead(newInfo.Data()).AddRead(newAttachments.Data())
		for _, w := range createRenderPass.Extras().Observations().Writes {
			newCmd.AddWrite(w.Range, w.ID)
		}
		return out.MutateAndWrite(ctx, id, newCmd)
	} else if e, ok := cmd.(*VkEnumeratePhysicalDevices); ok && !t.imagesOnly {
		if e.PPhysicalDevices() == 0 {
			// Querying for the number of devices.
			// No changes needed here.
			return out.MutateAndWrite(ctx, id, cmd)
		}
		l := s.MemoryLayout
		cmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
		numDev := e.PPhysicalDeviceCount().Slice(0, 1, l).MustRead(ctx, cmd, s, nil)[0]
		devSlice := e.PPhysicalDevices().Slice(0, uint64(numDev), l)
		devs := devSlice.MustRead(ctx, cmd, s, nil)
		allProps := externs{ctx, cmd, id, s, nil, nil}.fetchPhysicalDeviceProperties(e.Instance(), devSlice)
		propList := []VkPhysicalDeviceProperties{}
		for _, dev := range devs {
			propList = append(propList, allProps.PhyDevToProperties().Get(dev).Clone(s.Arena, api.CloneContext{}))
		}
		newEnumerate := buildReplayEnumeratePhysicalDevices(ctx, s, cb, e.Instance(), numDev, devs, propList)
		for _, e := range cmd.Extras().All() {
			newEnumerate.Extras().Add(e)
		}
		return out.MutateAndWrite(ctx, id, newEnumerate)
	} else if buffer, ok := cmd.(*VkCreateBuffer); ok {
		pinfo := buffer.PCreateInfo()
		info := pinfo.MustRead(ctx, buffer, s, nil)

		if newUsage, changed := patchBufferusage(info.Usage()); changed {

			info.SetUsage(newUsage)
			newInfo := s.AllocDataOrPanic(ctx, info)
			newCmd := cb.VkCreateBuffer(buffer.Device(), newInfo.Ptr(), buffer.PAllocator(), buffer.PBuffer(), buffer.Result())
			for _, e := range buffer.Extras().All() {
				if _, ok := e.(*api.CmdObservations); !ok {
					newCmd.Extras().Add(e)
				}
			}

			observations := buffer.Extras().Observations()
			for _, r := range observations.Reads {
				newCmd.AddRead(r.Range, r.ID)
			}
			newCmd.AddRead(newInfo.Data())
			for _, w := range observations.Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}
			return out.MutateAndWrite(ctx, id, newCmd)
		}
	}
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *makeAttachementReadable) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *makeAttachementReadable) PreLoop(ctx context.Context, out transform.Writer)     {}
func (t *makeAttachementReadable) PostLoop(ctx context.Context, out transform.Writer)    {}
func (t *makeAttachementReadable) BuffersCommands() bool                                 { return false }

func buildReplayEnumeratePhysicalDevices(
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

// dropInvalidDestroy is a transformation that drops all VkDestroyXXX commands
// whose destroying targets are not recorded in the state.
type dropInvalidDestroy struct{ tag string }

func (t *dropInvalidDestroy) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	s := out.State()
	l := s.MemoryLayout
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
	warnDropCmd := func(handles ...interface{}) {
		log.W(ctx, "[%v] Dropping [%d]:%v because the creation of %v was not recorded", t.tag, id, cmd, handles)
	}
	warnModCmd := func(handles ...interface{}) {
		log.W(ctx, "[%v] Modifing [%d]:%v to remove the reference to %v because the creation of them were not recorded", t.tag, id, cmd, handles)
	}
	switch cmd := cmd.(type) {
	case *VkDestroyInstance:
		if !GetState(s).Instances().Contains(cmd.Instance()) {
			warnDropCmd(cmd.Instance())
			return nil
		}
	case *VkDestroyDevice:
		if !GetState(s).Devices().Contains(cmd.Device()) {
			warnDropCmd(cmd.Device())
			return nil
		}
	case *VkFreeCommandBuffers:
		cmdBufCount := cmd.CommandBufferCount()
		if cmdBufCount > 0 {
			cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
			cmdBufs := cmd.PCommandBuffers().Slice(0, uint64(cmdBufCount), l).MustRead(ctx, cmd, s, nil)
			newCmdBufs := []VkCommandBuffer{}
			dropped := []VkCommandBuffer{}
			for _, b := range cmdBufs {
				if GetState(s).CommandBuffers().Contains(b) {
					newCmdBufs = append(newCmdBufs, b)
				} else {
					dropped = append(dropped, b)
				}
			}
			if len(newCmdBufs) == 0 {
				// no need to have this command
				warnDropCmd(cmdBufs)
				return nil
			}
			if len(newCmdBufs) != len(cmdBufs) {
				// need to modify the command to drop the command buffers not
				// in the state out of the command
				warnModCmd(dropped)
				newCmdBufsData := s.AllocDataOrPanic(ctx, newCmdBufs)
				defer newCmdBufsData.Free()
				newCmd := cb.VkFreeCommandBuffers(cmd.Device(), cmd.CommandPool(), uint32(len(newCmdBufs)), newCmdBufsData.Ptr()).AddRead(newCmdBufsData.Data())
				return out.MutateAndWrite(ctx, id, newCmd)
			}
			// No need to modify the command, just mutate and write the command
			// like others.
		}
	case *VkFreeMemory:
		if !GetState(s).DeviceMemories().Contains(cmd.Memory()) {
			warnDropCmd(cmd.Memory())
			return nil
		}
	case *VkDestroyBuffer:
		if !GetState(s).Buffers().Contains(cmd.Buffer()) {
			warnDropCmd(cmd.Buffer())
			return nil
		}
	case *VkDestroyBufferView:
		if !GetState(s).BufferViews().Contains(cmd.BufferView()) {
			warnDropCmd(cmd.BufferView())
			return nil
		}
	case *VkDestroyImage:
		if !GetState(s).Images().Contains(cmd.Image()) {
			warnDropCmd(cmd.Image())
			return nil
		}
	case *VkDestroyImageView:
		if !GetState(s).ImageViews().Contains(cmd.ImageView()) {
			warnDropCmd(cmd.ImageView())
			return nil
		}
	case *VkDestroyShaderModule:
		if !GetState(s).ShaderModules().Contains(cmd.ShaderModule()) {
			warnDropCmd(cmd.ShaderModule())
			return nil
		}
	case *VkDestroyPipeline:
		if !GetState(s).GraphicsPipelines().Contains(cmd.Pipeline()) &&
			!GetState(s).ComputePipelines().Contains(cmd.Pipeline()) {
			warnDropCmd(cmd.Pipeline())
			return nil
		}
	case *VkDestroyPipelineLayout:
		if !GetState(s).PipelineLayouts().Contains(cmd.PipelineLayout()) {
			warnDropCmd(cmd.PipelineLayout())
			return nil
		}
	case *VkDestroyPipelineCache:
		if !GetState(s).PipelineCaches().Contains(cmd.PipelineCache()) {
			warnDropCmd(cmd.PipelineCache())
			return nil
		}
	case *VkDestroySampler:
		if !GetState(s).Samplers().Contains(cmd.Sampler()) {
			warnDropCmd(cmd.Sampler())
			return nil
		}
	case *VkFreeDescriptorSets:
		descSetCount := cmd.DescriptorSetCount()
		if descSetCount > 0 {
			cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
			descSets := cmd.PDescriptorSets().Slice(0, uint64(descSetCount), l).MustRead(ctx, cmd, s, nil)
			newDescSets := []VkDescriptorSet{}
			dropped := []VkDescriptorSet{}
			for _, ds := range descSets {
				if GetState(s).DescriptorSets().Contains(ds) {
					newDescSets = append(newDescSets, ds)
				} else {
					dropped = append(dropped, ds)
				}
			}
			if len(newDescSets) == 0 {
				// no need to have this command
				warnDropCmd(descSets)
				return nil
			}
			if len(newDescSets) != len(descSets) {
				// need to modify the command to drop the command buffers not
				// in the state out of the command
				warnModCmd(dropped)
				newDescSetsData := s.AllocDataOrPanic(ctx, newDescSets)
				defer newDescSetsData.Free()
				newCmd := cb.VkFreeDescriptorSets(
					cmd.Device(), cmd.DescriptorPool(), uint32(len(newDescSets)),
					newDescSetsData.Ptr(), VkResult_VK_SUCCESS).AddRead(newDescSetsData.Data())
				return out.MutateAndWrite(ctx, id, newCmd)
			}
			// No need to modify the command, just mutate and write the command
			// like others.
		}
	case *VkDestroyDescriptorSetLayout:
		if !GetState(s).DescriptorSetLayouts().Contains(cmd.DescriptorSetLayout()) {
			warnDropCmd(cmd.DescriptorSetLayout())
			return nil
		}
	case *VkDestroyDescriptorPool:
		if !GetState(s).DescriptorPools().Contains(cmd.DescriptorPool()) {
			warnDropCmd(cmd.DescriptorPool())
			return nil
		}
	case *VkDestroyFence:
		if !GetState(s).Fences().Contains(cmd.Fence()) {
			warnDropCmd(cmd.Fence())
			return nil
		}
	case *VkDestroySemaphore:
		if !GetState(s).Semaphores().Contains(cmd.Semaphore()) {
			warnDropCmd(cmd.Semaphore())
			return nil
		}
	case *VkDestroyEvent:
		if !GetState(s).Events().Contains(cmd.Event()) {
			warnDropCmd(cmd.Event())
			return nil
		}
	case *VkDestroyQueryPool:
		if !GetState(s).QueryPools().Contains(cmd.QueryPool()) {
			warnDropCmd(cmd.QueryPool())
			return nil
		}
	case *VkDestroyFramebuffer:
		if !GetState(s).Framebuffers().Contains(cmd.Framebuffer()) {
			warnDropCmd(cmd.Framebuffer())
			return nil
		}
	case *VkDestroyRenderPass:
		if !GetState(s).RenderPasses().Contains(cmd.RenderPass()) {
			warnDropCmd(cmd.RenderPass())
			return nil
		}
	case *VkDestroyCommandPool:
		if !GetState(s).CommandPools().Contains(cmd.CommandPool()) {
			warnDropCmd(cmd.CommandPool())
			return nil
		}
	case *VkDestroySurfaceKHR:
		if !GetState(s).Surfaces().Contains(cmd.Surface()) {
			warnDropCmd(cmd.Surface())
			return nil
		}
	case *VkDestroySwapchainKHR:
		if !GetState(s).Swapchains().Contains(cmd.Swapchain()) {
			warnDropCmd(cmd.Swapchain())
			return nil
		}
	case *VkDestroyDebugReportCallbackEXT:
		if !GetState(s).DebugReportCallbacks().Contains(cmd.Callback()) {
			warnDropCmd(cmd.Callback())
			return nil
		}
	}
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *dropInvalidDestroy) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *dropInvalidDestroy) PreLoop(ctx context.Context, out transform.Writer)     {}
func (t *dropInvalidDestroy) PostLoop(ctx context.Context, out transform.Writer)    {}
func (t *dropInvalidDestroy) BuffersCommands() bool                                 { return false }

// destroyResourceAtEOS is a transformation that destroys all active
// resources at the end of stream.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *destroyResourcesAtEOS) Flush(ctx context.Context, out transform.Writer) error {
	s := out.State()
	so := getStateObject(s)
	id := api.CmdNoID
	cb := CommandBuilder{Thread: 0, Arena: s.Arena} // TODO: Check that using any old thread is okay.
	// TODO: use the correct pAllocator once we handle it.
	p := memory.Nullptr

	// Wait all queues in all devices to finish their jobs first.
	for handle := range so.Devices().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS)); err != nil {
			return err
		}
	}

	// Synchronization primitives.
	for handle, object := range so.Events().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyEvent(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.Fences().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyFence(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.Semaphores().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroySemaphore(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// SamplerYcbcrConversions
	for handle, object := range so.SamplerYcbcrConversions().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroySamplerYcbcrConversion(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Framebuffers, samplers.
	for handle, object := range so.Framebuffers().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyFramebuffer(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.Samplers().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroySampler(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Descriptor sets.
	for handle, object := range so.DescriptorPools().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorPool(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.DescriptorSetLayouts().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorSetLayout(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Buffers.
	for handle, object := range so.BufferViews().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyBufferView(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.Buffers().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyBuffer(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Shader modules.
	for handle, object := range so.ShaderModules().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyShaderModule(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Pipelines.
	for handle, object := range so.GraphicsPipelines().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.ComputePipelines().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.PipelineLayouts().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineLayout(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	for handle, object := range so.PipelineCaches().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineCache(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Render passes.
	for handle, object := range so.RenderPasses().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyRenderPass(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	for handle, object := range so.QueryPools().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyQueryPool(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Command buffers.
	for handle, object := range so.CommandPools().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyCommandPool(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Swapchains.
	for handle, object := range so.Swapchains().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroySwapchainKHR(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Memories.
	for handle, object := range so.DeviceMemories().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkFreeMemory(object.Device(), handle, p)); err != nil {
			return err
		}
	}

	// Images
	for handle, object := range so.ImageViews().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyImageView(object.Device(), handle, p)); err != nil {
			return err
		}
	}
	// Note: so.Images also contains Swapchain images. We do not want
	// to delete those, as that must be handled by VkDestroySwapchainKHR
	for handle, object := range so.Images().All() {
		if !object.IsSwapchainImage() {
			if err := out.MutateAndWrite(ctx, id, cb.VkDestroyImage(object.Device(), handle, p)); err != nil {
				return err
			}
		}
	}

	// Devices.
	for handle := range so.Devices().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyDevice(handle, p)); err != nil {
			return err
		}
	}

	// Surfaces.
	for handle, object := range so.Surfaces().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroySurfaceKHR(object.Instance(), handle, p)); err != nil {
			return err
		}
	}

	// Debug report callbacks
	for handle, object := range so.DebugReportCallbacks().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyDebugReportCallbackEXT(object.Instance(), handle, p)); err != nil {
			return err
		}
	}

	// Instances.
	for handle := range so.Instances().All() {
		if err := out.MutateAndWrite(ctx, id, cb.VkDestroyInstance(handle, p)); err != nil {
			return err
		}
	}

	return nil
}

func (t *destroyResourcesAtEOS) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *destroyResourcesAtEOS) PostLoop(ctx context.Context, out transform.Writer) {}
func (t *destroyResourcesAtEOS) BuffersCommands() bool                              { return false }

func newDisplayToSurface() *DisplayToSurface {
	return &DisplayToSurface{
		SurfaceTypes: map[uint64]uint32{},
	}
}

// DisplayToSurface is a transformation that enables rendering during replay to
// the original surface.
func (t *DisplayToSurface) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	switch c := cmd.(type) {
	case *VkCreateSwapchainKHR:
		newCmd := c.clone(out.State().Arena)
		newCmd.extras = api.CmdExtras{}
		// Add an extra to indicate to custom_replay to add a flag to
		// the virtual swapchain pNext
		newCmd.extras = append(api.CmdExtras{t}, cmd.Extras().All()...)
		return out.MutateAndWrite(ctx, id, newCmd)
	case *VkCreateAndroidSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR)
	case *VkCreateWaylandSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR)
	case *VkCreateWin32SurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR)
	case *VkCreateXcbSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR)
	case *VkCreateXlibSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_XLIB_SURFACE_CREATE_INFO_KHR)
	case *VkCreateMacOSSurfaceMVK:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_MACOS_SURFACE_CREATE_INFO_MVK)
	case *VkCreateStreamDescriptorSurfaceGGP:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_STREAM_DESCRIPTOR_SURFACE_CREATE_INFO_GGP)
	}
	return out.MutateAndWrite(ctx, id, cmd)
}

func (t *DisplayToSurface) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *DisplayToSurface) PreLoop(ctx context.Context, out transform.Writer)     {}
func (t *DisplayToSurface) PostLoop(ctx context.Context, out transform.Writer)    {}
func (t *DisplayToSurface) BuffersCommands() bool                                 { return false }

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct {
}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out              chan<- replay.Issue
	displayToSurface bool
	loopCount        int32
}

type timestampsConfig struct {
}

type timestampsRequest struct {
	handler   service.TimeStampsHandler
	loopCount int32
}

// uniqueConfig returns a replay.Config that is guaranteed to be unique.
// Any requests made with a Config returned from uniqueConfig will not be
// batched with any other request.
func uniqueConfig() replay.Config {
	return &struct{}{}
}

type profileRequest struct {
	traceOptions   *service.TraceOptions
	handler        *replay.SignalHandler
	buffer         *bytes.Buffer
	handleMappings *map[uint64][]service.VulkanHandleMappingItem
}

// GetInitialPayload creates a replay that emits instructions for
// state priming of a capture.
func (a API) GetInitialPayload(ctx context.Context,
	capture *path.Capture,
	device *device.Instance,
	out transform2.Writer) error {

	initialCmds, im, _ := initialcmds.InitialCommands(ctx, capture)
	out.State().Allocator.ReserveRanges(im)
	cmdGenerator := commandGenerator.NewLinearCommandGenerator(initialCmds, nil)

	transforms := getCommonInitializationTransforms("GetInitialPayload", false)

	chain := transform2.CreateTransformChain(cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "[GetInitialPayload] Error: %v", err)
		return err
	}

	return nil
}

// CleanupResources creates a replay that emits instructions for
// destroying resources at a given state
func (a API) CleanupResources(ctx context.Context, device *device.Instance, out transform2.Writer) error {
	cmdGenerator := commandGenerator.NewLinearCommandGenerator(nil, nil)
	transforms := []transform2.Transform{
		newDestroyResourcesAtEOS2(),
	}
	chain := transform2.CreateTransformChain(cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "[CleanupResources] Error: %v", err)
		return err
	}

	return nil
}

func getInitialCmds(ctx context.Context,
	dependentPayload string,
	intent replay.Intent,
	out transform2.Writer) []api.Cmd {

	// Melih TODO: Do we really need this(dependentPayload) for the particular replay type?
	// b/158597615
	if dependentPayload == "" {
		cmds, im, _ := initialcmds.InitialCommands(ctx, intent.Capture)
		out.State().Allocator.ReserveRanges(im)
		return cmds
	}

	return []api.Cmd{}
}

func replayProfile(ctx context.Context,
	intent replay.Intent,
	dependentPayload string,
	rrs []replay.RequestAndResult,
	c *capture.GraphicsCapture,
	device *device.Instance,
	out transform2.Writer) error {

	if len(rrs) > 1 {
		panic("Batched request is not supported for profile")
	}

	if len(rrs) == 0 {
		return fmt.Errorf("No request has been found for profile")
	}

	var layerName string
	if device.GetConfiguration().GetPerfettoCapability().GetGpuProfiling().GetHasRenderStageProducerLayer() {
		layerName = "VkRenderStagesProducer"
	}

	initialCmds := getInitialCmds(ctx, dependentPayload, intent, out)

	transforms := make([]transform2.Transform, 0)
	transforms = append(transforms, getCommonInitializationTransforms("ProfileReplay", true)...)

	profileTransform := newEndOfReplay()
	profileTransform.AddResult(rrs[0].Result)
	request := rrs[0].Request.(profileRequest)
	transforms = append(transforms, newWaitForPerfetto(request.traceOptions, request.handler, request.buffer, api.CmdID(len(initialCmds))))
	transforms = append(transforms, newProfilingLayers(layerName))
	transforms = append(transforms, newMappingExporter(ctx, request.handleMappings))
	transforms = append(transforms, profileTransform)

	transforms = append(transforms, newDestroyResourcesAtEOS2())
	transforms = appendLogTransforms(ctx, "ProfileReplay", c, transforms)

	cmdGenerator := commandGenerator.NewLinearCommandGenerator(initialCmds, c.Commands)
	chain := transform2.CreateTransformChain(cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	err := controlFlow.TransformAll(ctx)
	if err != nil {
		log.E(ctx, "[Profile Replay] Error: %v", err)
		return err
	}

	return nil
}

func replayIssues(ctx context.Context,
	intent replay.Intent,
	dependentPayload string,
	rrs []replay.RequestAndResult,
	c *capture.GraphicsCapture,
	out transform2.Writer) error {

	if len(rrs) > 1 {
		panic("Batched request is not supported for issues")
	}

	if len(rrs) == 0 {
		return fmt.Errorf("No request has been found for issues")
	}

	initialCmds := getInitialCmds(ctx, dependentPayload, intent, out)

	transforms := make([]transform2.Transform, 0)
	transforms = append(transforms, getCommonInitializationTransforms("IssuesReplay", false)...)

	issuesTransform := newFindIssues(ctx, c, api.CmdID(len(initialCmds)))
	issuesTransform.AddResult(rrs[0].Result)
	transforms = append(transforms, issuesTransform)

	req := rrs[0].Request.(issuesRequest)
	if req.displayToSurface {
		transforms = append(transforms, newDisplayToSurface2())
	}

	transforms = append(transforms, newDestroyResourcesAtEOS2())
	transforms = appendLogTransforms(ctx, "IssuesReplay", c, transforms)

	cmdGenerator := commandGenerator.NewLinearCommandGenerator(initialCmds, c.Commands)
	chain := transform2.CreateTransformChain(cmdGenerator, transforms, out)
	controlFlow := controlFlowGenerator.NewLinearControlFlowGenerator(chain)
	if err := controlFlow.TransformAll(ctx); err != nil {
		log.E(ctx, "[Issues Replay] Error: %v", err)
		return err
	}

	return nil
}

func (a API) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	dependentPayload string,
	rrs []replay.RequestAndResult,
	device *device.Instance,
	c *capture.GraphicsCapture,
	out transform.Writer) error {
	if a.GetReplayPriority(ctx, device, c.Header) == 0 {
		return log.Errf(ctx, nil, "Cannot replay Vulkan commands on device '%v'", device.Name)
	}

	// Melih TODO: This will be a dispatcher when other queries also merged.
	// This function written as if it can run multiple types of request at once but
	// it actually cannot. So, I am trying to clean this up.
	for _, rr := range rrs {
		switch rr.Request.(type) {
		case issuesRequest:
			return replayIssues(ctx, intent, dependentPayload, rrs, c, out)
		case profileRequest:
			return replayProfile(ctx, intent, dependentPayload, rrs, c, device, out)
		}
	}

	optimize := !config.DisableDeadCodeElimination

	cmds := c.Commands

	transforms := transform.Transforms{}
	makeReadable := &makeAttachementReadable{false}
	transforms.Add(makeReadable)
	transforms.Add(&dropInvalidDestroy{tag: "Replay"})

	splitter := NewCommandSplitter(ctx)
	readFramebuffer := newReadFramebuffer(ctx)
	injector := &transform.Injector{}
	// Gathers and reports any issues found.
	var timestamps *queryTimestamps
	var frameloop *frameLoop

	earlyTerminator, err := newVulkanTerminator(ctx, intent.Capture)
	if err != nil {
		return err
	}

	// Populate the dead-code eliminitation later, only once we are sure
	// we will need it.
	var dceBuilder *dependencygraph2.DCEBuilder

	initCmdExpandedWithOpt := false
	numInitialCmdWithOpt := 0
	initCmdExpandedWithoutOpt := false
	numInitialCmdWithoutOpt := 0
	dceExpanded := false
	expandCommands := func(opt bool) (int, error) {
		if opt && initCmdExpandedWithOpt {
			return numInitialCmdWithOpt, nil
		}
		if !opt && initCmdExpandedWithoutOpt {
			return numInitialCmdWithoutOpt, nil
		}
		if opt {
			if dceBuilder == nil && !dceExpanded {
				cfg := dependencygraph2.DependencyGraphConfig{
					MergeSubCmdNodes:       !config.DeadSubCmdElimination,
					IncludeInitialCommands: dependentPayload == "",
				}
				graph, err := dependencygraph2.TryGetDependencyGraph(ctx, capture.Get(ctx), cfg)
				if err != nil {
					return 0, fmt.Errorf("Could not build dependency graph for DCE: %v", err)
				}

				if graph != nil {
					log.I(ctx, "Got Dependency Graph")
					dceBuilder = dependencygraph2.NewDCEBuilder(graph)
				} else {
					log.I(ctx, "Dependency Graph still pending")
				}
			}
			dceExpanded = true
			if dceBuilder != nil {
				cmds = []api.Cmd{}
			}
			return 0, nil
		}
		// If we do not depend on another payload to get us into the right state,
		// we should get ourselves into the intial state
		if dependentPayload == "" {
			initialCmds, im, _ := initialcmds.InitialCommands(ctx, intent.Capture)
			out.State().Allocator.ReserveRanges(im)
			numInitialCmdWithoutOpt = len(initialCmds)
			if len(initialCmds) > 0 {
				cmds = append(initialCmds, cmds...)
			}
		}
		initCmdExpandedWithoutOpt = true
		return numInitialCmdWithoutOpt, nil
	}

	frameLoopEndCmdID := func(cmds []api.Cmd) api.CmdID {
		lastCmdID := api.CmdID(0)
		for i := len(cmds) - 1; i >= 0; i-- {
			if cmds[i].Terminated() {
				lastCmdID = api.CmdID(i)
				break
			}
		}
		return lastCmdID
	}

	wire := false
	doDisplayToSurface := false
	var overdraw *stencilOverdraw

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case timestampsRequest:
			if timestamps == nil {
				willLoop := req.loopCount > 1
				if willLoop {
					frameloop = newFrameLoop(ctx, c, api.CmdID(0), frameLoopEndCmdID(cmds), req.loopCount)
				}

				timestamps = newQueryTimestamps(ctx, c, cmds, willLoop, req.handler)
			}
			timestamps.AddResult(rr.Result)
			optimize = false
		case framebufferRequest:
			cfg := cfg.(drawConfig)
			if cfg.disableReplayOptimization {
				optimize = false
			}

			cmdID := req.after[0]

			if optimize {
				// Should have been built in expandCommands()
				if dceBuilder != nil {
					dceBuilder.Request(ctx, api.SubCmdIdx{cmdID})
				} else {
					optimize = false
				}
			}

			if cfg.drawMode == path.DrawMode_OVERDRAW {
				// TODO(subcommands): Add subcommand support here
				if err := earlyTerminator.Add(ctx, api.CmdID(cmdID), req.after[1:]); err != nil {
					return err
				}

				if overdraw == nil {
					overdraw = newStencilOverdraw()
				}
				overdraw.add(ctx, req.after, intent.Capture, rr.Result)
				break
			}
			if err := earlyTerminator.Add(ctx, api.CmdID(cmdID), api.SubCmdIdx{}); err != nil {
				return err
			}
			subIdx := append(api.SubCmdIdx{}, req.after...)
			splitter.Split(ctx, subIdx)
			switch cfg.drawMode {
			case path.DrawMode_WIREFRAME_ALL:
				wire = true
			case path.DrawMode_WIREFRAME_OVERLAY:
				return fmt.Errorf("Overlay wireframe view is not currently supported")
			// Overdraw is handled above, since it breaks out of the normal read flow.
			default:
			}

			switch req.attachment {
			case api.FramebufferAttachmentType_OutputDepth, api.FramebufferAttachmentType_InputDepth:
				readFramebuffer.Depth(ctx, subIdx, req.width, req.height, req.framebufferIndex, rr.Result)
			case api.FramebufferAttachmentType_OutputColor, api.FramebufferAttachmentType_InputColor:
				readFramebuffer.Color(ctx, subIdx, req.width, req.height, req.framebufferIndex, rr.Result)
			default:
				return fmt.Errorf("Stencil attachments are not currently supported")
			}

			if req.displayToSurface {
				doDisplayToSurface = true
			}
		default:
			return fmt.Errorf("Invalid Request Type; %v", req)
		}
	}

	numberOfInitialCmds, err := expandCommands(optimize)
	if err != nil {
		return err
	}

	// Use the dead code elimination pass
	if optimize {
		if dceBuilder != nil {
			transforms.Prepend(dceBuilder)
		} else {
			log.W(ctx, "Replay optimization enabled but cannot be done as dependency graph not ready")
		}
	}

	if wire {
		transforms.Add(wireframe(ctx))
	}

	if doDisplayToSurface {
		transforms.Add(newDisplayToSurface())
	}

	if frameloop != nil {
		transforms.Add(frameloop)
	}
	if timestamps != nil {
		transforms.Add(timestamps)
	} else {
		transforms.Add(earlyTerminator)
	}

	if overdraw != nil {
		transforms.Add(overdraw)
	}

	transforms.Add(splitter)
	transforms.Add(readFramebuffer, injector)

	// Cleanup
	transforms.Add(&destroyResourcesAtEOS{})

	if config.DebugReplay {
		log.I(ctx, "Replaying %d commands using transform chain:", len(cmds))
		for i, t := range transforms {
			log.I(ctx, "(%d) %#v", i, t)
		}
	}

	if config.LogTransformsToFile {
		newTransforms := transform.Transforms{}
		newTransforms.Add(transform.NewFileLog(ctx, "0_original_cmds"))
		for i, t := range transforms {
			var name string
			if n, ok := t.(interface {
				Name() string
			}); ok {
				name = n.Name()
			} else {
				name = strings.Replace(fmt.Sprintf("%T", t), "*", "", -1)
			}
			newTransforms.Add(t, transform.NewFileLog(ctx, fmt.Sprintf("%v_cmds_after_%v", i+1, name)))
		}
		transforms = newTransforms
	}
	if config.LogTransformsToCapture {
		transforms.Add(transform.NewCaptureLog(ctx, c, "replay_log.gfxtrace"))
	}
	if config.LogMappingsToFile {
		transforms.Add(replay.NewMappingExporterWithPrint(ctx, "mappings.txt"))
	}

	return transforms.TransformAll(ctx, cmds, uint64(numberOfInitialCmds), out)
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachmentType,
	framebufferIndex uint32,
	drawMode path.DrawMode,
	disableReplayOptimization bool,
	displayToSurface bool,
	hints *path.UsageHints) (*image.Data, error) {

	beginIndex := api.CmdID(0)
	endIndex := api.CmdID(0)
	subcommand := ""
	// We cant break up overdraw right now, but we can break up
	// everything else.
	if drawMode == path.DrawMode_OVERDRAW {
		if len(after) > 1 { // If we are replaying subcommands, then we can't batch at all
			beginIndex = api.CmdID(after[0])
			endIndex = api.CmdID(after[0])
			for i, j := range after[1:] {
				if i != 0 {
					subcommand += ":"
				}
				subcommand += fmt.Sprintf("%d", j)
			}
		}
	}

	c := drawConfig{beginIndex, endIndex, subcommand, drawMode, disableReplayOptimization}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, framebufferIndex: framebufferIndex, attachment: attachment, out: out, displayToSurface: displayToSurface}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, false)
	if err != nil {
		return nil, err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil, nil
	}
	return res.(*image.Data), nil
}

func (a API) QueryIssues(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	loopCount int32,
	displayToSurface bool,
	hints *path.UsageHints) ([]replay.Issue, error) {

	c, r := issuesConfig{}, issuesRequest{displayToSurface: displayToSurface, loopCount: loopCount}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints, true)

	if err != nil {
		return nil, err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil, nil
	}
	return res.([]replay.Issue), nil
}

func (a API) QueryTimestamps(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	loopCount int32,
	handler service.TimeStampsHandler,
	hints *path.UsageHints) error {

	c, r := timestampsConfig{}, timestampsRequest{
		handler:   handler,
		loopCount: loopCount}
	_, err := mgr.Replay(ctx, intent, c, r, a, hints, false)
	if err != nil {
		return err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil
	}
	return nil
}

func (a API) Profile(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	hints *path.UsageHints,
	traceOptions *service.TraceOptions) (*service.ProfilingData, error) {

	c := uniqueConfig()
	handler := replay.NewSignalHandler()
	var buffer bytes.Buffer
	handleMappings := make(map[uint64][]service.VulkanHandleMappingItem)
	r := profileRequest{traceOptions, handler, &buffer, &handleMappings}
	_, err := mgr.Replay(ctx, intent, c, r, a, hints, true)
	if err != nil {
		return nil, err
	}
	handler.DoneSignal.Wait(ctx)

	s, err := resolve.SyncData(ctx, intent.Capture)
	if err != nil {
		return nil, err
	}

	d, err := trace.ProcessProfilingData(ctx, intent.Device, intent.Capture, &buffer, &handleMappings, s)
	return d, err
}

func getCommonInitializationTransforms(tag string, imagesOnly bool) []transform2.Transform {
	return []transform2.Transform{
		newMakeAttachmentReadable2(imagesOnly),
		newDropInvalidDestroy2(tag),
	}
}

func appendLogTransforms(ctx context.Context, tag string, capture *capture.GraphicsCapture, transforms []transform2.Transform) []transform2.Transform {
	if config.LogTransformsToFile {
		newTransforms := make([]transform2.Transform, 0)
		newTransforms = append(newTransforms, newFileLog(ctx, "0_original_cmds"))
		for i, t := range transforms {
			var name string
			if n, ok := t.(interface {
				Name() string
			}); ok {
				name = n.Name()
			} else {
				name = strings.Replace(fmt.Sprintf("%T", t), "*", "", -1)
			}
			newTransforms = append(newTransforms, t, newFileLog(ctx, fmt.Sprintf("%v_cmds_after_%v", i+1, name)))
		}
		transforms = newTransforms
	}

	if config.LogTransformsToCapture {
		transforms = append(transforms, newCaptureLog(ctx, capture, tag+"_replay_log.gfxtrace"))
	}

	if config.LogMappingsToFile {
		transforms = append(transforms, newMappingExporterWithPrint(ctx, tag+"_mappings.txt"))
	}

	return transforms
}
