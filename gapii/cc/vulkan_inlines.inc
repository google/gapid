/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This file is intended to be included by vulkan_spy.h inside
// of the gapid namespace.

inline void VulkanSpy::notifyPendingCommandAdded(CallObserver*, VkQueue) {}

inline void VulkanSpy::vkErrInvalidHandle(CallObserver*, std::string handleType, uint64_t handle){
    GAPID_WARNING("Error: Invalid %s: %" PRIu64, handleType.c_str(), handle)
}

inline void VulkanSpy::vkErrNullPointer(CallObserver*, std::string pointerType) {
    GAPID_WARNING("Error: Null pointer of %s", pointerType.c_str())
}

inline void VulkanSpy::vkErrNotNullPointer(CallObserver*, std::string pointerType) {
    GAPID_WARNING("Error: Not Null Pointer: %s", pointerType.c_str());
}

inline void VulkanSpy::vkErrUnrecognizedExtension(CallObserver*, std::string name) {
    GAPID_WARNING("Error: Unrecognized extension: %s", name.c_str())
}

inline void VulkanSpy::vkErrExpectNVDedicatedlyAllocatedHandle(CallObserver*, std::string handleType, uint64_t handle) {
    GAPID_WARNING("Error: Expected handle that was allocated with a dedicated allocation: %s: %" PRIu64, handleType.c_str(), handle)
}

inline void VulkanSpy::vkErrInvalidDescriptorArrayElement(CallObserver*, uint64_t set, uint32_t binding, uint32_t array_index) {
  GAPID_WARNING("Error: Invalid descriptor array element specified by descriptor set: %" PRIu64 ", binding: %" PRIu32 ", array index: %" PRIu32, set, binding, array_index);
}

inline void VulkanSpy::vkErrCommandBufferIncomplete(CallObserver*, VkCommandBuffer cmdbuf) {
    GAPID_WARNING("Error: Executing command buffer %zu was not in the COMPLETED state", cmdbuf)
}

inline void VulkanSpy::vkErrCommandBufferNotRecording(CallObserver*, VkCommandBuffer cmdbuf) {
    GAPID_WARNING("Error: Executing command buffer %zu was not in the RECORDING state", cmdbuf)
}

inline void VulkanSpy::vkErrInvalidImageLayout(CallObserver*, VkImage img, uint32_t aspect, uint32_t layer, uint32_t level, uint32_t layout, uint32_t expectedLayout) {
    GAPID_WARNING("Error: Image subresource at Image: %" PRIu64 ", AspectBit: %" PRIu32 ", Layer: %" PRIu32 ", Level: %" PRIu32 " was in layout %" PRIu32 ", but was expected to be in layout %" PRIu32,
        img, aspect, layer, level, layout, expectedLayout);
}

inline void VulkanSpy::vkErrInvalidImageSubresource(CallObserver*, VkImage img, std::string subresourceType, uint32_t value) {
    GAPID_WARNING("Error: Accessing invalid image subresource at Image: %" PRIu64 ", %s: %" PRIu32, img, subresourceType.c_str(), value);
}

inline void VulkanSpy::vkErrInvalidDescriptorBindingType(CallObserver*, VkDescriptorSet set, uint32_t binding, uint32_t layout_type, uint32_t update_type) {
    GAPID_WARNING("Error: Updating descriptor binding at: %" PRIu64 ": %" PRIu32 " with type: %" PRIu32 ", but the type defined in descriptor set layout is: %" PRIu32 "", set, binding, layout_type, update_type);
}
