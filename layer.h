#pragma once

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#ifdef __cplusplus
#include <string>
#endif
struct LayerOptions;

enum LogType {
  debug = 0,
  info = 1,
  error = 2,
  critical = 3,
};

void LayerOptions_CaptureCommands(LayerOptions*, VkCommandBuffer);
void LayerOptions_CaptureAllCommands(LayerOptions*);
void Rerecord_CommandBuffer(VkCommandBuffer cb);
void Split_CommandBuffer(VkCommandBuffer cb, uint64_t* indices, uint32_t num_indices);
void SendJson(const char* json, size_t length);
void LogMessage(LogType type, const char* json, size_t length);
uint64_t GetCommandIndex();
const char* LayerOptions_GetUserConfig(LayerOptions* options);

struct LayerOptions {
#ifdef __cplusplus
  void CaptureCommands(VkCommandBuffer cb) {
    LayerOptions_CaptureCommands(this, cb);
  }

  void CaptureAllCommands() { LayerOptions_CaptureAllCommands(this); }

  const char* GetUserConfig() { return LayerOptions_GetUserConfig(this); };
#endif
};

#ifdef __cplusplus
void SendJson(const std::string& str) {
  return SendJson(str.c_str(), str.size());
}

void LogMessage(LogType type, const std::string& str) {
  return LogMessage(type, str.c_str(), str.size());
}
#endif

#include "layer_internal_setup.h"
// Don't reorder these
#include "layer_internal.inl"
