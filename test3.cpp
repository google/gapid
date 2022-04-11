#include "layer.h"

#include <iostream>

namespace foo {
VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkInstance* pInstance) {
  std::cout << __FUNCTION__ << " :: " << std::endl;

  return vkCreateInstance(pCreateInfo, pAllocator, pInstance);
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkSignalSemaphoreKHR(VkDevice device,
                              const VkSemaphoreSignalInfo* pSignalInfo) {
  std::cout << "vkSignalSemaphoreKHR { device: "
            << reinterpret_cast<uintptr_t>(device) << std::endl;

  std::cout << "  Semaphore : "
            << reinterpret_cast<uintptr_t>(pSignalInfo->semaphore) << std::endl;
  std::cout << "  Value :" << pSignalInfo->value << std::endl;

  return vkSignalSemaphoreKHR(device, pSignalInfo);
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkWaitSemaphoresKHR(VkDevice device,
                             const VkSemaphoreWaitInfo* pWaitInfo,
                             uint64_t timeout) {
  std::cout << "vkWaitSemaphoresKHR { device: "
            << reinterpret_cast<uintptr_t>(device) << std::endl;
  for (size_t i = 0; i < pWaitInfo->semaphoreCount; ++i) {
    std::cout << "  Semaphore " << i << " : "
              << reinterpret_cast<uintptr_t>(pWaitInfo->pSemaphores[i])
              << std::endl;
    if (pWaitInfo->pValues) {
      std::cout << "  Value " << i << " : " << pWaitInfo->pValues[i]
                << std::endl;
    }
  }
  return vkWaitSemaphoresKHR(device, pWaitInfo, timeout);
}

VKAPI_ATTR void VKAPI_CALL
override_vkCmdDrawIndexed(VkCommandBuffer commandBuffer,
                          uint32_t indexCount,
                          uint32_t instanceCount,
                          uint32_t firstIndex,
                          int32_t vertexOffset,
                          uint32_t firstInstance) {
  if (reinterpret_cast<uintptr_t>(commandBuffer) == 1933471327024ull + 1) {
    std::cout << "  Dropping " << __FUNCTION__ << " in commmand buffer  "
              << reinterpret_cast<uintptr_t>(commandBuffer) << std::endl;
    return;
  }
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkQueueWaitIdle(VkQueue queue) {
  auto ret = vkQueueWaitIdle(queue);
  if (ret != VK_SUCCESS) {
    std::cout << __FUNCTION__ << " QUEUE WAIT IDLE FAILED : "
              << reinterpret_cast<uintptr_t>(queue) << std::endl;
    std::abort();
  }
  return ret;
}

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  options->CaptureCommands(VK_NULL_HANDLE);
  options->CaptureAllCommands();
}

}  // namespace foo