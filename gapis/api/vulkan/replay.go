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
	"strings"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/resolve/initialcmds"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(API{})
	_ = replay.QueryFramebufferAttachment(API{})
	_ = replay.Support(API{})
	_ = replay.QueryTimestamps(API{})
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
				if devPhyInfo.GetApiVersion() != tracePhyInfo.GetApiVersion() {
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
type makeAttachementReadable struct{}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	startScope                api.CmdID
	endScope                  api.CmdID
	subindices                string // drawConfig needs to be comparable, so we cannot use a slice
	drawMode                  service.DrawMode
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
	attachment       api.FramebufferAttachment
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

func (t *makeAttachementReadable) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
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
			out.MutateAndWrite(ctx, id, newCmd)
			return
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
			out.MutateAndWrite(ctx, id, newCmd)
			return
		}
	} else if createRenderPass, ok := cmd.(*VkCreateRenderPass); ok {
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
			out.MutateAndWrite(ctx, id, cmd)
			return
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
		out.MutateAndWrite(ctx, id, newCmd)
		return
	} else if e, ok := cmd.(*VkEnumeratePhysicalDevices); ok {
		if e.PPhysicalDevices() == 0 {
			// Querying for the number of devices.
			// No changes needed here.
			out.MutateAndWrite(ctx, id, cmd)
			return
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
		out.MutateAndWrite(ctx, id, newEnumerate)
		return
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *makeAttachementReadable) Flush(ctx context.Context, out transform.Writer) {}

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

func (t *dropInvalidDestroy) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
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
			return
		}
	case *VkDestroyDevice:
		if !GetState(s).Devices().Contains(cmd.Device()) {
			warnDropCmd(cmd.Device())
			return
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
				return
			}
			if len(newCmdBufs) != len(cmdBufs) {
				// need to modify the command to drop the command buffers not
				// in the state out of the command
				warnModCmd(dropped)
				newCmdBufsData := s.AllocDataOrPanic(ctx, newCmdBufs)
				defer newCmdBufsData.Free()
				newCmd := cb.VkFreeCommandBuffers(cmd.Device(), cmd.CommandPool(), uint32(len(newCmdBufs)), newCmdBufsData.Ptr()).AddRead(newCmdBufsData.Data())
				out.MutateAndWrite(ctx, id, newCmd)
				return
			}
			// No need to modify the command, just mutate and write the command
			// like others.
		}
	case *VkFreeMemory:
		if !GetState(s).DeviceMemories().Contains(cmd.Memory()) {
			warnDropCmd(cmd.Memory())
			return
		}
	case *VkDestroyBuffer:
		if !GetState(s).Buffers().Contains(cmd.Buffer()) {
			warnDropCmd(cmd.Buffer())
			return
		}
	case *VkDestroyBufferView:
		if !GetState(s).BufferViews().Contains(cmd.BufferView()) {
			warnDropCmd(cmd.BufferView())
			return
		}
	case *VkDestroyImage:
		if !GetState(s).Images().Contains(cmd.Image()) {
			warnDropCmd(cmd.Image())
			return
		}
	case *VkDestroyImageView:
		if !GetState(s).ImageViews().Contains(cmd.ImageView()) {
			warnDropCmd(cmd.ImageView())
			return
		}
	case *VkDestroyShaderModule:
		if !GetState(s).ShaderModules().Contains(cmd.ShaderModule()) {
			warnDropCmd(cmd.ShaderModule())
			return
		}
	case *VkDestroyPipeline:
		if !GetState(s).GraphicsPipelines().Contains(cmd.Pipeline()) &&
			!GetState(s).ComputePipelines().Contains(cmd.Pipeline()) {
			warnDropCmd(cmd.Pipeline())
			return
		}
	case *VkDestroyPipelineLayout:
		if !GetState(s).PipelineLayouts().Contains(cmd.PipelineLayout()) {
			warnDropCmd(cmd.PipelineLayout())
			return
		}
	case *VkDestroyPipelineCache:
		if !GetState(s).PipelineCaches().Contains(cmd.PipelineCache()) {
			warnDropCmd(cmd.PipelineCache())
			return
		}
	case *VkDestroySampler:
		if !GetState(s).Samplers().Contains(cmd.Sampler()) {
			warnDropCmd(cmd.Sampler())
			return
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
				return
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
				out.MutateAndWrite(ctx, id, newCmd)
				return
			}
			// No need to modify the command, just mutate and write the command
			// like others.
		}
	case *VkDestroyDescriptorSetLayout:
		if !GetState(s).DescriptorSetLayouts().Contains(cmd.DescriptorSetLayout()) {
			warnDropCmd(cmd.DescriptorSetLayout())
			return
		}
	case *VkDestroyDescriptorPool:
		if !GetState(s).DescriptorPools().Contains(cmd.DescriptorPool()) {
			warnDropCmd(cmd.DescriptorPool())
			return
		}
	case *VkDestroyFence:
		if !GetState(s).Fences().Contains(cmd.Fence()) {
			warnDropCmd(cmd.Fence())
			return
		}
	case *VkDestroySemaphore:
		if !GetState(s).Semaphores().Contains(cmd.Semaphore()) {
			warnDropCmd(cmd.Semaphore())
			return
		}
	case *VkDestroyEvent:
		if !GetState(s).Events().Contains(cmd.Event()) {
			warnDropCmd(cmd.Event())
			return
		}
	case *VkDestroyQueryPool:
		if !GetState(s).QueryPools().Contains(cmd.QueryPool()) {
			warnDropCmd(cmd.QueryPool())
			return
		}
	case *VkDestroyFramebuffer:
		if !GetState(s).Framebuffers().Contains(cmd.Framebuffer()) {
			warnDropCmd(cmd.Framebuffer())
			return
		}
	case *VkDestroyRenderPass:
		if !GetState(s).RenderPasses().Contains(cmd.RenderPass()) {
			warnDropCmd(cmd.RenderPass())
			return
		}
	case *VkDestroyCommandPool:
		if !GetState(s).CommandPools().Contains(cmd.CommandPool()) {
			warnDropCmd(cmd.CommandPool())
			return
		}
	case *VkDestroySurfaceKHR:
		if !GetState(s).Surfaces().Contains(cmd.Surface()) {
			warnDropCmd(cmd.Surface())
			return
		}
	case *VkDestroySwapchainKHR:
		if !GetState(s).Swapchains().Contains(cmd.Swapchain()) {
			warnDropCmd(cmd.Swapchain())
			return
		}
	case *VkDestroyDebugReportCallbackEXT:
		if !GetState(s).DebugReportCallbacks().Contains(cmd.Callback()) {
			warnDropCmd(cmd.Callback())
			return
		}
	}
	out.MutateAndWrite(ctx, id, cmd)
	return
}

