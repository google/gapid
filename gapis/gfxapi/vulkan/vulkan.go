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

// Package vulkan implementes the API interface for the Vulkan graphics library.
package vulkan

// binary: cpp = vulkan

import (
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/gfxapi"
)

func getStateObject(s *gfxapi.State) *State {
	return GetState(s)
}

type VulkanContext struct{}

func (VulkanContext) Name() string {
	return "Vulkan Context"
}

func (VulkanContext) ID() gfxapi.ContextID {
	// ID returns the context's unique identifier
	return gfxapi.ContextID{1}
}

func (api) Context(s *gfxapi.State) gfxapi.Context {
	return VulkanContext{}
}

func (api) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (w, h uint32, f *image.Format, err error) {
	w, h, form, _, err := GetState(state).getFramebufferAttachmentInfo(attachment)
	switch attachment {
	case gfxapi.FramebufferAttachment_Stencil:
		return 0, 0, nil, fmt.Errorf("Unsupported Stencil")
	case gfxapi.FramebufferAttachment_Depth:
		format, err := getDepthImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("Unknown format for Depth attachment")
		}
		return w, h, format, err
	default:
		format, err := getImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("Unknown format for Color attachment")
		}
		return w, h, format, err
	}
}
