#pragma once

void SetupInternalPointers(void* user_data,
                           void*(fn)(void*, const char*, void**)) {
  void* unused_user_data;
  LayerOptions_CaptureCommands = (void (*)(LayerOptions*, VkCommandBuffer))fn(
      user_data, "LayerOptions_CaptureCommands", &unused_user_data);
  LayerOptions_CaptureAllCommands = (void (*)(LayerOptions*))fn(
      user_data, "LayerOptions_CaptureAllCommands", &unused_user_data);
}
