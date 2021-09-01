// Copyright (C) 2021 Google Inc.
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

	"github.com/google/gapid/gapis/api"
)

// This file contains Label functions for each VkHandle (VkInstenace, VkDevice,
// etc). Adding this function to each handle type ensures that it implements the
// Labeled (labeled.go) interface, which is part of the Handle (handle.go)
// interface.
// When updating this file, you probably also want to update links.go as it
// ensures the handles implement the path.Linker interface.

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkInstance) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Instances().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkPhysicalDevice) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).PhysicalDevices().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDevice) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Devices().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkQueue) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Queues().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkFramebuffer) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Framebuffers().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkSampler) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Samplers().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDescriptorSetLayout) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DescriptorSetLayouts().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDescriptorPool) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DescriptorPools().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDescriptorSet) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DescriptorSets().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkPipelineLayout) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).PipelineLayouts().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkShaderModule) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).ShaderModules().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkRenderPass) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).RenderPasses().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkCommandPool) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).CommandPools().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkCommandBuffer) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).CommandBuffers().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDeviceMemory) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DeviceMemories().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkBuffer) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Buffers().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkBufferView) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).BufferViews().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkImage) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Images().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkImageView) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).ImageViews().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkFence) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Fences().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkSemaphore) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Semaphores().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkEvent) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Events().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkQueryPool) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).QueryPools().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkPipelineCache) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).PipelineCaches().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkSurfaceKHR) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Surfaces().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkSwapchainKHR) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).Swapchains().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDisplayKHR) Label(ctx context.Context, s *api.GlobalState) string {
	// TODO: we don't currently track these in the state.
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDisplayModeKHR) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DisplayModes().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDebugReportCallbackEXT) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DebugReportCallbacks().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkSamplerYcbcrConversion) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).SamplerYcbcrConversions().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkPipeline) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).GraphicsPipelines().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	} else if obj, ok := GetState(s).ComputePipelines().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDescriptorUpdateTemplate) Label(ctx context.Context, s *api.GlobalState) string {
	if obj, ok := GetState(s).DescriptorUpdateTemplates().Lookup(h); ok {
		if !obj.DebugInfo().IsNil() && len(obj.DebugInfo().ObjectName()) > 0 {
			return obj.DebugInfo().ObjectName()
		}
	}
	return ""
}

// Label implements the Labeled interface returning the debug label of the
// object this handle represents.
func (h VkDebugUtilsMessengerEXT) Label(ctx context.Context, s *api.GlobalState) string {
	// TODO: we don't currently track these in the state.
	return ""
}
