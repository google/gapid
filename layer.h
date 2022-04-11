#pragma once

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

struct LayerOptions;

void (*LayerOptions_CaptureCommands)(LayerOptions*, VkCommandBuffer);
void (*LayerOptions_CaptureAllCommands)(LayerOptions*);

struct LayerOptions {
#ifdef __cplusplus
  void CaptureCommands(VkCommandBuffer cb) {
    LayerOptions_CaptureCommands(this, cb);
  }

  void CaptureAllCommands() { LayerOptions_CaptureAllCommands(this); }
#endif
};

#include "layer_internal_setup.h"

#include "layer_internal.inl"