func (t *dropInvalidDestroy) Flush(ctx context.Context, out transform.Writer) {}

// destroyResourceAtEOS is a transformation that destroys all active
// resources at the end of stream.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *destroyResourcesAtEOS) Flush(ctx context.Context, out transform.Writer) {
	s := out.State()
	so := getStateObject(s)
	id := api.CmdNoID
	cb := CommandBuilder{Thread: 0, Arena: s.Arena} // TODO: Check that using any old thread is okay.
	// TODO: use the correct pAllocator once we handle it.
	p := memory.Nullptr

	// Wait all queues in all devices to finish their jobs first.
	for handle := range so.Devices().All() {
		out.MutateAndWrite(ctx, id, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}

	// Synchronization primitives.
	for handle, object := range so.Events().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyEvent(object.Device(), handle, p))
	}
	for handle, object := range so.Fences().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyFence(object.Device(), handle, p))
	}
	for handle, object := range so.Semaphores().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySemaphore(object.Device(), handle, p))
	}

	// Framebuffers, samplers.
	for handle, object := range so.Framebuffers().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyFramebuffer(object.Device(), handle, p))
	}
	for handle, object := range so.Samplers().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySampler(object.Device(), handle, p))
	}

	// Descriptor sets.
	for handle, object := range so.DescriptorPools().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorPool(object.Device(), handle, p))
	}
	for handle, object := range so.DescriptorSetLayouts().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorSetLayout(object.Device(), handle, p))
	}

	// Buffers.
	for handle, object := range so.BufferViews().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyBufferView(object.Device(), handle, p))
	}
	for handle, object := range so.Buffers().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyBuffer(object.Device(), handle, p))
	}

	// Shader modules.
	for handle, object := range so.ShaderModules().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyShaderModule(object.Device(), handle, p))
	}

	// Pipelines.
	for handle, object := range so.GraphicsPipelines().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device(), handle, p))
	}
	for handle, object := range so.ComputePipelines().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device(), handle, p))
	}
	for handle, object := range so.PipelineLayouts().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineLayout(object.Device(), handle, p))
	}
	for handle, object := range so.PipelineCaches().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineCache(object.Device(), handle, p))
	}

	// Render passes.
	for handle, object := range so.RenderPasses().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyRenderPass(object.Device(), handle, p))
	}

	for handle, object := range so.QueryPools().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyQueryPool(object.Device(), handle, p))
	}

	// Command buffers.
	for handle, object := range so.CommandPools().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyCommandPool(object.Device(), handle, p))
	}

	// Swapchains.
	for handle, object := range so.Swapchains().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySwapchainKHR(object.Device(), handle, p))
	}

	// Memories.
	for handle, object := range so.DeviceMemories().All() {
		out.MutateAndWrite(ctx, id, cb.VkFreeMemory(object.Device(), handle, p))
	}

	// Images
	for handle, object := range so.ImageViews().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyImageView(object.Device(), handle, p))
	}
	// Note: so.Images also contains Swapchain images. We do not want
	// to delete those, as that must be handled by VkDestroySwapchainKHR
	for handle, object := range so.Images().All() {
		if !object.IsSwapchainImage() {
			out.MutateAndWrite(ctx, id, cb.VkDestroyImage(object.Device(), handle, p))
		}
	}

	// Devices.
	for handle := range so.Devices().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDevice(handle, p))
	}

	// Surfaces.
	for handle, object := range so.Surfaces().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySurfaceKHR(object.Instance(), handle, p))
	}

	// Debug report callbacks
	for handle, object := range so.DebugReportCallbacks().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDebugReportCallbackEXT(object.Instance(), handle, p))
	}

	// Instances.
	for handle := range so.Instances().All() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyInstance(handle, p))
	}
}

