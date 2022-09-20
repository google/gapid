#include <chrono>
#include <format>
#include <fstream>
#include <iostream>
#include <sstream>

#include "layer.h"

namespace foo {
/*
static bool re_recording = false;
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
  std::cout << __FUNCTION__ << " QUEUE SUBMIT: "
            << reinterpret_cast<uintptr_t>(queue) << std::endl;
  re_recording = true;
  for (size_t i = 0; i < submitCount; ++i) {
    for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
      VkCommandBuffer cb = pSubmits[i].pCommandBuffers[i];
      OutputDebugStringA(std::format("Rerecording command buffer {}\n", reinterpret_cast<uintptr_t>(cb)).c_str());
      Rerecord_CommandBuffer(cb);
    }
  }
  re_recording = false;
  return vkQueueSubmit(queue, submitCount, pSubmits, fence);
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkBeginCommandBuffer(VkCommandBuffer commandBuffer, const VkCommandBufferBeginInfo* pBeginInfo) {
  if (re_recording) {
    OutputDebugStringA(std::format("RERECORDING {}\n", reinterpret_cast<uintptr_t>(commandBuffer)).c_str());
  } else {
    OutputDebugStringA(std::format("INITIALRECORDING {}\n", reinterpret_cast<uintptr_t>(commandBuffer)).c_str());
  }
  return vkBeginCommandBuffer(commandBuffer, pBeginInfo);
}

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  options->CaptureCommands(VK_NULL_HANDLE);
  options->CaptureAllCommands();
}
*/
static std::chrono::high_resolution_clock::time_point reset_time = std::chrono::high_resolution_clock::now();
static std::ofstream of("D:\\src\\data.out");
VKAPI_ATTR VkResult VKAPI_CALL override_vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
  of << std::chrono::duration_cast<std::chrono::microseconds>(std::chrono::high_resolution_clock::now() - reset_time).count() << std::endl;
  return vkQueuePresentKHR(queue, pPresentInfo);
}
//
}  // namespace foo