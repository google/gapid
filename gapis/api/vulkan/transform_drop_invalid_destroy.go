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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

var _ transform.Transform = &dropInvalidDestroy{}

type dropInvalidDestroy struct {
	tag         string
	allocations *allocationTracker
}

func newDropInvalidDestroy(tag string) *dropInvalidDestroy {
	return &dropInvalidDestroy{
		tag:         tag,
		allocations: nil,
	}
}

func (dropTransform *dropInvalidDestroy) RequiresAccurateState() bool {
	return false
}

func (dropTransform *dropInvalidDestroy) RequiresInnerStateMutation() bool {
	return false
}

func (dropTransform *dropInvalidDestroy) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (dropTransform *dropInvalidDestroy) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	dropTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (dropTransform *dropInvalidDestroy) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (dropTransform *dropInvalidDestroy) ClearTransformResources(ctx context.Context) {
	dropTransform.allocations.FreeAllocations()
}

func (dropTransform *dropInvalidDestroy) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	outputCmds := make([]api.Cmd, 0)

	for i, cmd := range inputCommands {
		if newCmd := dropTransform.dropOrModifyCommand(ctx, inputState, id.GetID(), i, cmd); newCmd != nil {
			outputCmds = append(outputCmds, newCmd)
		}
	}

	return outputCmds, nil
}

func (dropTransform *dropInvalidDestroy) dropOrModifyCommand(ctx context.Context, inputState *api.GlobalState, id api.CmdID, index int, cmd api.Cmd) api.Cmd {
	vulkanState := GetState(inputState)

	switch cmd := cmd.(type) {
	case *VkDestroyInstance:
		if !vulkanState.Instances().Contains(cmd.Instance()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Instance())
			return nil
		}
	case *VkDestroyDevice:
		if !vulkanState.Devices().Contains(cmd.Device()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Device())
			return nil
		}
	case *VkFreeMemory:
		if !vulkanState.DeviceMemories().Contains(cmd.Memory()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Memory())
			return nil
		}
	case *VkDestroyBuffer:
		if !vulkanState.Buffers().Contains(cmd.Buffer()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Buffer())
			return nil
		}
	case *VkDestroyBufferView:
		if !vulkanState.BufferViews().Contains(cmd.BufferView()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.BufferView())
			return nil
		}
	case *VkDestroyImage:
		if !vulkanState.Images().Contains(cmd.Image()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Image())
			return nil
		}
	case *VkDestroyImageView:
		if !vulkanState.ImageViews().Contains(cmd.ImageView()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.ImageView())
			return nil
		}
	case *VkDestroyShaderModule:
		if !vulkanState.ShaderModules().Contains(cmd.ShaderModule()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.ShaderModule())
			return nil
		}
	case *VkDestroyPipeline:
		if !vulkanState.GraphicsPipelines().Contains(cmd.Pipeline()) &&
			!vulkanState.ComputePipelines().Contains(cmd.Pipeline()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Pipeline())
			return nil
		}
	case *VkDestroyPipelineLayout:
		if !vulkanState.PipelineLayouts().Contains(cmd.PipelineLayout()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.PipelineLayout())
			return nil
		}
	case *VkDestroyPipelineCache:
		if !vulkanState.PipelineCaches().Contains(cmd.PipelineCache()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.PipelineCache())
			return nil
		}
	case *VkDestroySampler:
		if !vulkanState.Samplers().Contains(cmd.Sampler()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Sampler())
			return nil
		}
	case *VkDestroyDescriptorSetLayout:
		if !vulkanState.DescriptorSetLayouts().Contains(cmd.DescriptorSetLayout()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.DescriptorSetLayout())
			return nil
		}
	case *VkDestroyDescriptorPool:
		if !vulkanState.DescriptorPools().Contains(cmd.DescriptorPool()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.DescriptorPool())
			return nil
		}
	case *VkDestroyFence:
		if !vulkanState.Fences().Contains(cmd.Fence()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Fence())
			return nil
		}
	case *VkDestroySemaphore:
		if !vulkanState.Semaphores().Contains(cmd.Semaphore()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Semaphore())
			return nil
		}
	case *VkDestroyEvent:
		if !vulkanState.Events().Contains(cmd.Event()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Event())
			return nil
		}
	case *VkDestroyQueryPool:
		if !vulkanState.QueryPools().Contains(cmd.QueryPool()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.QueryPool())
			return nil
		}
	case *VkDestroyFramebuffer:
		if !vulkanState.Framebuffers().Contains(cmd.Framebuffer()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Framebuffer())
			return nil
		}
	case *VkDestroyRenderPass:
		if !vulkanState.RenderPasses().Contains(cmd.RenderPass()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.RenderPass())
			return nil
		}
	case *VkDestroyCommandPool:
		if !vulkanState.CommandPools().Contains(cmd.CommandPool()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.CommandPool())
			return nil
		}
	case *VkDestroySurfaceKHR:
		if !vulkanState.Surfaces().Contains(cmd.Surface()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Surface())
			return nil
		}
	case *VkDestroySwapchainKHR:
		if !vulkanState.Swapchains().Contains(cmd.Swapchain()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Swapchain())
			return nil
		}
	case *VkDestroyDebugReportCallbackEXT:
		if !vulkanState.DebugReportCallbacks().Contains(cmd.Callback()) {
			dropTransform.warnDropCmd(ctx, id, index, cmd, cmd.Callback())
			return nil
		}
	case *VkFreeDescriptorSets:
		return dropTransform.dropOrModifyFreeDescriptorSets(ctx, inputState, id, index, cmd)
	case *VkFreeCommandBuffers:
		return dropTransform.dropOrModifyFreeCommandBuffers(ctx, inputState, id, index, cmd)
	}

	return cmd
}

