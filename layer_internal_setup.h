#pragma once

void (*LayerOptions_CaptureCommands_internal)(LayerOptions*, VkCommandBuffer);
void (*LayerOptions_CaptureAllCommands_internal)(LayerOptions*);

void SetupInternalPointers(void* user_data,
                           void*(fn)(void*, const char*, void**)) {
  void* unused_user_data;
  LayerOptions_CaptureCommands_internal =
      (void (*)(LayerOptions*, VkCommandBuffer))fn(
          user_data, "LayerOptions_CaptureCommands", &unused_user_data);
  LayerOptions_CaptureAllCommands_internal = (void (*)(LayerOptions*))fn(
      user_data, "LayerOptions_CaptureAllCommands", &unused_user_data);
}

void* Rerecord_CommandBuffer_internal_user_data;
void (*Rerecord_CommandBuffer_internal)(void* user_data, VkCommandBuffer cb);

extern "C" __declspec(dllexport) void PostSetupInternalPointers(
    void* user_data,
    void*(fn)(void*, const char*, void**)) {
  Rerecord_CommandBuffer_internal = (void (*)(void*, VkCommandBuffer))fn(
      user_data, "Rerecord_CommandBuffer",
      &Rerecord_CommandBuffer_internal_user_data);
}

void Rerecord_CommandBuffer(VkCommandBuffer cb) {
  return (*Rerecord_CommandBuffer_internal)(
      Rerecord_CommandBuffer_internal_user_data, cb);
}

void LayerOptions_CaptureCommands(LayerOptions* o, VkCommandBuffer cb) {
  return (*LayerOptions_CaptureCommands_internal)(o, cb);
}

void LayerOptions_CaptureAllCommands(LayerOptions* o) {
  return (*LayerOptions_CaptureAllCommands_internal)(o);
}