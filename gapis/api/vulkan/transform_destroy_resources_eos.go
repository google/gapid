// Copyright (C) 2020 Google Inc.
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
	"github.com/google/gapid/gapis/api/transform2"
	"github.com/google/gapid/gapis/memory"
)

var _ transform2.Transform = &destroyResourcesAtEOS{}

type destroyResourcesAtEOS struct {
}

func newDestroyResourcesAtEOS() *destroyResourcesAtEOS {
	return &destroyResourcesAtEOS{}
}

func (dropTransform *destroyResourcesAtEOS) RequiresAccurateState() bool {
	return false
}

func (dropTransform *destroyResourcesAtEOS) RequiresInnerStateMutation() bool {
	return false
}

func (dropTransform *destroyResourcesAtEOS) SetInnerStateMutationFunction(mutator transform2.StateMutator) {
	// This transform do not require inner state mutation
}

func (dropTransform *destroyResourcesAtEOS) BeginTransform(ctx context.Context, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (dropTransform *destroyResourcesAtEOS) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	vulkanState := GetState(inputState)

	cb := CommandBuilder{Thread: 0, Arena: inputState.Arena} // TODO: Check that using any old thread is okay.
	// TODO: use the correct pAllocator once we handle it.
	p := memory.Nullptr

	cleanupCommands := make([]api.Cmd, 0)

	// Wait all queues in all devices to finish their jobs first.
	for handle := range vulkanState.Devices().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}

	// Synchronization primitives.
	for handle, object := range vulkanState.Events().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyEvent(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.Fences().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyFence(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.Semaphores().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroySemaphore(object.Device(), handle, p))
	}

	// SamplerYcbcrConversions
	for handle, object := range vulkanState.SamplerYcbcrConversions().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroySamplerYcbcrConversion(object.Device(), handle, p))
	}

	// Framebuffers, samplers.
	for handle, object := range vulkanState.Framebuffers().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyFramebuffer(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.Samplers().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroySampler(object.Device(), handle, p))
	}

	// Descriptor sets.
	for handle, object := range vulkanState.DescriptorPools().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyDescriptorPool(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.DescriptorSetLayouts().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyDescriptorSetLayout(object.Device(), handle, p))
	}

	// Buffers.
	for handle, object := range vulkanState.BufferViews().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyBufferView(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.Buffers().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyBuffer(object.Device(), handle, p))
	}

	// Shader modules.
	for handle, object := range vulkanState.ShaderModules().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyShaderModule(object.Device(), handle, p))
	}

	// Pipelines.
	for handle, object := range vulkanState.GraphicsPipelines().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyPipeline(object.Device(), handle, p))
	}
	for handle, object := range vulkanState.ComputePipelines().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyPipeline(object.Device(), handle, p))
	}
	for handle, object := range vulkanState.PipelineLayouts().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyPipelineLayout(object.Device(), handle, p))
	}
	for handle, object := range vulkanState.PipelineCaches().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyPipelineCache(object.Device(), handle, p))
	}

	// Render passes.
	for handle, object := range vulkanState.RenderPasses().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyRenderPass(object.Device(), handle, p))
	}

	for handle, object := range vulkanState.QueryPools().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyQueryPool(object.Device(), handle, p))
	}

	// Command buffers.
	for handle, object := range vulkanState.CommandPools().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyCommandPool(object.Device(), handle, p))
	}

	// Swapchains.
	for handle, object := range vulkanState.Swapchains().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroySwapchainKHR(object.Device(), handle, p))
	}

	// Memories.
	for handle, object := range vulkanState.DeviceMemories().All() {
		cleanupCommands = append(cleanupCommands, cb.VkFreeMemory(object.Device(), handle, p))
	}

	// Images
	for handle, object := range vulkanState.ImageViews().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyImageView(object.Device(), handle, p))
	}
	// Note: vulkanState.Images also contains Swapchain images. We do not want
	// to delete those, as that must be handled by VkDestroySwapchainKHR
	for handle, object := range vulkanState.Images().All() {
		if !object.IsSwapchainImage() {
			cleanupCommands = append(cleanupCommands, cb.VkDestroyImage(object.Device(), handle, p))
		}
	}

	// Devices.
	for handle := range vulkanState.Devices().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyDevice(handle, p))
	}

	// Surfaces.
	for handle, object := range vulkanState.Surfaces().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroySurfaceKHR(object.Instance(), handle, p))
	}

	// Debug report callbacks
	for handle, object := range vulkanState.DebugReportCallbacks().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyDebugReportCallbackEXT(object.Instance(), handle, p))
	}

	// Instances.
	for handle := range vulkanState.Instances().All() {
		cleanupCommands = append(cleanupCommands, cb.VkDestroyInstance(handle, p))
	}

	return cleanupCommands, nil
}

func (dropTransform *destroyResourcesAtEOS) TransformCommand(ctx context.Context, id transform2.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	return inputCommands, nil
}

func (dropTransform *destroyResourcesAtEOS) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}
