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

	"github.com/google/gapid/gapis/api"
)

func (st *State) getFramebufferAttachmentCount() (uint32, error) {
	if st.LastSubmission() == LastSubmissionType_SUBMIT {
		lastQueue := st.LastBoundQueue()
		if lastQueue.IsNil() {
			return 0, fmt.Errorf("No previous queue submission")
		}

		lastDrawInfo, ok := st.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
		if !ok {
			return 0, fmt.Errorf("There have been no previous draws")
		}

		if lastDrawInfo.Framebuffer().IsNil() || !st.Framebuffers().Contains(lastDrawInfo.Framebuffer().VulkanHandle()) {
			return 0, fmt.Errorf("framebuffer is not bound")
		}

		if lastDrawInfo.Framebuffer().RenderPass().IsNil() {
			return 0, fmt.Errorf("Renderpass is not bound")
		}

		return uint32(lastDrawInfo.Framebuffer().RenderPass().AttachmentDescriptions().Len()), nil
	}

	return st.LastPresentInfo().PresentImageCount(), nil
}

func (st *State) getSubmitAttachmentInfo(attachment uint32) (w, h uint32,
	f VkFormat, attachmentIndex uint32,
	canResize bool,
	attachmentType api.FramebufferAttachmentType,
	err error) {

	returnError := func(format_str string, e ...interface{}) (w, h uint32,
		f VkFormat, attachmentIndex uint32,
		canResize bool,
		attachmentType api.FramebufferAttachmentType,
		err error) {

		return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, true, api.FramebufferAttachmentType_OutputColor, fmt.Errorf(format_str, e...)
	}

	lastQueue := st.LastBoundQueue()
	if lastQueue.IsNil() {
		return returnError("No previous queue submission")
	}

	lastDrawInfo, ok := st.LastDrawInfos().Lookup(lastQueue.VulkanHandle())
	if !ok {
		return returnError("There have been no previous draws")
	}

	if lastDrawInfo.Framebuffer().IsNil() || !st.Framebuffers().Contains(lastDrawInfo.Framebuffer().VulkanHandle()) {
		return returnError("Attachment %d is not bound", attachment)
	}

	if lastDrawInfo.Framebuffer().RenderPass().IsNil() {
		return returnError("Attachment %d is not bound to any renderpass", attachment)
	}

	lastSubpass := lastDrawInfo.LastSubpass()

	subpassDesc := lastDrawInfo.Framebuffer().RenderPass().SubpassDescriptions().Get(lastSubpass)

	// Search for index in color attachments
	for _, attIndex := range subpassDesc.ColorAttachments().Keys() {
		attRef := subpassDesc.ColorAttachments().Get(attIndex)
		if attachment == attRef.Attachment() {
			if colorAttachment, ok := lastDrawInfo.FramebufferAttachments().Lookup(attachment); ok && !colorAttachment.IsNil() {
				colorImg := colorAttachment.Image()

				// This can occur if we destroy the image-view, we remove it from the framebuffer,
				// but may not unbind the framebuffer.
				if !colorImg.IsNil() {
					return colorImg.Info().Extent().Width(),
						colorImg.Info().Extent().Height(),
						colorImg.Info().Fmt(),
						attRef.Attachment(), true,
						api.FramebufferAttachmentType_OutputColor, nil
				}
			}
		}
	}

	// Search for index in input attachments
	for _, attIndex := range subpassDesc.InputAttachments().Keys() {
		attRef := subpassDesc.InputAttachments().Get(attIndex)
		if attachment == attRef.Attachment() {
			if inputAttachment, ok := lastDrawInfo.FramebufferAttachments().Lookup(attachment); ok && !inputAttachment.IsNil() {
				inputImg := inputAttachment.Image()

				if (VkImageAspectFlagBits(inputImg.ImageAspect()) & VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) != 0 {
					if !inputImg.IsNil() {
						return inputImg.Info().Extent().Width(),
							inputImg.Info().Extent().Height(),
							inputImg.Info().Fmt(),
							attRef.Attachment(), true,
							api.FramebufferAttachmentType_InputColor, nil
					}
				}

				if (VkImageAspectFlagBits(inputImg.ImageAspect()) & VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) != 0 {
					if !inputImg.IsNil() {
						return inputImg.Info().Extent().Width(),
							inputImg.Info().Extent().Height(),
							inputImg.Info().Fmt(),
							attRef.Attachment(), true,
							api.FramebufferAttachmentType_InputDepth, nil
					}
				}
			}
		}
	}

	// See if index corresponds to depth attachment
	if !subpassDesc.DepthStencilAttachment().IsNil() && attachment == subpassDesc.DepthStencilAttachment().Attachment() {
		if depthAttachment, ok := lastDrawInfo.FramebufferAttachments().Lookup(attachment); ok && !depthAttachment.IsNil() {
			depthImg := depthAttachment.Image()

			// This can occur if we destroy the image-view, we remove it from the framebuffer,
			// but may not unbind the framebuffer.
			if !depthImg.IsNil() {
				return depthImg.Info().Extent().Width(),
					depthImg.Info().Extent().Height(),
					depthImg.Info().Fmt(),
					subpassDesc.DepthStencilAttachment().Attachment(), true,
					api.FramebufferAttachmentType_OutputDepth, nil
			}
		}
	}

	// Ignore resolve attachments

	return returnError("Attachment %d is not bound", attachment)
}

func (st *State) getPresentAttachmentInfo(attachment uint32) (w, h uint32,
	f VkFormat,
	attachmentIndex uint32,
	canResize bool,
	attachmentType api.FramebufferAttachmentType,
	err error) {

	returnError := func(format_str string, e ...interface{}) (w, h uint32,
		f VkFormat,
		attachmentIndex uint32,
		canResize bool,
		attachmentType api.FramebufferAttachmentType,
		err error) {

		return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, false, api.FramebufferAttachmentType_OutputColor, fmt.Errorf(format_str, e...)
	}

	if st.LastPresentInfo().PresentImageCount() <= attachment {
		return returnError("Swapchain does not contain image %v", attachment)
	}
	colorImg := st.LastPresentInfo().PresentImages().Get(attachment)
	if !colorImg.IsNil() {
		queue := st.Queues().Get(st.LastPresentInfo().Queue())
		vkDevice := queue.Device()
		device := st.Devices().Get(vkDevice)
		vkPhysicalDevice := device.PhysicalDevice()
		physicalDevice := st.PhysicalDevices().Get(vkPhysicalDevice)
		if properties, ok := physicalDevice.QueueFamilyProperties().Lookup(queue.Family()); ok {
			if properties.QueueFlags()&VkQueueFlags(VkQueueFlagBits_VK_QUEUE_GRAPHICS_BIT) != 0 {
				return colorImg.Info().Extent().Width(),
					colorImg.Info().Extent().Height(),
					colorImg.Info().Fmt(), attachment, true,
					api.FramebufferAttachmentType_OutputColor, nil
			}
			return colorImg.Info().Extent().Width(),
				colorImg.Info().Extent().Height(),
				colorImg.Info().Fmt(), attachment, false,
				api.FramebufferAttachmentType_OutputColor, nil
		}

		return returnError("Last present queue does not exist", attachment)
	}

	return returnError("Swapchain attachment %d does not exist", attachment)
}

func (st *State) getFramebufferAttachmentInfo(attachment uint32) (uint32, uint32, VkFormat, uint32, bool, api.FramebufferAttachmentType, error) {
	if st.LastSubmission() == LastSubmissionType_SUBMIT {
		return st.getSubmitAttachmentInfo(attachment)
	}
	return st.getPresentAttachmentInfo(attachment)
}
