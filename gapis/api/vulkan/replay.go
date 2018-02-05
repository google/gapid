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
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(API{})
	_ = replay.QueryFramebufferAttachment(API{})
	_ = replay.Support(API{})
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
		if !abi.GetMemoryLayout().SameAs(h.GetAbi().GetMemoryLayout()) {
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
				if devPhyInfo.GetVendorID() != tracePhyInfo.GetVendorID() {
					continue
				}
				if devPhyInfo.GetDeviceID() != tracePhyInfo.GetDeviceID() {
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

// makeAttachementReadable is a transformation marking all color/depth/stencil attachment
// images created via vkCreateImage atoms as readable (by patching the transfer src bit).
type makeAttachementReadable struct {
	Instance                  VkInstance
	EnumeratedPhysicalDevices []VkPhysicalDevice
	Properties                map[VkPhysicalDevice]VkPhysicalDeviceProperties
}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
	startScope                api.CmdID
	endScope                  api.CmdID
	subindices                string // drawConfig needs to be comparable, so we cannot use a slice
	wireframeMode             replay.WireframeMode
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
}

type deadCodeEliminationInfo struct {
	dependencyGraph     *dependencygraph.DependencyGraph
	deadCodeElimination *transform.DeadCodeElimination
}

type dCEInfo struct {
	ft  *dependencygraph.Footprint
	dce *transform.DCE
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
	cb := CommandBuilder{Thread: cmd.Thread()}
	cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

	// Info for physical device enumeration
	inPhyDevEnumeration := false
	numPhyDevLeft := uint32(0)
	vkInst := VkInstance(0)
	phyDevs := []VkPhysicalDevice{}
	phyDevProps := []VkPhysicalDeviceProperties{}

	if image, ok := cmd.(*VkCreateImage); ok {
		pinfo := image.PCreateInfo
		info := pinfo.MustRead(ctx, image, s, nil)

		if newUsage, changed := patchImageUsage(info.Usage); changed {
			device := image.Device
			palloc := memory.Pointer(image.PAllocator)
			pimage := memory.Pointer(image.PImage)
			result := image.Result

			info.Usage = newUsage
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
		pinfo := swapchain.PCreateInfo
		info := pinfo.MustRead(ctx, swapchain, s, nil)

		if newUsage, changed := patchImageUsage(info.ImageUsage); changed {
			device := swapchain.Device
			palloc := memory.Pointer(swapchain.PAllocator)
			pswapchain := memory.Pointer(swapchain.PSwapchain)
			result := swapchain.Result

			info.ImageUsage = newUsage
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
		pInfo := createRenderPass.PCreateInfo
		info := pInfo.MustRead(ctx, createRenderPass, s, nil)
		pAttachments := info.PAttachments
		attachments := pAttachments.Slice(uint64(0), uint64(info.AttachmentCount), l).MustRead(ctx, createRenderPass, s, nil)
		changed := false
		for i := range attachments {
			if attachments[i].StoreOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE {
				changed = true
				attachments[i].StoreOp = VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE
			}
		}
		// Returns if no attachment description needs to be changed
		if !changed {
			out.MutateAndWrite(ctx, id, cmd)
			return
		}
		// Build new attachments data, new create info and new command
		newAttachments := s.AllocDataOrPanic(ctx, attachments)
		info.PAttachments = NewVkAttachmentDescriptionᶜᵖ(newAttachments.Ptr())
		newInfo := s.AllocDataOrPanic(ctx, info)
		newCmd := cb.VkCreateRenderPass(createRenderPass.Device,
			newInfo.Ptr(),
			memory.Pointer(createRenderPass.PAllocator),
			memory.Pointer(createRenderPass.PRenderPass),
			createRenderPass.Result)
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
		if e.PPhysicalDevices == NewVkPhysicalDeviceᵖ(nil) {
			// First call to vkEnumeratePhysicalDevices
			out.MutateAndWrite(ctx, id, cmd)
			return
		}
		// Second call to vkEnumeratePhysicalDevices
		l := s.MemoryLayout
		cmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
		numPhyDevLeft = uint32(e.PPhysicalDeviceCount.Slice(uint64(0), uint64(1), l).Index(uint64(0), l).MustRead(ctx, cmd, s, nil))
		// Do not mutate the second call to vkEnumeratePhysicalDevices, all the following vkGetPhysicalDeviceProperties belong to this
		// physical device enumeration.
		inPhyDevEnumeration = true

	} else if g, ok := cmd.(*VkGetPhysicalDeviceProperties); ok {
		if inPhyDevEnumeration {
			cmd.Extras().Observations().ApplyWrites(s.Memory.ApplicationPool())
			prop := g.PProperties.MustRead(ctx, cmd, s, nil)
			phyDevProps = append(phyDevProps, prop)
			phyDevs = append(phyDevs, g.PhysicalDevice)
			if numPhyDevLeft > 0 {
				numPhyDevLeft--
			}
			if numPhyDevLeft == 0 {
				newEnumerate := buildReplayEnumeratePhysicalDevices(ctx, s, cb, vkInst, uint32(len(phyDevs)), phyDevs, phyDevProps)
				out.MutateAndWrite(ctx, id, newEnumerate)
				inPhyDevEnumeration = false
			}
			return
		} else {
			out.MutateAndWrite(ctx, id, g)
			return
		}
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
			propertiesInOrder[i].VendorID)<<32|
			uint64(propertiesInOrder[i].DeviceID))
	}
	devIdData := s.AllocDataOrPanic(ctx, dids)
	return cb.ReplayEnumeratePhysicalDevices(
		instance, numDevData.Ptr(), phyDevData.Ptr(), devIdData.Ptr(),
		VkResult_VK_SUCCESS).AddRead(
		numDevData.Data()).AddRead(phyDevData.Data()).AddRead(devIdData.Data())
}

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
	cb := CommandBuilder{Thread: 0} // TODO: Check that using any old thread is okay.
	// TODO: use the correct pAllocator once we handle it.
	p := memory.Nullptr

	// Wait all queues in all devices to finish their jobs first.
	for handle := range so.Devices.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}

	// Synchronization primitives.
	for handle, object := range so.Events.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyEvent(object.Device, handle, p))
	}
	for handle, object := range so.Fences.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyFence(object.Device, handle, p))
	}
	for handle, object := range so.Semaphores.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySemaphore(object.Device, handle, p))
	}

	// Framebuffers, samplers.
	for handle, object := range so.Framebuffers.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyFramebuffer(object.Device, handle, p))
	}
	for handle, object := range so.Samplers.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySampler(object.Device, handle, p))
	}

	for handle, object := range so.ImageViews.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyImageView(object.Device, handle, p))
	}

	// Buffers.
	for handle, object := range so.BufferViews.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyBufferView(object.Device, handle, p))
	}
	for handle, object := range so.Buffers.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyBuffer(object.Device, handle, p))
	}

	// Descriptor sets.
	for handle, object := range so.DescriptorPools.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorPool(object.Device, handle, p))
	}
	for handle, object := range so.DescriptorSetLayouts.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDescriptorSetLayout(object.Device, handle, p))
	}

	// Shader modules.
	for handle, object := range so.ShaderModules.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyShaderModule(object.Device, handle, p))
	}

	// Pipelines.
	for handle, object := range so.GraphicsPipelines.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device, handle, p))
	}
	for handle, object := range so.ComputePipelines.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipeline(object.Device, handle, p))
	}
	for handle, object := range so.PipelineLayouts.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineLayout(object.Device, handle, p))
	}
	for handle, object := range so.PipelineCaches.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyPipelineCache(object.Device, handle, p))
	}

	// Render passes.
	for handle, object := range so.RenderPasses.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyRenderPass(object.Device, handle, p))
	}

	for handle, object := range so.QueryPools.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyQueryPool(object.Device, handle, p))
	}

	// Command buffers.
	for handle, object := range so.CommandPools.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyCommandPool(object.Device, handle, p))
	}

	// Swapchains.
	for handle, object := range so.Swapchains.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySwapchainKHR(object.Device, handle, p))
	}

	// Memories.
	for handle, object := range so.DeviceMemories.Range() {
		out.MutateAndWrite(ctx, id, cb.VkFreeMemory(object.Device, handle, p))
	}

	// Note: so.Images also contains Swapchain images. We do not want
	// to delete those, as that must be handled by VkDestroySwapchainKHR
	for handle, object := range so.Images.Range() {
		if !object.IsSwapchainImage {
			out.MutateAndWrite(ctx, id, cb.VkDestroyImage(object.Device, handle, p))
		}
	}
	// Devices.
	for handle := range so.Devices.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyDevice(handle, p))
	}

	// Surfaces.
	for handle, object := range so.Surfaces.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroySurfaceKHR(object.Instance, handle, p))
	}

	// Instances.
	for handle := range so.Instances.Range() {
		out.MutateAndWrite(ctx, id, cb.VkDestroyInstance(handle, p))
	}
}

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct{}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out chan<- replay.Issue
}

