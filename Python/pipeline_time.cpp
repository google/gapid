#include <chrono>

#include "json.hpp"
#include "layer.h"

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions*) {}

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateGraphicsPipelines(
    VkDevice device,
    VkPipelineCache pipelineCache,
    uint32_t createInfoCount,
    const VkGraphicsPipelineCreateInfo* pCreateInfos,
    const VkAllocationCallbacks* pAllocator,
    VkPipeline* pPipelines) {
  auto start = std::chrono::high_resolution_clock::now();
  auto ret = vkCreateGraphicsPipelines(device,
                                       pipelineCache,
                                       createInfoCount,
                                       pCreateInfos,
                                       pAllocator,
                                       pPipelines);
  if (ret != VK_SUCCESS) {
    return ret;
  }
  auto diff = std::chrono::duration_cast<std::chrono::nanoseconds>(std::chrono::high_resolution_clock::now() - start).count();
  auto data = nlohmann::json();
  auto pipelines = nlohmann::json();
  for (size_t i = 0; i < createInfoCount; ++i) {
    pipelines.push_back(reinterpret_cast<uintptr_t>(pPipelines[i]));
  }
  data["pipelines"] = pipelines;
  data["time"] = diff;

  auto str = data.dump();
  SendJson(str.c_str(), str.size());

  return ret;
}