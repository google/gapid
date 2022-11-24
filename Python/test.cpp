#include "layer.h"

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  options->CaptureCommands(VK_NULL_HANDLE);
  options->CaptureAllCommands();
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateInstance(
    const VkInstanceCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkInstance* pInstance) {
  LogMessage(debug, "Random test message!");
  return vkCreateInstance(pCreateInfo, pAllocator, pInstance);
}
