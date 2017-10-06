////////////////////////////////////////////////////////////////////////////////
// Automatically generated file. Do not modify!
////////////////////////////////////////////////////////////////////////////////

package vulkan

import (
	"context"
	"unsafe"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api/vulkan/vulkan_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// Just in case it is not used
var _ memory.PoolID
var _ memory_pb.Pointer
var _ unsafe.Pointer

func init() {
	protoconv.Register(
	func(ctx context.Context, ref_cb func(interface{}) uint64, in *State) (*vulkan_pb.InitialState, error) { return in.ToProto(ref_cb), nil },
	func(ctx context.Context, ref_cb func(uint64, interface{}), in *vulkan_pb.InitialState) (*State, error) { v := StateFrom(in, ref_cb); return &v, nil },
)
}

// ToProto returns the storage form of the State.
func (ϟa *State) ToProto(ref_cb func(interface{}) uint64) *vulkan_pb.InitialState {
to := &vulkan_pb.InitialState{}
to.Instances = make([]*vulkan_pb.InitialState_InstancesEntry, 0, len(ϟa.Instances))
for ϟk, ϟv := range ϟa.Instances {
	to.Instances = append(to.Instances, &vulkan_pb.InitialState_InstancesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.PhysicalDevices = make([]*vulkan_pb.InitialState_PhysicalDevicesEntry, 0, len(ϟa.PhysicalDevices))
for ϟk, ϟv := range ϟa.PhysicalDevices {
	to.PhysicalDevices = append(to.PhysicalDevices, &vulkan_pb.InitialState_PhysicalDevicesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Devices = make([]*vulkan_pb.InitialState_DevicesEntry, 0, len(ϟa.Devices))
for ϟk, ϟv := range ϟa.Devices {
	to.Devices = append(to.Devices, &vulkan_pb.InitialState_DevicesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Queues = make([]*vulkan_pb.InitialState_QueuesEntry, 0, len(ϟa.Queues))
for ϟk, ϟv := range ϟa.Queues {
	to.Queues = append(to.Queues, &vulkan_pb.InitialState_QueuesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.CommandBuffers = make([]*vulkan_pb.InitialState_CommandBuffersEntry, 0, len(ϟa.CommandBuffers))
for ϟk, ϟv := range ϟa.CommandBuffers {
	to.CommandBuffers = append(to.CommandBuffers, &vulkan_pb.InitialState_CommandBuffersEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.DeviceMemories = make([]*vulkan_pb.InitialState_DeviceMemoriesEntry, 0, len(ϟa.DeviceMemories))
for ϟk, ϟv := range ϟa.DeviceMemories {
	to.DeviceMemories = append(to.DeviceMemories, &vulkan_pb.InitialState_DeviceMemoriesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Buffers = make([]*vulkan_pb.InitialState_BuffersEntry, 0, len(ϟa.Buffers))
for ϟk, ϟv := range ϟa.Buffers {
	to.Buffers = append(to.Buffers, &vulkan_pb.InitialState_BuffersEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.BufferViews = make([]*vulkan_pb.InitialState_BufferViewsEntry, 0, len(ϟa.BufferViews))
for ϟk, ϟv := range ϟa.BufferViews {
	to.BufferViews = append(to.BufferViews, &vulkan_pb.InitialState_BufferViewsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Images = make([]*vulkan_pb.InitialState_ImagesEntry, 0, len(ϟa.Images))
for ϟk, ϟv := range ϟa.Images {
	to.Images = append(to.Images, &vulkan_pb.InitialState_ImagesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.ImageViews = make([]*vulkan_pb.InitialState_ImageViewsEntry, 0, len(ϟa.ImageViews))
for ϟk, ϟv := range ϟa.ImageViews {
	to.ImageViews = append(to.ImageViews, &vulkan_pb.InitialState_ImageViewsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.ShaderModules = make([]*vulkan_pb.InitialState_ShaderModulesEntry, 0, len(ϟa.ShaderModules))
for ϟk, ϟv := range ϟa.ShaderModules {
	to.ShaderModules = append(to.ShaderModules, &vulkan_pb.InitialState_ShaderModulesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.GraphicsPipelines = make([]*vulkan_pb.InitialState_GraphicsPipelinesEntry, 0, len(ϟa.GraphicsPipelines))
for ϟk, ϟv := range ϟa.GraphicsPipelines {
	to.GraphicsPipelines = append(to.GraphicsPipelines, &vulkan_pb.InitialState_GraphicsPipelinesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.ComputePipelines = make([]*vulkan_pb.InitialState_ComputePipelinesEntry, 0, len(ϟa.ComputePipelines))
for ϟk, ϟv := range ϟa.ComputePipelines {
	to.ComputePipelines = append(to.ComputePipelines, &vulkan_pb.InitialState_ComputePipelinesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.PipelineLayouts = make([]*vulkan_pb.InitialState_PipelineLayoutsEntry, 0, len(ϟa.PipelineLayouts))
for ϟk, ϟv := range ϟa.PipelineLayouts {
	to.PipelineLayouts = append(to.PipelineLayouts, &vulkan_pb.InitialState_PipelineLayoutsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Samplers = make([]*vulkan_pb.InitialState_SamplersEntry, 0, len(ϟa.Samplers))
for ϟk, ϟv := range ϟa.Samplers {
	to.Samplers = append(to.Samplers, &vulkan_pb.InitialState_SamplersEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.DescriptorSets = make([]*vulkan_pb.InitialState_DescriptorSetsEntry, 0, len(ϟa.DescriptorSets))
for ϟk, ϟv := range ϟa.DescriptorSets {
	to.DescriptorSets = append(to.DescriptorSets, &vulkan_pb.InitialState_DescriptorSetsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.DescriptorSetLayouts = make([]*vulkan_pb.InitialState_DescriptorSetLayoutsEntry, 0, len(ϟa.DescriptorSetLayouts))
for ϟk, ϟv := range ϟa.DescriptorSetLayouts {
	to.DescriptorSetLayouts = append(to.DescriptorSetLayouts, &vulkan_pb.InitialState_DescriptorSetLayoutsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.DescriptorPools = make([]*vulkan_pb.InitialState_DescriptorPoolsEntry, 0, len(ϟa.DescriptorPools))
for ϟk, ϟv := range ϟa.DescriptorPools {
	to.DescriptorPools = append(to.DescriptorPools, &vulkan_pb.InitialState_DescriptorPoolsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Fences = make([]*vulkan_pb.InitialState_FencesEntry, 0, len(ϟa.Fences))
for ϟk, ϟv := range ϟa.Fences {
	to.Fences = append(to.Fences, &vulkan_pb.InitialState_FencesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Semaphores = make([]*vulkan_pb.InitialState_SemaphoresEntry, 0, len(ϟa.Semaphores))
for ϟk, ϟv := range ϟa.Semaphores {
	to.Semaphores = append(to.Semaphores, &vulkan_pb.InitialState_SemaphoresEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Events = make([]*vulkan_pb.InitialState_EventsEntry, 0, len(ϟa.Events))
for ϟk, ϟv := range ϟa.Events {
	to.Events = append(to.Events, &vulkan_pb.InitialState_EventsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.QueryPools = make([]*vulkan_pb.InitialState_QueryPoolsEntry, 0, len(ϟa.QueryPools))
for ϟk, ϟv := range ϟa.QueryPools {
	to.QueryPools = append(to.QueryPools, &vulkan_pb.InitialState_QueryPoolsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Framebuffers = make([]*vulkan_pb.InitialState_FramebuffersEntry, 0, len(ϟa.Framebuffers))
for ϟk, ϟv := range ϟa.Framebuffers {
	to.Framebuffers = append(to.Framebuffers, &vulkan_pb.InitialState_FramebuffersEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.RenderPasses = make([]*vulkan_pb.InitialState_RenderPassesEntry, 0, len(ϟa.RenderPasses))
for ϟk, ϟv := range ϟa.RenderPasses {
	to.RenderPasses = append(to.RenderPasses, &vulkan_pb.InitialState_RenderPassesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.PipelineCaches = make([]*vulkan_pb.InitialState_PipelineCachesEntry, 0, len(ϟa.PipelineCaches))
for ϟk, ϟv := range ϟa.PipelineCaches {
	to.PipelineCaches = append(to.PipelineCaches, &vulkan_pb.InitialState_PipelineCachesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.CommandPools = make([]*vulkan_pb.InitialState_CommandPoolsEntry, 0, len(ϟa.CommandPools))
for ϟk, ϟv := range ϟa.CommandPools {
	to.CommandPools = append(to.CommandPools, &vulkan_pb.InitialState_CommandPoolsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Surfaces = make([]*vulkan_pb.InitialState_SurfacesEntry, 0, len(ϟa.Surfaces))
for ϟk, ϟv := range ϟa.Surfaces {
	to.Surfaces = append(to.Surfaces, &vulkan_pb.InitialState_SurfacesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.Swapchains = make([]*vulkan_pb.InitialState_SwapchainsEntry, 0, len(ϟa.Swapchains))
for ϟk, ϟv := range ϟa.Swapchains {
	to.Swapchains = append(to.Swapchains, &vulkan_pb.InitialState_SwapchainsEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.DisplayModes = make([]*vulkan_pb.InitialState_DisplayModesEntry, 0, len(ϟa.DisplayModes))
for ϟk, ϟv := range ϟa.DisplayModes {
	to.DisplayModes = append(to.DisplayModes, &vulkan_pb.InitialState_DisplayModesEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.LastBoundQueue = &memory_pb.Reference{ref_cb(ϟa.LastBoundQueue)}
to.CurrentComputePipeline = &memory_pb.Reference{ref_cb(ϟa.CurrentComputePipeline)}
to.LastDrawInfos = make([]*vulkan_pb.InitialState_LastDrawInfosEntry, 0, len(ϟa.LastDrawInfos))
for ϟk, ϟv := range ϟa.LastDrawInfos {
	to.LastDrawInfos = append(to.LastDrawInfos, &vulkan_pb.InitialState_LastDrawInfosEntry{Key:  (uint64)(ϟk),Value: &memory_pb.Reference{ref_cb(ϟv)}})
}
to.LastPresentInfo = ϟa.LastPresentInfo.ToProto(ref_cb)
to.LastSubmission = (uint32)(ϟa.LastSubmission)
return to
}

// StateFrom builds a State from the storage form.
func StateFrom(from *vulkan_pb.InitialState,ref_cb func(uint64, interface{})) State {
ϟa := State{}
ϟa.Instances = make(VkInstanceːInstanceObjectʳᵐ, len(from.Instances))
for _, ϟe := range from.Instances {
	ϟk := (VkInstance)(ϟe.Key)
	var ϟv = (*InstanceObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Instances, ϟk})
	ϟa.Instances[ϟk] = ϟv
}
ϟa.PhysicalDevices = make(VkPhysicalDeviceːPhysicalDeviceObjectʳᵐ, len(from.PhysicalDevices))
for _, ϟe := range from.PhysicalDevices {
	ϟk := (VkPhysicalDevice)(ϟe.Key)
	var ϟv = (*PhysicalDeviceObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.PhysicalDevices, ϟk})
	ϟa.PhysicalDevices[ϟk] = ϟv
}
ϟa.Devices = make(VkDeviceːDeviceObjectʳᵐ, len(from.Devices))
for _, ϟe := range from.Devices {
	ϟk := (VkDevice)(ϟe.Key)
	var ϟv = (*DeviceObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Devices, ϟk})
	ϟa.Devices[ϟk] = ϟv
}
ϟa.Queues = make(VkQueueːQueueObjectʳᵐ, len(from.Queues))
for _, ϟe := range from.Queues {
	ϟk := (VkQueue)(ϟe.Key)
	var ϟv = (*QueueObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Queues, ϟk})
	ϟa.Queues[ϟk] = ϟv
}
ϟa.CommandBuffers = make(VkCommandBufferːCommandBufferObjectʳᵐ, len(from.CommandBuffers))
for _, ϟe := range from.CommandBuffers {
	ϟk := (VkCommandBuffer)(ϟe.Key)
	var ϟv = (*CommandBufferObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.CommandBuffers, ϟk})
	ϟa.CommandBuffers[ϟk] = ϟv
}
ϟa.DeviceMemories = make(VkDeviceMemoryːDeviceMemoryObjectʳᵐ, len(from.DeviceMemories))
for _, ϟe := range from.DeviceMemories {
	ϟk := (VkDeviceMemory)(ϟe.Key)
	var ϟv = (*DeviceMemoryObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.DeviceMemories, ϟk})
	ϟa.DeviceMemories[ϟk] = ϟv
}
ϟa.Buffers = make(VkBufferːBufferObjectʳᵐ, len(from.Buffers))
for _, ϟe := range from.Buffers {
	ϟk := (VkBuffer)(ϟe.Key)
	var ϟv = (*BufferObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Buffers, ϟk})
	ϟa.Buffers[ϟk] = ϟv
}
ϟa.BufferViews = make(VkBufferViewːBufferViewObjectʳᵐ, len(from.BufferViews))
for _, ϟe := range from.BufferViews {
	ϟk := (VkBufferView)(ϟe.Key)
	var ϟv = (*BufferViewObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.BufferViews, ϟk})
	ϟa.BufferViews[ϟk] = ϟv
}
ϟa.Images = make(VkImageːImageObjectʳᵐ, len(from.Images))
for _, ϟe := range from.Images {
	ϟk := (VkImage)(ϟe.Key)
	var ϟv = (*ImageObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Images, ϟk})
	ϟa.Images[ϟk] = ϟv
}
ϟa.ImageViews = make(VkImageViewːImageViewObjectʳᵐ, len(from.ImageViews))
for _, ϟe := range from.ImageViews {
	ϟk := (VkImageView)(ϟe.Key)
	var ϟv = (*ImageViewObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.ImageViews, ϟk})
	ϟa.ImageViews[ϟk] = ϟv
}
ϟa.ShaderModules = make(VkShaderModuleːShaderModuleObjectʳᵐ, len(from.ShaderModules))
for _, ϟe := range from.ShaderModules {
	ϟk := (VkShaderModule)(ϟe.Key)
	var ϟv = (*ShaderModuleObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.ShaderModules, ϟk})
	ϟa.ShaderModules[ϟk] = ϟv
}
ϟa.GraphicsPipelines = make(VkPipelineːGraphicsPipelineObjectʳᵐ, len(from.GraphicsPipelines))
for _, ϟe := range from.GraphicsPipelines {
	ϟk := (VkPipeline)(ϟe.Key)
	var ϟv = (*GraphicsPipelineObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.GraphicsPipelines, ϟk})
	ϟa.GraphicsPipelines[ϟk] = ϟv
}
ϟa.ComputePipelines = make(VkPipelineːComputePipelineObjectʳᵐ, len(from.ComputePipelines))
for _, ϟe := range from.ComputePipelines {
	ϟk := (VkPipeline)(ϟe.Key)
	var ϟv = (*ComputePipelineObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.ComputePipelines, ϟk})
	ϟa.ComputePipelines[ϟk] = ϟv
}
ϟa.PipelineLayouts = make(VkPipelineLayoutːPipelineLayoutObjectʳᵐ, len(from.PipelineLayouts))
for _, ϟe := range from.PipelineLayouts {
	ϟk := (VkPipelineLayout)(ϟe.Key)
	var ϟv = (*PipelineLayoutObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.PipelineLayouts, ϟk})
	ϟa.PipelineLayouts[ϟk] = ϟv
}
ϟa.Samplers = make(VkSamplerːSamplerObjectʳᵐ, len(from.Samplers))
for _, ϟe := range from.Samplers {
	ϟk := (VkSampler)(ϟe.Key)
	var ϟv = (*SamplerObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Samplers, ϟk})
	ϟa.Samplers[ϟk] = ϟv
}
ϟa.DescriptorSets = make(VkDescriptorSetːDescriptorSetObjectʳᵐ, len(from.DescriptorSets))
for _, ϟe := range from.DescriptorSets {
	ϟk := (VkDescriptorSet)(ϟe.Key)
	var ϟv = (*DescriptorSetObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.DescriptorSets, ϟk})
	ϟa.DescriptorSets[ϟk] = ϟv
}
ϟa.DescriptorSetLayouts = make(VkDescriptorSetLayoutːDescriptorSetLayoutObjectʳᵐ, len(from.DescriptorSetLayouts))
for _, ϟe := range from.DescriptorSetLayouts {
	ϟk := (VkDescriptorSetLayout)(ϟe.Key)
	var ϟv = (*DescriptorSetLayoutObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.DescriptorSetLayouts, ϟk})
	ϟa.DescriptorSetLayouts[ϟk] = ϟv
}
ϟa.DescriptorPools = make(VkDescriptorPoolːDescriptorPoolObjectʳᵐ, len(from.DescriptorPools))
for _, ϟe := range from.DescriptorPools {
	ϟk := (VkDescriptorPool)(ϟe.Key)
	var ϟv = (*DescriptorPoolObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.DescriptorPools, ϟk})
	ϟa.DescriptorPools[ϟk] = ϟv
}
ϟa.Fences = make(VkFenceːFenceObjectʳᵐ, len(from.Fences))
for _, ϟe := range from.Fences {
	ϟk := (VkFence)(ϟe.Key)
	var ϟv = (*FenceObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Fences, ϟk})
	ϟa.Fences[ϟk] = ϟv
}
ϟa.Semaphores = make(VkSemaphoreːSemaphoreObjectʳᵐ, len(from.Semaphores))
for _, ϟe := range from.Semaphores {
	ϟk := (VkSemaphore)(ϟe.Key)
	var ϟv = (*SemaphoreObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Semaphores, ϟk})
	ϟa.Semaphores[ϟk] = ϟv
}
ϟa.Events = make(VkEventːEventObjectʳᵐ, len(from.Events))
for _, ϟe := range from.Events {
	ϟk := (VkEvent)(ϟe.Key)
	var ϟv = (*EventObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Events, ϟk})
	ϟa.Events[ϟk] = ϟv
}
ϟa.QueryPools = make(VkQueryPoolːQueryPoolObjectʳᵐ, len(from.QueryPools))
for _, ϟe := range from.QueryPools {
	ϟk := (VkQueryPool)(ϟe.Key)
	var ϟv = (*QueryPoolObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.QueryPools, ϟk})
	ϟa.QueryPools[ϟk] = ϟv
}
ϟa.Framebuffers = make(VkFramebufferːFramebufferObjectʳᵐ, len(from.Framebuffers))
for _, ϟe := range from.Framebuffers {
	ϟk := (VkFramebuffer)(ϟe.Key)
	var ϟv = (*FramebufferObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Framebuffers, ϟk})
	ϟa.Framebuffers[ϟk] = ϟv
}
ϟa.RenderPasses = make(VkRenderPassːRenderPassObjectʳᵐ, len(from.RenderPasses))
for _, ϟe := range from.RenderPasses {
	ϟk := (VkRenderPass)(ϟe.Key)
	var ϟv = (*RenderPassObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.RenderPasses, ϟk})
	ϟa.RenderPasses[ϟk] = ϟv
}
ϟa.PipelineCaches = make(VkPipelineCacheːPipelineCacheObjectʳᵐ, len(from.PipelineCaches))
for _, ϟe := range from.PipelineCaches {
	ϟk := (VkPipelineCache)(ϟe.Key)
	var ϟv = (*PipelineCacheObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.PipelineCaches, ϟk})
	ϟa.PipelineCaches[ϟk] = ϟv
}
ϟa.CommandPools = make(VkCommandPoolːCommandPoolObjectʳᵐ, len(from.CommandPools))
for _, ϟe := range from.CommandPools {
	ϟk := (VkCommandPool)(ϟe.Key)
	var ϟv = (*CommandPoolObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.CommandPools, ϟk})
	ϟa.CommandPools[ϟk] = ϟv
}
ϟa.Surfaces = make(VkSurfaceKHRːSurfaceObjectʳᵐ, len(from.Surfaces))
for _, ϟe := range from.Surfaces {
	ϟk := (VkSurfaceKHR)(ϟe.Key)
	var ϟv = (*SurfaceObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Surfaces, ϟk})
	ϟa.Surfaces[ϟk] = ϟv
}
ϟa.Swapchains = make(VkSwapchainKHRːSwapchainObjectʳᵐ, len(from.Swapchains))
for _, ϟe := range from.Swapchains {
	ϟk := (VkSwapchainKHR)(ϟe.Key)
	var ϟv = (*SwapchainObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.Swapchains, ϟk})
	ϟa.Swapchains[ϟk] = ϟv
}
ϟa.DisplayModes = make(VkDisplayModeKHRːDisplayModeObjectʳᵐ, len(from.DisplayModes))
for _, ϟe := range from.DisplayModes {
	ϟk := (VkDisplayModeKHR)(ϟe.Key)
	var ϟv = (*DisplayModeObject)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.DisplayModes, ϟk})
	ϟa.DisplayModes[ϟk] = ϟv
}
ref_cb(from.LastBoundQueue.Identifier, &ϟa.LastBoundQueue)
ref_cb(from.CurrentComputePipeline.Identifier, &ϟa.CurrentComputePipeline)
ϟa.LastDrawInfos = make(VkQueueːDrawInfoʳᵐ, len(from.LastDrawInfos))
for _, ϟe := range from.LastDrawInfos {
	ϟk := (VkQueue)(ϟe.Key)
	var ϟv = (*DrawInfo)(nil)
	ref_cb(ϟe.Value.Identifier, memory.MapReferenceValue{ϟa.LastDrawInfos, ϟk})
	ϟa.LastDrawInfos[ϟk] = ϟv
}
ϟa.LastPresentInfo = PresentInfoFrom(from.LastPresentInfo, ref_cb)
ϟa.LastSubmission = (LastSubmissionType)(from.LastSubmission)
return ϟa
}