func newDisplayToSurface() *DisplayToSurface {
	return &DisplayToSurface{
		SurfaceTypes: map[uint64]uint32{},
	}
}

// DisplayToSurface is a transformation that enables rendering during replay to
// the original surface.
func (t *DisplayToSurface) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	switch c := cmd.(type) {
	case *VkCreateSwapchainKHR:
		newCmd := c.clone(out.State().Arena)
		newCmd.extras = api.CmdExtras{}
		// Add an extra to indicate to custom_replay to add a flag to
		// the virtual swapchain pNext
		newCmd.extras = append(api.CmdExtras{t}, cmd.Extras().All()...)
		out.MutateAndWrite(ctx, id, newCmd)
		return
	case *VkCreateAndroidSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR)
	case *VkCreateMirSurfaceKHR:
		cmd.Extras().Observations().ApplyWrites(out.State().Memory.ApplicationPool())
		surface := c.PSurface().MustRead(ctx, cmd, out.State(), nil)
		t.SurfaceTypes[uint64(surface)] = uint32(VkStructureType_VK_STRUCTURE_TYPE_MIR_SURFACE_CREATE_INFO_KHR)
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
	}
	out.MutateAndWrite(ctx, id, cmd)
}

func (t *DisplayToSurface) Flush(ctx context.Context, out transform.Writer) {
}

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct {
}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out              chan<- replay.Issue
	displayToSurface bool
}

type timestampsConfig struct {
}

type timestampsRequest struct {
}

func (a API) GetInitialPayload(ctx context.Context,
	capture *path.Capture,
	device *device.Instance,
	out transform.Writer) error {
	transforms := transform.Transforms{}
	transforms.Add(&makeAttachementReadable{})
	transforms.Add(&dropInvalidDestroy{tag: "GetInitialPayload"})
	initialCmds, im, _ := initialcmds.InitialCommands(ctx, capture)
	out.State().Allocator.ReserveRanges(im)

	transforms.Transform(ctx, initialCmds, out)
	return nil
}

