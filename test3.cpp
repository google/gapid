#include "layer.h"

#include <iostream>
#include <sstream>

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

VKAPI_ATTR VkResult VKAPI_CALL
override_vkSignalSemaphore(VkDevice device,
                           const VkSemaphoreSignalInfo* pSignalInfo) {
  std::cout << "vkSignalSemaphore { device: "
            << reinterpret_cast<uintptr_t>(device) << std::endl;

  std::cout << "  Semaphore : "
            << reinterpret_cast<uintptr_t>(pSignalInfo->semaphore) << std::endl;
  std::cout << "  Value :" << pSignalInfo->value << std::endl;

  return vkSignalSemaphore(device, pSignalInfo);
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkWaitSemaphores(VkDevice device,
                          const VkSemaphoreWaitInfo* pWaitInfo,
                          uint64_t timeout) {
  std::cout << "vkWaitSemaphores { device: "
            << reinterpret_cast<uintptr_t>(device) << std::endl;
  std::cout << " timeout: " << timeout << std::endl;
  for (size_t i = 0; i < pWaitInfo->semaphoreCount; ++i) {
    std::cout << "  Semaphore " << i << " : "
              << reinterpret_cast<uintptr_t>(pWaitInfo->pSemaphores[i])
              << std::endl;
    if (pWaitInfo->pValues) {
      std::cout << "  Value " << i << " : " << pWaitInfo->pValues[i]
                << std::endl;
    }
  }
  return vkWaitSemaphores(device, pWaitInfo, timeout);
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkQueueWaitIdle(VkQueue queue) {
  std::cout << __FUNCTION__
            << " QUEUE WAIT IDLE: " << reinterpret_cast<uintptr_t>(queue)
            << std::endl;
  auto ret = vkQueueWaitIdle(queue);
  if (ret != VK_SUCCESS) {
    std::cout << __FUNCTION__ << " QUEUE WAIT IDLE FAILED : "
              << reinterpret_cast<uintptr_t>(queue) << std::endl;
    std::abort();
  }
  return ret;
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkQueueSubmit(VkQueue queue,
                       uint32_t submitCount,
                       const VkSubmitInfo* pSubmits,
                       VkFence fence) {
  for (size_t i = 0; i < submitCount; ++i) {
    for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
      VkCommandBuffer cb = pSubmits[i].pCommandBuffers[i];
      std::stringstream ss;
      ss << "Rerecording command buffer " << cb << std::endl;
      OutputDebugStringA(ss.str().c_str());
      Rerecord_CommandBuffer(cb);
    }
  }
  return vkQueueSubmit(queue, submitCount, pSubmits, fence);
}

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  options->CaptureCommands(VK_NULL_HANDLE);
  options->CaptureAllCommands();
}

}  // namespace foo