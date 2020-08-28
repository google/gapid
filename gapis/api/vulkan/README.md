# Vulkan Status

## Trace and Replay
Vulkan is currently WIP. Many samples, and applications work correctly, but some
bugs may still be present.

## Mid-Execution Capture
Mid-Execution capture allows an application to be traced starting at an arbitrary point in time.

If tracing from the command line this can be done with

`gapit trace -start-defer`

When using the GUI, this can be acheived by unchecking
`Trace from Beginning`

### Invalid VkDestoryXXX and VkFreeXXX Commands Due to Mid-Execution Capture
When using Mid-Execution capture, it is possible that at the time of
capturing, an object's dependee has been destroyed but the object itself is
still there, such objects will **NOT** be rebuilt for replay and the
`VkDestroyXXX` referring to such object will be **dropped** for replay, also
`VkFreeXXX` commands referring to such objects will be **modified** or
**dropped** to not freeing those not-rebuilt objects.

For example: A `VkImage` might have been destroyed when a
Mid-Execution capture starts, but the `VkImageView` handles that were created
with the destroyed `VkImage` might still be there. In such cases, those
`VkImageView` handles are invalid and will not be created during replay, also
the `VkFramebuffer` handles that depend on those `VkImageView` handles will
not be created during replay too. `VkDestroyImageView` and `VkDestroyFramebuffer`
commands that refer to those `VkImageView` and `VkFramebuffer` handles will be
dropped, so won't be called during replay.

## Subcommands
When visualizing the tree of Commands, every VkQueueSubmit is expanded into
a list of the commands that are run during that submission. From there you can
query information about any call in the program.

## Performance
We are still tuning performance for Vulkan in GAPID. For Posix based platforms
we handle persistently mapped coherent memory efficiently, but for Windows this
is currently in progress. Large blocks of mapped coherent memory can greatly
reduce replay performance and increase trace size.

## Test applications
We use a set of [Test Applications](https://github.com/google/vulkan_test_applications) to validate
whether or not Vulkan support is functioning. This repository contains
applications that use most parts of the API, and will be expanded as more interesting and tricky
uses of the API are found.

## Partial extension support
Some extensions are not supported by GAPID, but have stubs defined in the API
files.
The current list of partially supported extensions:

* VK_EXT_conditional_rendering

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
| vkCmdExecuteCommands                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_medium_square:      |
| vkCmdFillBuffer                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdNextSubpass                                      |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdPipelineBarrier                                  |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdPushConstants                                    |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdResetEvent                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdResetQueryPool                                   |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdResolveImage                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetBlendConstants                                |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetDepthBias                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetDepthBounds                                   |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetEvent                                         |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetLineWidth                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetScissor                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilCompareMask                            |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilReference                              |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetStencilWriteMask                              |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdSetViewport                                      |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdUpdateBuffer                                     |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
| vkCmdWaitEvents                                       |   :white_check_mark:      |      :white_check_mark:         |      :white_check_mark:         |
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
| vkCreateEvent                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateFramebuffer                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateGraphicsPipelines                             |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateImage                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateImageView                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateMirSurfaceKHR                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreatePipelineCache                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreatePipelineLayout                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateQueryPool                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateRenderPass                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSampler                                       |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSemaphore                                     |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateShaderModule                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateSharedSwapchainsKHR                           |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkCreateSwapchainKHR                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateWaylandSurfaceKHR                             |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateWin32SurfaceKHR                               |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateXcbSurfaceKHR                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkCreateXlibSurfaceKHR                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyBuffer                                       |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyBufferView                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyCommandPool                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyDescriptorPool                               |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyDescriptorSetLayout                          |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyEvent                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
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
| vkGetEventStatus                                      |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetFenceStatus                                      |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageMemoryRequirements                          |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageSparseMemoryRequirements                    |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetImageSubresourceLayout                           |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceDisplayPlanePropertiesKHR          |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceDisplayPropertiesKHR               |   :white_medium_square:   |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceMirPresentationSupportKHR          |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceCapabilitiesKHR             |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceFormatsKHR                  |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfacePresentModesKHR             |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceSurfaceSupportKHR                  |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceWaylandPresentationSupportKHR      |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceWin32PresentationSupportKHR        |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceXcbPresentationSupportKHR          |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPhysicalDeviceXlibPresentationSupportKHR         |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetPipelineCacheData                                |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetQueryPoolResults                                 |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetRenderAreaGranularity                            |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkGetSwapchainImagesKHR                               |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkInvalidateMappedMemoryRanges                        |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkMapMemory                                           |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkMergePipelineCaches                                 |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkQueueBindSparse                                     |   :white_medium_square:   |      :white_medium_square:      |      :heavy_minus_sign:         |
| vkQueuePresentKHR                                     |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkQueueSubmit                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkQueueWaitIdle                                       |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkResetCommandBuffer                                  |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetCommandPool                                    |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetDescriptorPool                                 |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetEvent                                          |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkResetFences                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkSetEvent                                            |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkUnmapMemory                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkUpdateDescriptorSets                                |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkWaitForFences                                       |   :white_check_mark:      |      :heavy_minus_sign:         |      :heavy_minus_sign:         |
| vkCreateFence                                         |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroyFence                                        |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
| vkDestroySurfaceKHR                                   |   :white_check_mark:      |      :white_check_mark:         |      :heavy_minus_sign:         |