func (a API) CleanupResources(ctx context.Context,
	device *device.Instance,
	out transform.Writer) error {
	transforms := transform.Transforms{}
	transforms.Add(&destroyResourcesAtEOS{})
	transforms.Transform(ctx, []api.Cmd{}, out)
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

	optimize := !config.DisableDeadCodeElimination

	cmds := c.Commands

	transforms := transform.Transforms{}
	transforms.Add(&makeAttachementReadable{})
	transforms.Add(&dropInvalidDestroy{tag: "Replay"})

	readFramebuffer := newReadFramebuffer(ctx)
	injector := &transform.Injector{}
	// Gathers and reports any issues found.
	var issues *findIssues

	var timestamps *queryTimestamps

	earlyTerminator, err := NewVulkanTerminator(ctx, intent.Capture)
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

	wire := false
	doDisplayToSurface := false
	var overdraw *stencilOverdraw

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case issuesRequest:
			if issues == nil {
				n, err := expandCommands(false)
				if err != nil {
					return err
				}
				issues = newFindIssues(ctx, c, n)
			}
			issues.reportTo(rr.Result)
			optimize = false
			if req.displayToSurface {
				doDisplayToSurface = true
			}
		case timestampsRequest:
			if timestamps == nil {
				n, err := expandCommands(false)
				if err != nil {
					return err
				}
				timestamps = newQueryTimestamps(ctx, c, n)
			}
			timestamps.reportTo(rr.Result)
			optimize = false
		case framebufferRequest:

			cfg := cfg.(drawConfig)
			if cfg.disableReplayOptimization {
				optimize = false
			}
			extraCommands, err := expandCommands(optimize)
			if err != nil {
				return err
			}
			cmdid := req.after[0] + uint64(extraCommands)
			// TODO(subcommands): Add subcommand support here
			if err := earlyTerminator.Add(ctx, extraCommands, api.CmdID(cmdid), req.after[1:]); err != nil {
				return err
			}

			after := api.CmdID(cmdid)
			if len(req.after) > 1 {
				// If we are dealing with subcommands, 2 things are true.
				// 2) the earlyTerminator.lastRequest is the last command we have to actually run.
				//     Either the VkQueueSubmit, or the VkSetEvent if synchronization comes in to play
				after = earlyTerminator.lastRequest
			}

			if optimize {
				// Should have been built in expandCommands()
				if dceBuilder != nil {
					dceBuilder.Request(ctx, api.SubCmdIdx{cmdid})
				} else {
					optimize = false
				}
			}

			switch cfg.drawMode {
			case service.DrawMode_WIREFRAME_ALL:
				wire = true
			case service.DrawMode_WIREFRAME_OVERLAY:
				return fmt.Errorf("Overlay wireframe view is not currently supported")
			case service.DrawMode_OVERDRAW:
				if overdraw == nil {
					overdraw = newStencilOverdraw()
				}
				overdraw.add(ctx, uint64(extraCommands), req.after, intent.Capture, rr.Result)
			}

			if cfg.drawMode != service.DrawMode_OVERDRAW {
				switch req.attachment {
				case api.FramebufferAttachment_Depth:
					readFramebuffer.Depth(after, req.framebufferIndex, rr.Result)
				case api.FramebufferAttachment_Stencil:
					return fmt.Errorf("Stencil attachments are not currently supported")
				default:
					readFramebuffer.Color(after, req.width, req.height, req.framebufferIndex, rr.Result)
				}
			}
			if req.displayToSurface {
				doDisplayToSurface = true
			}
		}
	}

	_, err = expandCommands(optimize)
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

	if issues != nil {
		transforms.Add(issues) // Issue reporting required.
	} else {
		if timestamps != nil {
			transforms.Add(timestamps)
		} else {
			transforms.Add(earlyTerminator)
		}

	}

	if overdraw != nil {
		transforms.Add(overdraw)
	}

	if issues == nil {
		transforms.Add(readFramebuffer, injector)
	}

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
		transforms.Add(replay.NewMappingPrinter(ctx, "mappings.txt"))
	}

	transforms.Transform(ctx, cmds, out)
	return nil
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	drawMode service.DrawMode,
	disableReplayOptimization bool,
	displayToSurface bool,
	hints *service.UsageHints) (*image.Data, error) {

	s, err := resolve.SyncData(ctx, intent.Capture)
	if err != nil {
		return nil, err
	}
	beginIndex := api.CmdID(0)
	endIndex := api.CmdID(0)
	subcommand := ""
	if len(after) == 1 {
		a := api.CmdID(after[0])
		// If we are not running subcommands we can probably batch
		for _, v := range s.SortedKeys() {
			if v > a {
				break
			}
			for _, k := range s.CommandRanges[v].SortedKeys() {
				if k > a {
					beginIndex = v
					endIndex = k
				}
			}
		}
	} else { // If we are replaying subcommands, then we can't batch at all
		beginIndex = api.CmdID(after[0])
		endIndex = api.CmdID(after[0])
		for i, j := range after[1:] {
			if i != 0 {
				subcommand += ":"
			}
			subcommand += fmt.Sprintf("%d", j)
		}
	}

	c := drawConfig{beginIndex, endIndex, subcommand, drawMode, disableReplayOptimization}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, framebufferIndex: framebufferIndex, attachment: attachment, out: out, displayToSurface: displayToSurface}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
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
	displayToSurface bool,
	hints *service.UsageHints) ([]replay.Issue, error) {

	c, r := issuesConfig{}, issuesRequest{displayToSurface: displayToSurface}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
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
	hints *service.UsageHints) ([]replay.Timestamp, error) {

	c, r := timestampsConfig{}, timestampsRequest{}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
	if err != nil {
		return nil, err
	}
	if _, ok := mgr.(replay.Exporter); ok {
		return nil, nil
	}
	return res.([]replay.Timestamp), nil
}
