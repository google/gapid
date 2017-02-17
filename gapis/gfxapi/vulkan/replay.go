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
	"fmt"
	"strings"

	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/cause"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
)

var (
	// Interface compliance tests
	_ = replay.QueryIssues(api{})
	_ = replay.QueryFramebufferAttachment(api{})
	_ = replay.Support(api{})
)

// CanReplayOnLocalAndroidDevice returns true if the API can be replayed on a
// locally connected Android device.
func (a api) CanReplayOnLocalAndroidDevice(log.Context) bool { return true }

// CanReplayOn returns true if the API can be replayed on the specified device.
// Vulkan can currently only replay on Android devices.
func (a api) CanReplayOn(ctx log.Context, i *device.Instance) bool {
	return i.GetConfiguration().GetOS().GetKind() == device.Android
}

// makeAttachementReadable is a transformation marking all color/depth/stencil attachment
// images created via vkCreateImage atoms as readable (by patching the transfer src bit).
type makeAttachementReadable struct {
}

// drawConfig is a replay.Config used by colorBufferRequest and
// depthBufferRequests.
type drawConfig struct {
}

type imgRes struct {
	img *image.Image2D // The image data.
	err error          // The error that occurred generating the image.
}

// framebufferRequest requests a postback of a framebuffer's attachment.
type framebufferRequest struct {
	after            atom.ID
	width, height    uint32
	attachment       gfxapi.FramebufferAttachment
	out              chan imgRes
	wireframeOverlay bool
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

func (t *makeAttachementReadable) Transform(ctx log.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	s := out.State()
	a.Extras().Observations().ApplyReads(s.Memory[memory.ApplicationPool])
	if image, ok := a.(*VkCreateImage); ok {
		pinfo := image.PCreateInfo
		info := pinfo.Read(ctx, image, s, nil)

		if newUsage, changed := patchImageUsage(info.Usage); changed {
			device := image.Device
			palloc := memory.Pointer(image.PAllocator)
			pimage := memory.Pointer(image.PImage)
			result := image.Result

			info.Usage = newUsage
			newInfo := atom.Must(atom.AllocData(ctx, s, info))
			newAtom := NewVkCreateImage(device, newInfo.Ptr(), palloc, pimage, result)
			// Carry all non-observation extras through.
			for _, e := range image.Extras().All() {
				if _, ok := e.(*atom.Observations); !ok {
					newAtom.Extras().Add(e)
				}
			}
			// Carry observations through. We cannot merge these code with the
			// above code for handling extras together since we'd like to change
			// the observations, which are slices.
			observations := image.Extras().Observations()
			for _, r := range observations.Reads {
				// TODO: filter out the old VkImageCreateInfo. That should be done via
				// creating new observations for data we are interested from t.state.
				newAtom.AddRead(r.Range, r.ID)
			}
			// Use our new VkImageCreateInfo.
			newAtom.AddRead(newInfo.Data())
			for _, w := range observations.Writes {
				newAtom.AddWrite(w.Range, w.ID)
			}
			out.MutateAndWrite(ctx, id, newAtom)
			return
		}
	} else if swapchain, ok := a.(*VkCreateSwapchainKHR); ok {
		pinfo := swapchain.PCreateInfo
		info := pinfo.Read(ctx, swapchain, s, nil)

		if newUsage, changed := patchImageUsage(info.ImageUsage); changed {
			device := swapchain.Device
			palloc := memory.Pointer(swapchain.PAllocator)
			pswapchain := memory.Pointer(swapchain.PSwapchain)
			result := swapchain.Result

			info.ImageUsage = newUsage
			newInfo := atom.Must(atom.AllocData(ctx, s, info))
			newAtom := NewVkCreateSwapchainKHR(device, newInfo.Ptr(), palloc, pswapchain, result)
			for _, e := range swapchain.Extras().All() {
				if _, ok := e.(*atom.Observations); !ok {
					newAtom.Extras().Add(e)
				}
			}
			observations := swapchain.Extras().Observations()
			for _, r := range observations.Reads {
				// TODO: filter out the old VkSwapchainCreateInfoKHR. That should be done via
				// creating new observations for data we are interested from t.state.
				newAtom.AddRead(r.Range, r.ID)
			}
			newAtom.AddRead(newInfo.Data())
			for _, w := range observations.Writes {
				newAtom.AddWrite(w.Range, w.ID)
			}
			out.MutateAndWrite(ctx, id, newAtom)
			return
		}
	} else if createRenderPass, ok := a.(*VkCreateRenderPass); ok {
		pInfo := createRenderPass.PCreateInfo
		info := pInfo.Read(ctx, createRenderPass, s, nil)
		pAttachments := info.PAttachments
		attachments := pAttachments.Slice(uint64(0), uint64(info.AttachmentCount), s).Read(ctx, createRenderPass, s, nil)
		changed := false
		for i := range attachments {
			if attachments[i].StoreOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE {
				changed = true
				attachments[i].StoreOp = VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE
			}
		}
		// Returns if no attachment description needs to be changed
		if !changed {
			out.MutateAndWrite(ctx, id, a)
			return
		}
		// Build new attachments data, new create info and new atom
		newAttachments := atom.Must(atom.AllocData(ctx, s, attachments))
		info.PAttachments = VkAttachmentDescriptionᶜᵖ(newAttachments.Ptr())
		newInfo := atom.Must(atom.AllocData(ctx, s, info))
		newAtom := NewVkCreateRenderPass(createRenderPass.Device,
			newInfo.Ptr(),
			memory.Pointer(createRenderPass.PAllocator),
			memory.Pointer(createRenderPass.PRenderPass),
			createRenderPass.Result)
		// Add back the extras and read/write observations
		for _, e := range createRenderPass.Extras().All() {
			if _, ok := e.(*atom.Observations); !ok {
				newAtom.Extras().Add(e)
			}
		}
		for _, r := range createRenderPass.Extras().Observations().Reads {
			newAtom.AddRead(r.Range, r.ID)
		}
		newAtom.AddRead(newInfo.Data()).AddRead(newAttachments.Data())
		for _, w := range createRenderPass.Extras().Observations().Writes {
			newAtom.AddWrite(w.Range, w.ID)
		}
		out.MutateAndWrite(ctx, id, newAtom)
		return
	}
	out.MutateAndWrite(ctx, id, a)
}

func (t *makeAttachementReadable) Flush(ctx log.Context, out transform.Writer) {}

// destroyResourceAtEOS is a transformation that destroys all active
// resources at the end of stream.
type destroyResourcesAtEOS struct {
}

func (t *destroyResourcesAtEOS) Transform(ctx log.Context, id atom.ID, a atom.Atom, out transform.Writer) {
	out.MutateAndWrite(ctx, id, a)
}

func (t *destroyResourcesAtEOS) Flush(ctx log.Context, out transform.Writer) {
	s := out.State()
	so := getStateObject(s)
	id := atom.NoID
	// TODO: use the correct pAllocator once we handle it.
	p := memory.Nullptr

	// Wait all queues in all devices to finish their jobs first.
	for handle := range so.Devices {
		out.MutateAndWrite(ctx, id, NewVkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}

	// Synchronization primitives.
	for handle, object := range so.Events {
		out.MutateAndWrite(ctx, id, NewVkDestroyEvent(object.Device, handle, p))
	}
	for handle, object := range so.Fences {
		out.MutateAndWrite(ctx, id, NewVkDestroyFence(object.Device, handle, p))
	}
	for handle, object := range so.Semaphores {
		out.MutateAndWrite(ctx, id, NewVkDestroySemaphore(object.Device, handle, p))
	}

	// Framebuffers, samplers.
	for handle, object := range so.Framebuffers {
		out.MutateAndWrite(ctx, id, NewVkDestroyFramebuffer(object.Device, handle, p))
	}
	for handle, object := range so.Samplers {
		out.MutateAndWrite(ctx, id, NewVkDestroySampler(object.Device, handle, p))
	}

	for handle, object := range so.ImageViews {
		out.MutateAndWrite(ctx, id, NewVkDestroyImageView(object.Device, handle, p))
	}

	// Buffers.
	for handle, object := range so.BufferViews {
		out.MutateAndWrite(ctx, id, NewVkDestroyBufferView(object.Device, handle, p))
	}
	for handle, object := range so.Buffers {
		out.MutateAndWrite(ctx, id, NewVkDestroyBuffer(object.Device, handle, p))
	}

	// Descriptor sets.
	for handle, object := range so.DescriptorPools {
		out.MutateAndWrite(ctx, id, NewVkDestroyDescriptorPool(object.Device, handle, p))
	}
	for handle, object := range so.DescriptorSetLayouts {
		out.MutateAndWrite(ctx, id, NewVkDestroyDescriptorSetLayout(object.Device, handle, p))
	}

	// Shader modules.
	for handle, object := range so.ShaderModules {
		out.MutateAndWrite(ctx, id, NewVkDestroyShaderModule(object.Device, handle, p))
	}

	// Pipelines.
	for handle, object := range so.GraphicsPipelines {
		out.MutateAndWrite(ctx, id, NewVkDestroyPipeline(object.Device, handle, p))
	}
	for handle, object := range so.ComputePipelines {
		out.MutateAndWrite(ctx, id, NewVkDestroyPipeline(object.Device, handle, p))
	}
	for handle, object := range so.PipelineLayouts {
		out.MutateAndWrite(ctx, id, NewVkDestroyPipelineLayout(object.Device, handle, p))
	}
	for handle, object := range so.PipelineCaches {
		out.MutateAndWrite(ctx, id, NewVkDestroyPipelineCache(object.Device, handle, p))
	}

	// Render passes.
	for handle, object := range so.RenderPasses {
		out.MutateAndWrite(ctx, id, NewVkDestroyRenderPass(object.Device, handle, p))
	}

	for handle, object := range so.QueryPools {
		out.MutateAndWrite(ctx, id, NewVkDestroyQueryPool(object.Device, handle, p))
	}

	// Command buffers.
	for handle, object := range so.CommandPools {
		out.MutateAndWrite(ctx, id, NewVkDestroyCommandPool(object.Device, handle, p))
	}

	// Swapchains.
	for handle, object := range so.Swapchains {
		out.MutateAndWrite(ctx, id, NewVkDestroySwapchainKHR(object.Device, handle, p))
	}

	// Memories.
	for handle, object := range so.DeviceMemories {
		out.MutateAndWrite(ctx, id, NewVkFreeMemory(object.Device, handle, p))
	}

	// Note: so.Images also contains Swapchain images. We do not want
	// to delete those, as that must be handled by VkDestroySwapchainKHR
	for handle, object := range so.Images {
		if !object.IsSwapchainImage {
			out.MutateAndWrite(ctx, id, NewVkDestroyImage(object.Device, handle, p))
		}
	}
	// Devices.
	for handle := range so.Devices {
		out.MutateAndWrite(ctx, id, NewVkDestroyDevice(handle, p))
	}

	// Surfaces.
	for handle, object := range so.Surfaces {
		out.MutateAndWrite(ctx, id, NewVkDestroySurfaceKHR(object.Instance, handle, p))
	}

	// Instances.
	for handle := range so.Instances {
		out.MutateAndWrite(ctx, id, NewVkDestroyInstance(handle, p))
	}
}

// issuesConfig is a replay.Config used by issuesRequests.
type issuesConfig struct{}

// issuesRequest requests all issues found during replay to be reported to out.
type issuesRequest struct {
	out chan<- replay.Issue
}

func (a api) Replay(
	ctx log.Context,
	intent replay.Intent,
	cfg replay.Config,
	requests []replay.Request,
	device *device.Instance,
	capture *capture.Capture,
	out transform.Writer) error {

	atoms, err := capture.Atoms(ctx)
	if err != nil {
		return cause.Explain(ctx, err, "Failed to load atom stream")
	}

	transforms := transform.Transforms{}
	transforms.Add(&makeAttachementReadable{})

	readFramebuffer := newReadFramebuffer(ctx)
	injector := &transform.Injector{}
	// Gathers and reports any issues found.
	var issues *findIssues

	// Terminate after all atoms of interest.
	earlyTerminator := &transform.EarlyTerminator{}

	for _, req := range requests {
		switch req := req.(type) {
		case issuesRequest:
			if issues == nil {
				issues = &findIssues{}
			}
			issues.reportTo(req.out)

		case framebufferRequest:
			earlyTerminator.Add(req.after)
			switch req.attachment {
			case gfxapi.FramebufferAttachment_Depth:
				readFramebuffer.Depth(req.after, req.out)
			case gfxapi.FramebufferAttachment_Stencil:
				return fmt.Errorf("Stencil attachments are not currently supported")
			default:
				idx := uint32(req.attachment - gfxapi.FramebufferAttachment_Color0)
				readFramebuffer.Color(req.after, req.width, req.height, idx, req.out)
			}
		}
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
		ctx.Info().Logf("Replaying %d atoms using transform chain:", len(atoms.Atoms))
		for i, t := range transforms {
			ctx.Info().Logf("(%d) %#v", i, t)
		}
	}

	if config.LogTransformsToFile {
		newTransforms := transform.Transforms{}
		newTransforms.Add(transform.NewFileLog(ctx, "0_original_atoms"))
		for i, t := range transforms {
			var name string
			if n, ok := t.(interface {
				Name() string
			}); ok {
				name = n.Name()
			} else {
				name = strings.Replace(fmt.Sprintf("%T", t), "*", "", -1)
			}
			newTransforms.Add(t, transform.NewFileLog(ctx, fmt.Sprintf("%v_atoms_after_%v", i+1, name)))
		}
		transforms = newTransforms
	}

	return catchPanics(ctx, func() { transforms.Transform(ctx, *atoms, out) })
}

func catchPanics(ctx log.Context, do func()) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = cause.Explainf(ctx, nil, "Panic raised: %v", e)
		}
	}()
	do()
	return nil
}

func (a api) QueryFramebufferAttachment(
	ctx log.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	after atom.ID,
	width, height uint32,
	attachment gfxapi.FramebufferAttachment,
	wireframeMode replay.WireframeMode) (*image.Image2D, error) {

	c := drawConfig{}
	out := make(chan imgRes, 1)
	r := framebufferRequest{after: after, width: width, height: height, attachment: attachment, out: out}
	if err := mgr.Replay(ctx, intent, c, r, a); err != nil {
		return nil, err
	}
	select {
	case res := <-out:
		return res.img, res.err
	case <-task.ShouldStop(ctx):
		return nil, task.StopReason(ctx)
	}
}

func (a api) QueryIssues(
	ctx log.Context,
	intent replay.Intent,
	mgr *replay.Manager,
	out chan<- replay.Issue) {

	c := issuesConfig{}
	r := issuesRequest{out: out}
	if err := mgr.Replay(ctx, intent, c, r, a); err != nil {
		out <- replay.Issue{Atom: atom.NoID, Error: err}
		close(out)
	}
}
