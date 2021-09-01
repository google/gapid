// Copyright (C) 2019 Google Inc.
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

	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

// This file contains Link functions for each VkHandle (VkInstenace, VkDevice,
// etc). Adding this function to each handle type ensures that it implements the
// path.Linker (path/linker.go) interface.
// When updating this file, you probably also want to update labels.go as it
// ensures the handles implement the Labeled interface.

// state returns the state at p.
func state(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, *State, error) {
	if cmdPath := path.FindCommand(p); cmdPath != nil {
		stateObj, err := resolve.State(ctx, cmdPath.StateAfter(), r)
		if err != nil {
			return nil, nil, err
		}
		state := stateObj.(*State)
		return resolve.APIStateAfter(cmdPath, ID), state, nil
	}
	return nil, nil, fmt.Errorf("Invalid path for Link")
}

// Link returns the link to the instance in the state block.
func (o VkInstance) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Instances().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Instances", i).MapIndex(o), nil
}

// Link returns the link to the physical device in the state block.
func (o VkPhysicalDevice) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.PhysicalDevices().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("PhysicalDevices", i).MapIndex(o), nil
}

// Link returns the link to the device in the state block.
func (o VkDevice) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Devices().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Devices", i).MapIndex(o), nil
}

// Link returns the link to the queue in the state block.
func (o VkQueue) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Queues().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Queues", i).MapIndex(o), nil
}

// Link returns the link to the framebuffer in the state block.
func (o VkFramebuffer) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Framebuffers().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Framebuffers", i).MapIndex(o), nil
}

// Link returns the link to the sampler in the state block.
func (o VkSampler) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Samplers().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Samplers", i).MapIndex(o), nil
}

// Link returns the link to the descriptor set layout in the state block.
func (o VkDescriptorSetLayout) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DescriptorSetLayouts().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DescriptorSetLayouts", i).MapIndex(o), nil
}

// Link returns the link to the descriptor pool in the state block.
func (o VkDescriptorPool) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DescriptorPools().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DescriptorPools", i).MapIndex(o), nil
}

// Link returns the link to the descriptor set in the state block.
func (o VkDescriptorSet) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DescriptorSets().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DescriptorSets", i).MapIndex(o), nil
}

// Link returns the link to the pipeline layout in the state block.
func (o VkPipelineLayout) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.PipelineLayouts().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("PipelineLayouts", i).MapIndex(o), nil
}

// Link returns the link to the shader module in the state block.
func (o VkShaderModule) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.ShaderModules().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("ShaderModules", i).MapIndex(o), nil
}

// Link returns the link to the renderpass in the state block.
func (o VkRenderPass) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.RenderPasses().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("RenderPasses", i).MapIndex(o), nil
}

// Link returns the link to the command pool in the state block.
func (o VkCommandPool) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.CommandPools().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("CommandPools", i).MapIndex(o), nil
}

// Link returns the link to the command buffer in the state block.
func (o VkCommandBuffer) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.CommandBuffers().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("CommandBuffers", i).MapIndex(o), nil
}

// Link returns the link to the device memory in the state block.
func (o VkDeviceMemory) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DeviceMemories().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DeviceMemories", i).MapIndex(o), nil
}

// Link returns the link to the buffer in the state block.
func (o VkBuffer) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Buffers().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Buffers", i).MapIndex(o), nil
}

// Link returns the link to the bufferview in the state block.
func (o VkBufferView) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.BufferViews().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("BufferViews", i).MapIndex(o), nil
}

// Link returns the link to the image in the state block.
func (o VkImage) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Images().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Images", i).MapIndex(o), nil
}

// Link returns the link to the imageview in the state block.
func (o VkImageView) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.ImageViews().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("ImageViews", i).MapIndex(o), nil
}

// Link returns the link to the fence in the state block.
func (o VkFence) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Fences().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Fences", i).MapIndex(o), nil
}

// Link returns the link to the semaphore in the state block.
func (o VkSemaphore) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Semaphores().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Semaphores", i).MapIndex(o), nil
}

// Link returns the link to the event in the state block.
func (o VkEvent) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Events().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Events", i).MapIndex(o), nil
}

// Link returns the link to the query pool in the state block.
func (o VkQueryPool) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.QueryPools().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("QueryPools", i).MapIndex(o), nil
}

// Link returns the link to the pipeline cache in the state block.
func (o VkPipelineCache) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.PipelineCaches().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("PipelineCaches", i).MapIndex(o), nil
}

// Link returns the link to the surface in the state block.
func (o VkSurfaceKHR) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Surfaces().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Surfaces", i).MapIndex(o), nil
}

// Link returns the link to the swapchain in the state block.
func (o VkSwapchainKHR) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.Swapchains().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("Swapchains", i).MapIndex(o), nil
}

// Link returns the link to the display mode in the state block.
func (o VkDisplayModeKHR) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DisplayModes().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DisplayModes", i).MapIndex(o), nil
}

// Link returns the link to the debug report callback in the state block.
func (o VkDebugReportCallbackEXT) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.DebugReportCallbacks().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("DebugReportCallbacks", i).MapIndex(o), nil
}

// Link returns the link to the ycbcr conversion in the state block.
func (o VkSamplerYcbcrConversion) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if !c.SamplerYcbcrConversions().Contains(o) {
		return nil, fmt.Errorf("State does not contain link target")
	}
	return path.NewField("SamplerYcbcrConversions", i).MapIndex(o), nil
}

// Link returns the link to the pipeline in the state block.
func (o VkPipeline) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if c.GraphicsPipelines().Contains(o) {
		return path.NewField("GraphicsPipelines", i).MapIndex(o), nil
	} else if c.ComputePipelines().Contains(o) {
		return path.NewField("ComputePipelines", i).MapIndex(o), nil
	} else {
		return nil, fmt.Errorf("State does not contain link target")
	}
}

// Link returns the link to the descriptorUpdateTemplate in the state block.
func (o VkDescriptorUpdateTemplate) Link(ctx context.Context, p path.Node, r *path.ResolveConfig) (path.Node, error) {
	i, c, err := state(ctx, p, r)
	if err != nil {
		return nil, err
	}
	if c.DescriptorUpdateTemplates().Contains(o) {
		return path.NewField("DescriptorUpdateTemplates", i).MapIndex(o), nil
	} else {
		return nil, fmt.Errorf("State does not contain link target")
	}
}
