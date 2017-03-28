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

	"github.com/google/gapid/gapis/gfxapi"
)

func (st *State) getFramebufferAttachmentInfo(attachment gfxapi.FramebufferAttachment) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
	returnError := func(format_str string, e ...interface{}) (w, h uint32, f VkFormat, attachmentIndex uint32, err error) {
		return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, fmt.Errorf(format_str, e...)
	}

	if st.LastDrawInfo.Framebuffer == nil {
		return returnError("%s is not bound", attachment)
	}

	if st.LastDrawInfo.Framebuffer.RenderPass == nil {
		return returnError("%s is not bound to any renderpass", attachment)
	}

	lastSubpass := st.LastDrawInfo.LastSubpass

	subpass_desc := st.LastDrawInfo.Framebuffer.RenderPass.SubpassDescriptions[lastSubpass]
	switch attachment {
	case gfxapi.FramebufferAttachment_Color0,
		gfxapi.FramebufferAttachment_Color1,
		gfxapi.FramebufferAttachment_Color2,
		gfxapi.FramebufferAttachment_Color3:
		num_of_color_att_before_the_query_one := attachment - gfxapi.FramebufferAttachment_Color0
		for _, att_ref_index := range subpass_desc.ColorAttachments.KeysSorted() {
			att_ref := subpass_desc.ColorAttachments[att_ref_index]
			color_img := st.LastDrawInfo.Framebuffer.ImageAttachments[att_ref.Attachment].Image
			if uint32(color_img.Info.Usage)&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT) != 0 {
				if num_of_color_att_before_the_query_one == 0 {
					return color_img.Info.Extent.Width, color_img.Info.Extent.Height, color_img.Info.Format, att_ref.Attachment, nil
				} else {
					num_of_color_att_before_the_query_one -= 1
				}
			}
		}
	case gfxapi.FramebufferAttachment_Depth:
		if subpass_desc.DepthStencilAttachment != nil {
			att_ref := subpass_desc.DepthStencilAttachment
			depth_img := st.LastDrawInfo.Framebuffer.ImageAttachments[att_ref.Attachment].Image
			if (uint32(depth_img.Info.Usage)&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT) != 0) &&
				(depth_img.Info.Samples == VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT) {
				return depth_img.Info.Extent.Width, depth_img.Info.Extent.Height, depth_img.Info.Format, att_ref.Attachment, nil
			}
		}
	case gfxapi.FramebufferAttachment_Stencil:
		fallthrough
	default:
		return returnError("Framebuffer attachment %v currently unsupported", attachment)
	}

	return returnError("%s is not bound", attachment)
}
