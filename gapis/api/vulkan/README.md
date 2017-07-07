# Vulkan Status

## Trace and Replay
Vulkan is currently WIP. Many samples, and applications do work correctly,
but not all.

## Mid-Execution Capture
Mid-Execution capture is currently in progress. Currently there is no way
exposed to start a capture at any frame other than 0, but this will
be exposed once the functionality is at parity with non mid-execution capture.

## Subcommands
When replaying to a specific command within a command buffer, we have to
re-write the command-buffer. The Subcommands list shows all commands in
command-buffers that support re-writing.


## Current Support
The current status of support for the Vulkan API on a method by method basis
are as follows.

| Command Name                                          | Capture | Mid-Execution |  Subcommands  |
|-------------------------------------------------------|:-------:|:-------------:|:-------------:|
| vkAllocateCommandBuffers                              |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateDevice                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateInstance                                      |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyDevice                                       |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyInstance                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkEnumerateDeviceExtensionProperties                  |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkEnumerateDeviceLayerProperties                      |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkEnumerateInstanceExtensionProperties                |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkEnumerateInstanceLayerProperties                    |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkEnumeratePhysicalDevices                            |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkFreeCommandBuffers                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkGetDeviceProcAddr                                   |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetDeviceQueue                                      |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkGetInstanceProcAddr                                 |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSparseImageFormatProperties        |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceFeatures                           |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceFormatProperties                   |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceImageFormatProperties              |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceMemoryProperties                   |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceProperties                         |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceQueueFamilyProperties              |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkAcquireNextImageKHR                                 |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkAllocateDescriptorSets                              |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkAllocateMemory                                      |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkBeginCommandBuffer                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkBindBufferMemory                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkBindImageMemory                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCmdBeginQuery                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBeginRenderPass                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBindDescriptorSets                               |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBindIndexBuffer                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBindPipeline                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBindVertexBuffers                                |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdBlitImage                                        |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdClearAttachments                                 |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdClearColorImage                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdClearDepthStencilImage                           |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdCopyBuffer                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdCopyBufferToImage                                |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdCopyImage                                        |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdCopyImageToBuffer                                |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdCopyQueryPoolResults                             |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDispatch                                         |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDispatchIndirect                                 |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDraw                                             |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDrawIndexed                                      |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDrawIndexedIndirect                              |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdDrawIndirect                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdEndQuery                                         |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdEndRenderPass                                    |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdExecuteCommands                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdFillBuffer                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdNextSubpass                                      |   :white_medium_square:   |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdPipelineBarrier                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdPushConstants                                    |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdResetEvent                                       |   :white_medium_square:   |      :white_medium_square:      |      :white_medium_square:      |
| vkCmdResetQueryPool                                   |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdResolveImage                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetBlendConstants                                |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetDepthBias                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetDepthBounds                                   |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetEvent                                         |   :white_medium_square:   |      :white_medium_square:      |      :white_check_mark:         |
| vkCmdSetLineWidth                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetScissor                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilCompareMask                            |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilReference                              |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilWriteMask                              |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetViewport                                      |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdUpdateBuffer                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdWaitEvents                                       |   :white_medium_square:   |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdWriteTimestamp                                   |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCreateAndroidSurfaceKHR                             |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateBuffer                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateBufferView                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateCommandPool                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateComputePipelines                              |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateDescriptorPool                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateDescriptorSetLayout                           |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateDisplayModeKHR                                |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateDisplayPlaneSurfaceKHR                        |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateEvent                                         |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateFramebuffer                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateGraphicsPipelines                             |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateImage                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateImageView                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateMirSurfaceKHR                                 |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreatePipelineCache                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreatePipelineLayout                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateQueryPool                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateRenderPass                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSampler                                       |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSemaphore                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateShaderModule                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSharedSwapchainsKHR                           |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateSwapchainKHR                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateWaylandSurfaceKHR                             |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateWin32SurfaceKHR                               |   :white_check_mark:      |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateXcbSurfaceKHR                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateXlibSurfaceKHR                                |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkDestroyBuffer                                       |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyBufferView                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyCommandPool                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyDescriptorPool                               |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyDescriptorSetLayout                          |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyEvent                                        |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkDestroyFramebuffer                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyImage                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyImageView                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyPipeline                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyPipelineCache                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyPipelineLayout                               |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyQueryPool                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyRenderPass                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroySampler                                      |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroySemaphore                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyShaderModule                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroySwapchainKHR                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDeviceWaitIdle                                      |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkEndCommandBuffer                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkFlushMappedMemoryRanges                             |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkFreeDescriptorSets                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkFreeMemory                                          |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkGetBufferMemoryRequirements                         |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetDeviceMemoryCommitment                           |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetDisplayModePropertiesKHR                         |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetDisplayPlaneCapabilitiesKHR                      |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetDisplayPlaneSupportedDisplaysKHR                 |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetEventStatus                                      |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetFenceStatus                                      |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageMemoryRequirements                          |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageSparseMemoryRequirements                    |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageSubresourceLayout                           |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceDisplayPlanePropertiesKHR          |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceDisplayPropertiesKHR               |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceMirPresentationSupportKHR          |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceCapabilitiesKHR             |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceFormatsKHR                  |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfacePresentModesKHR             |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceSupportKHR                  |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceWaylandPresentationSupportKHR      |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceWin32PresentationSupportKHR        |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceXcbPresentationSupportKHR          |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceXlibPresentationSupportKHR         |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPipelineCacheData                                |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetQueryPoolResults                                 |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetRenderAreaGranularity                            |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetSwapchainImagesKHR                               |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkInvalidateMappedMemoryRanges                        |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkMapMemory                                           |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkMergePipelineCaches                                 |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkQueueBindSparse                                     |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkQueuePresentKHR                                     |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkQueueSubmit                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkQueueWaitIdle                                       |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkResetCommandBuffer                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetCommandPool                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetDescriptorPool                                 |   :white_check_mark:      |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkResetEvent                                          |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkResetFences                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkSetEvent                                            |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkUnmapMemory                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkUpdateDescriptorSets                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkWaitForFences                                       |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkCreateFence                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyFence                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroySurfaceKHR                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