func (a API) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	rrs []replay.RequestAndResult,
	device *device.Instance,
	capture *capture.Capture,
	out transform.Writer) error {
	if a.GetReplayPriority(ctx, device, capture.Header) == 0 {
		return log.Errf(ctx, nil, "Cannot replay Vulkan commands on device '%v'", device.Name)
	}

	optimize := !config.DisableDeadCodeElimination

	cmds := capture.Commands

	transforms := transform.Transforms{}
	transforms.Add(&makeAttachementReadable{
		VkInstance(0),
		make([]VkPhysicalDevice, 0),
		make(map[VkPhysicalDevice]VkPhysicalDeviceProperties),
	})

	readFramebuffer := newReadFramebuffer(ctx)
	injector := &transform.Injector{}
	// Gathers and reports any issues found.
	var issues *findIssues

	earlyTerminator, err := NewVulkanTerminator(ctx, intent.Capture)
	if err != nil {
		return err
	}

	// Populate the dead-code eliminitation later, only once we are sure
	// we will need it.
	dceInfo := dCEInfo{}
	initMem := interval.U64RangeList{}

	expandedCmds := false
	numInitialCommands := 0
	expandCommands := func() (int, error) {
		if expandedCmds {
			return numInitialCommands, nil
		}
		if optimize {
			// If we have not set up the dependency graph, do it now.
			if dceInfo.ft == nil {
				ft, err := dependencygraph.GetFootprint(ctx)
				if err != nil {
					return 0, err
				}
				dceInfo.ft = ft
				dceInfo.dce = transform.NewDCE(ctx, dceInfo.ft)
			}
			cmds = []api.Cmd{}
			numInitialCommands = dceInfo.ft.NumInitialCommands
		} else {
			// If the capture contains initial state, prepend the commands to build the state.
			initialCmds, im := capture.GetInitialCommands(ctx)
			initMem = im
			numInitialCommands = len(initialCmds)
			if len(initialCmds) > 0 {
				cmds = append(initialCmds, cmds...)
			}
		}
		expandedCmds = true
		return numInitialCommands, nil
	}

	wire := false

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case issuesRequest:
			if issues == nil {
				issues = newFindIssues(ctx, capture)
			}
			issues.reportTo(rr.Result)
			optimize = false

		case framebufferRequest:

			cfg := cfg.(drawConfig)
			if cfg.disableReplayOptimization {
				optimize = false
			}
			extraCommands, err := expandCommands()
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
				dceInfo.dce.Request(ctx, api.SubCmdIdx{cmdid})
			}

			switch cfg.wireframeMode {
			case replay.WireframeMode_All:
				wire = true
			case replay.WireframeMode_Overlay:
				return fmt.Errorf("Overlay wireframe view is not currently supported")
			}

			switch req.attachment {
			case api.FramebufferAttachment_Depth:
				readFramebuffer.Depth(after, req.framebufferIndex, rr.Result)
			case api.FramebufferAttachment_Stencil:
				return fmt.Errorf("Stencil attachments are not currently supported")
			default:
				readFramebuffer.Color(after, req.width, req.height, req.framebufferIndex, rr.Result)
			}
		}
	}

	_, err = expandCommands()
	if err != nil {
		return err
	}

	// Use the dead code elimination pass
	if optimize {
		transforms.Prepend(dceInfo.dce)
	}

	if wire {
		transforms.Add(wireframe(ctx))
	}

	if issues != nil {
		transforms.Add(issues) // Issue reporting required.
	} else {
		transforms.Add(earlyTerminator)
	}

	// Cleanup
	transforms.Add(readFramebuffer, injector)
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

	transforms.Transform(ctx, cmds, out)
	return nil
}

func (a API) QueryFramebufferAttachment(
	ctx context.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	wireframeMode replay.WireframeMode,
	disableReplayOptimization bool,
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

	c := drawConfig{beginIndex, endIndex, subcommand, wireframeMode, disableReplayOptimization}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, framebufferIndex: framebufferIndex, attachment: attachment, out: out}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
	if err != nil {
		return nil, err
	}
	return res.(*image.Data), nil
}

func (a API) QueryIssues(
	ctx context.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	hints *service.UsageHints) ([]replay.Issue, error) {

	c, r := issuesConfig{}, issuesRequest{}
	res, err := mgr.Replay(ctx, intent, c, r, a, hints)
	if err != nil {
		return nil, err
	}
	return res.([]replay.Issue), nil
}
