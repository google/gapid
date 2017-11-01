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

func (st *State) getSubmitAttachmentInfo(attachment api.FramebufferAttachment) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
	returnError := func(format_str string, e ...interface{}) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
		return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, fmt.Errorf(format_str, e...)
	}

	lastQueue := st.LastBoundQueue
	if lastQueue == nil {
		return returnError("No previous queue submission")
	}

	lastDrawInfo, ok := st.LastDrawInfos.Lookup(lastQueue.VulkanHandle)
	if !ok {
		return returnError("There have been no previous draws")
	}

	if lastDrawInfo.Framebuffer == nil {
		return returnError("%s is not bound", attachment)
	}

	if lastDrawInfo.Framebuffer.RenderPass == nil {
		return returnError("%s is not bound to any renderpass", attachment)
	}

	lastSubpass := lastDrawInfo.LastSubpass

	subpass_desc := lastDrawInfo.Framebuffer.RenderPass.SubpassDescriptions.Get(lastSubpass)
	switch attachment {
	case api.FramebufferAttachment_Color0,
		api.FramebufferAttachment_Color1,
		api.FramebufferAttachment_Color2,
		api.FramebufferAttachment_Color3:
		attachment_index := uint32(attachment - api.FramebufferAttachment_Color0)
		if att_ref, ok := subpass_desc.ColorAttachments.Lookup(attachment_index); ok {
			if ca, ok := lastDrawInfo.Framebuffer.ImageAttachments.Lookup(att_ref.Attachment); ok {
				return ca.Image.Info.Extent.Width, ca.Image.Info.Extent.Height, ca.Image.Info.Format, att_ref.Attachment, nil
			}

		}
	case api.FramebufferAttachment_Depth:
		if subpass_desc.DepthStencilAttachment != nil && lastDrawInfo.Framebuffer != nil {
			att_ref := subpass_desc.DepthStencilAttachment
			if attachment, ok := lastDrawInfo.Framebuffer.ImageAttachments.Lookup(att_ref.Attachment); ok {
				depth_img := attachment.Image
				return depth_img.Info.Extent.Width, depth_img.Info.Extent.Height, depth_img.Info.Format, att_ref.Attachment, nil
			}
		}
	case api.FramebufferAttachment_Stencil:
		fallthrough
	default:
		return returnError("Framebuffer attachment %v currently unsupported", attachment)
	}

	return returnError("%s is not bound", attachment)
}

func (st *State) getPresentAttachmentInfo(attachment api.FramebufferAttachment) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
	returnError := func(format_str string, e ...interface{}) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
		return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, fmt.Errorf(format_str, e...)
	}

	switch attachment {
	case api.FramebufferAttachment_Color0,
		api.FramebufferAttachment_Color1,
		api.FramebufferAttachment_Color2,
		api.FramebufferAttachment_Color3:
		image_idx := uint32(attachment - api.FramebufferAttachment_Color0)
		if st.LastPresentInfo.PresentImageCount <= image_idx {
			return returnError("Swapchain does not contain image %v", attachment)
		}
		color_img := st.LastPresentInfo.PresentImages.Get(image_idx)
		return color_img.Info.Extent.Width, color_img.Info.Extent.Height, color_img.Info.Format, image_idx, nil
	case api.FramebufferAttachment_Depth:
		fallthrough
	case api.FramebufferAttachment_Stencil:
		fallthrough
	default:
		return returnError("Swapchain attachment %v does not exist", attachment)
	}
}

func (st *State) getFramebufferAttachmentInfo(attachment api.FramebufferAttachment) (uint32, uint32, VkFormat, uint32, error) {
	if st.LastSubmission == LastSubmissionType_SUBMIT {
		return st.getSubmitAttachmentInfo(attachment)
	} else {
		return st.getPresentAttachmentInfo(attachment)
	}
}
