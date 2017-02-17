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
	if st.LastUsedFramebuffer != nil {
		var index int

		switch attachment {
		case gfxapi.FramebufferAttachment_Color0:
			index = 0
		case gfxapi.FramebufferAttachment_Color1:
			index = 1
		case gfxapi.FramebufferAttachment_Color2:
			index = 2
		case gfxapi.FramebufferAttachment_Color3:
			index = 3
		case gfxapi.FramebufferAttachment_Depth:
			break
		case gfxapi.FramebufferAttachment_Stencil:
			fallthrough
		default:
			return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, fmt.Errorf("Framebuffer attachment %v currently unsupported", attachment)
		}

		currentColorIndex := 0
		for _, a := range st.LastUsedFramebuffer.ImageAttachments.KeysSorted() {
			view := st.LastUsedFramebuffer.ImageAttachments[a]
			i := view.Image
			layer0 := i.Layers[0]
			level0 := layer0.Levels[0]

			switch attachment {
			case gfxapi.FramebufferAttachment_Depth:
				if 0 != (uint32(i.Info.Usage) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) {
					// Use the first-found depth image that is not multi-sampled.
					if i.Info.Samples != VkSampleCountFlagBits_VK_SAMPLE_COUNT_1_BIT {
						continue
					}
					return level0.Width, level0.Height, i.Info.Format, a, nil
				}
			case gfxapi.FramebufferAttachment_Stencil:
				if 0 != (uint32(i.Info.Usage) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) {
					return level0.Width, level0.Height, i.Info.Format, a, nil
				}
			default:
				if 0 != (uint32(i.Info.Usage) & uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)) {
					continue
				}
				if currentColorIndex == index {
					return level0.Width, level0.Height, i.Info.Format, a, nil
				} else {
					currentColorIndex += 1
				}
			}
		}
	}
	return 0, 0, VkFormat_VK_FORMAT_UNDEFINED, 0, fmt.Errorf("%s is not bound", attachment)
}
