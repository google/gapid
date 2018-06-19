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
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/resolve/initialcmds"
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
	drawMode                  replay.DrawMode
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
		allProps := externs{ctx, cmd, id, s, nil}.fetchPhysicalDeviceProperties(e.Instance(), devSlice)
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
	transforms.Add(&makeAttachementReadable{})

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

	initCmdExpandedWithOpt := false
	numInitialCmdWithOpt := 0
	initCmdExpandedWithoutOpt := false
	numInitialCmdWithoutOpt := 0
	expandCommands := func(opt bool) (int, error) {
		if opt && initCmdExpandedWithOpt {
			return numInitialCmdWithOpt, nil
		}
		if !opt && initCmdExpandedWithoutOpt {
			return numInitialCmdWithoutOpt, nil
		}
		if opt {
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
			numInitialCmdWithOpt = dceInfo.ft.NumInitialCommands
			initCmdExpandedWithOpt = true
			return numInitialCmdWithOpt, nil
		}
		// If the capture contains initial state, prepend the commands to build the state.
		initialCmds, im, _ := initialcmds.InitialCommands(ctx, intent.Capture)
		out.State().Allocator.ReserveRanges(im)
		numInitialCmdWithoutOpt = len(initialCmds)
		if len(initialCmds) > 0 {
			cmds = append(initialCmds, cmds...)
		}
		initCmdExpandedWithoutOpt = true
		return numInitialCmdWithoutOpt, nil
	}

	wire := false
	overdraw := newStencilOverdraw()

	for _, rr := range rrs {
		switch req := rr.Request.(type) {
		case issuesRequest:
			if issues == nil {
				n, err := expandCommands(false)
				if err != nil {
					return err
				}
				issues = newFindIssues(ctx, capture, n)
			}
			issues.reportTo(rr.Result)
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
				dceInfo.dce.Request(ctx, api.SubCmdIdx{cmdid})
			}

			switch cfg.drawMode {
			case replay.DrawMode_WIREFRAME_ALL:
				wire = true
			case replay.DrawMode_WIREFRAME_OVERLAY:
				return fmt.Errorf("Overlay wireframe view is not currently supported")
			case replay.DrawMode_OVERDRAW:
				overdraw.add(ctx, req.after, intent.Capture, rr.Result)
			}

			if cfg.drawMode != replay.DrawMode_OVERDRAW {
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
	}

	_, err = expandCommands(optimize)
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

	transforms.Add(overdraw)

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
	if config.LogTransformsToCapture {
		transforms.Add(transform.NewCaptureLog(ctx, capture, "replay_log.gfxtrace"))
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
	mgr *replay.Manager,
	after []uint64,
	width, height uint32,
	attachment api.FramebufferAttachment,
	framebufferIndex uint32,
	drawMode replay.DrawMode,
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

	c := drawConfig{beginIndex, endIndex, subcommand, drawMode, disableReplayOptimization}
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