func (dropTransform *dropInvalidDestroy) dropOrModifyFreeDescriptorSets(ctx context.Context, inputState *api.GlobalState, id api.CmdID, index int, cmd *VkFreeDescriptorSets) api.Cmd {
	descSetCount := cmd.DescriptorSetCount()
	if descSetCount == 0 {
		return cmd
	}

	layout := inputState.MemoryLayout
	cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
	descSets := cmd.PDescriptorSets().Slice(0, uint64(descSetCount), layout).MustRead(ctx, cmd, inputState, nil)
	newDescSets := []VkDescriptorSet{}
	dropped := []VkDescriptorSet{}
	for _, ds := range descSets {
		if GetState(inputState).DescriptorSets().Contains(ds) {
			newDescSets = append(newDescSets, ds)
		} else {
			dropped = append(dropped, ds)
		}
	}

	if len(newDescSets) == len(descSets) {
		// No need to modify the command
		return cmd
	}

	if len(newDescSets) == 0 {
		// no need to have this command
		dropTransform.warnDropCmd(ctx, id, index, cmd, descSets)
		return nil
	}

	// need to modify the command to drop the command buffers not
	// in the state out of the command
	dropTransform.warnModifyCmd(ctx, id, index, cmd, dropped)

	newDescSetsData := dropTransform.allocations.AllocDataOrPanic(ctx, newDescSets)

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkFreeDescriptorSets(
		cmd.Device(), cmd.DescriptorPool(), uint32(len(newDescSets)),
		newDescSetsData.Ptr(), VkResult_VK_SUCCESS).AddRead(newDescSetsData.Data())
	return newCmd
}

func (dropTransform *dropInvalidDestroy) dropOrModifyFreeCommandBuffers(ctx context.Context, inputState *api.GlobalState, id api.CmdID, index int, cmd *VkFreeCommandBuffers) api.Cmd {
	cmdBufCount := cmd.CommandBufferCount()
	if cmdBufCount == 0 {
		return cmd
	}

	layout := inputState.MemoryLayout
	cmd.Extras().Observations().ApplyWrites(inputState.Memory.ApplicationPool())
	cmdBufs := cmd.PCommandBuffers().Slice(0, uint64(cmdBufCount), layout).MustRead(ctx, cmd, inputState, nil)
	newCmdBufs := []VkCommandBuffer{}
	dropped := []VkCommandBuffer{}
	for _, commandBuffer := range cmdBufs {
		if GetState(inputState).CommandBuffers().Contains(commandBuffer) {
			newCmdBufs = append(newCmdBufs, commandBuffer)
		} else {
			dropped = append(dropped, commandBuffer)
		}
	}

	if len(newCmdBufs) == len(cmdBufs) {
		// no need to modify this command
		return cmd
	}

	if len(newCmdBufs) == 0 {
		// no need to have this command
		dropTransform.warnDropCmd(ctx, id, index, cmd, cmdBufs)
		return nil
	}

	// need to modify the command to drop the command buffers not
	// in the state out of the command
	dropTransform.warnModifyCmd(ctx, id, index, cmd, dropped)

	newCmdBufsData := dropTransform.allocations.AllocDataOrPanic(ctx, newCmdBufs)

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkFreeCommandBuffers(cmd.Device(), cmd.CommandPool(), uint32(len(newCmdBufs)), newCmdBufsData.Ptr()).AddRead(newCmdBufsData.Data())

	return newCmd
}

func (dropTransform *dropInvalidDestroy) warnDropCmd(ctx context.Context, id api.CmdID, index int, cmd api.Cmd, handles ...interface{}) {
	log.W(ctx, "[%v] Dropping [%d:%v]:%v because the creation of %v was not recorded", dropTransform.tag, id, index, cmd, handles)
}

func (dropTransform *dropInvalidDestroy) warnModifyCmd(ctx context.Context, id api.CmdID, index int, cmd api.Cmd, handles ...interface{}) {
	log.W(ctx, "[%v] Modifing [%d:%v]:%v to remove the reference to %v because the creation of them were not recorded", dropTransform.tag, id, index, cmd, handles)
}
