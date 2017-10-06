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

// Note this file is included inside vulkan_gfx_api.h:
//
// namespace gapir {
//
// class Vulkan : public Api {
// public:

typedef std::unordered_map<VkPhysicalDevice, VkInstance> VkPhysicalDeviceToVkInstance;
typedef std::unordered_map<VkDevice, VkPhysicalDevice> VkDeviceToVkPhysicalDevice;
typedef std::unordered_map<VkQueue, VkDevice> VkQueueToVkDevice;
typedef std::unordered_map<VkCommandBuffer, VkDevice> VkCommandBufferToVkDevice;
typedef struct {
    VkPhysicalDeviceToVkInstance VkPhysicalDevicesToVkInstances;
    VkDeviceToVkPhysicalDevice VkDevicesToVkPhysicalDevices;
    VkQueueToVkDevice VkQueuesToVkDevices;
    VkCommandBufferToVkDevice VkCommandBuffersToVkDevices;
} IndirectMaps;

IndirectMaps mIndirectMaps;

// Function for wrapping around the normal vkCreateInstance to inject
// virtual swapchain as an additional enabled layer.
bool replayCreateVkInstance(Stack* stack, bool pushReturn);

// Function for wrapping around the normal vkCreateDevice to null the
// pNext field in VkDeviceCreateInfo.
bool replayCreateVkDevice(Stack* stack, bool pushReturn);

// Builtin function for registering instance-level function pointers and
// binding all physical devices associated with the given instance.
// The instance is popped from the top of the stack.
bool replayRegisterVkInstance(Stack* stack);

// Builtin function for destroying instance-level function pointers.
// The instance is popped from the top of the stack.
bool replayUnregisterVkInstance(Stack* stack);

// Builtin function for creating device-level function pointers.
// From the top of the stack, pop three arguments sequentially:
// - pointer to the VkDeviceCreateInfo struct for this device,
// - the device,
// - the physical device.
bool replayRegisterVkDevice(Stack* stack);

// Builtin function for destroying device-level function pointers.
// The device is popped from the top of the stack.
bool replayUnregisterVkDevice(Stack* stack);

// Builtin function for linking command buffers to their device.
// From the top of the stack, pop three arguments sequentially:
// - ponter to a sequence of command buffers,
// - number of command buffers,
// - the device.
bool replayRegisterVkCommandBuffers(Stack* stack);

// Builtin function for discarding linking of command buffers.
// From the top of the stack, pop two arguments sequentially:
// - ponter to a sequence of command buffers,
// - number of command buffers.
bool replayUnregisterVkCommandBuffers(Stack* stack);

// Builtin function for setting the virtual swapchain to
// always returns the requsted swapchain imge.
bool toggleVirtualSwapchainReturnAcquiredImage(Stack* stack);

// Builtin function for replaying vkGetFenceStatus. If the return of
// vkGetFenceStatus is VK_SUCCESS, this function makes sure the replay will not
// proceed until VK_SUCCESS is returned from vkGetFenceStatus in the replay
// side.
bool replayGetFenceStatus(Stack* stack, bool pushReturn);

// Builtin function for replaying vkGetEventStatus.  The traced return of
// vkGetEventStatus can be used to block this function if and only if the
// traced return matches with the global state mutation result.  For example:
// Call vkQueueSubmit a queue with vkCmdSetEvent in the command buffer first,
// then call vkGetEventStatus. In the trace, the return of vkGetEventStatus
// might be 'unsignaled', but after the mutation of the state, the record in
// the global state should be 'signaled'. In such a case, waiting for the
// vkGetEventStatus returns 'unsignaled' on the replay may cause an infinite
// long waiting.
bool replayGetEventStatus(Stack* stack, bool pushReturn);

// Builtin function for getting image memory requirement and allocating
// corresponding memory for a image on the replay side.
bool replayAllocateImageMemory(Stack* stack, bool pushReturn);

// Builtin function for recreating physical devices. The reason we have
// to customize this is that the device can choose to return the
// physical devices in any order.
bool replayEnumeratePhysicalDevices(Stack* stack, bool pushReturn);
