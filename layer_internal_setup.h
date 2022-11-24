#pragma once

void (*LayerOptions_CaptureCommands_internal)(LayerOptions*, VkCommandBuffer);
void (*LayerOptions_CaptureAllCommands_internal)(LayerOptions*);
const char* (*LayerOptions_GetUserConfig_internal)(LayerOptions*);
void (*SendJson_internal)(void* user_data, const char* json, size_t length);
void* SendJson_user_data;
void (*LogMessage_internal)(void* user_data, uint32_t log_level, const char* json, size_t length);
void* LogMessage_user_data;
uint64_t (*GetCommandIndex_internal)(void* user_data);
void* GetCommandIndex_user_data;

void SetupInternalPointers(void* user_data,
                           void*(fn)(void*, const char*, void**)) {
  void* unused_user_data;
  LayerOptions_CaptureCommands_internal =
      (void (*)(LayerOptions*, VkCommandBuffer))fn(
          user_data, "LayerOptions_CaptureCommands", &unused_user_data);
  LayerOptions_CaptureAllCommands_internal = (void (*)(LayerOptions*))fn(
      user_data, "LayerOptions_CaptureAllCommands", &unused_user_data);
  LayerOptions_GetUserConfig_internal = (const char* (*)(LayerOptions*))fn(
      user_data, "LayerOptions_GetUserConfig", &unused_user_data);
  SendJson_internal = (void (*)(void*, const char*, size_t))fn(
      user_data, "SendJson", &SendJson_user_data);
  GetCommandIndex_internal = (uint64_t(*)(void*))fn(
      user_data, "GetCommandIndex", &GetCommandIndex_user_data);
  LogMessage_internal = (void (*)(void*, uint32_t, const char*, size_t))fn(
      user_data, "LogMessage", &LogMessage_user_data);
}

void* Rerecord_CommandBuffer_internal_user_data;
void (*Rerecord_CommandBuffer_internal)(void* user_data, VkCommandBuffer cb);
void (*Split_CommandBuffer_internal)(void* user_data, VkCommandBuffer cb, const uint64_t* indices, uint32_t num_indices);

extern "C" __declspec(dllexport) void PostSetupInternalPointers(
    void* user_data,
    void*(fn)(void*, const char*, void**)) {
  Rerecord_CommandBuffer_internal = (void (*)(void*, VkCommandBuffer))fn(
      user_data, "Rerecord_CommandBuffer",
      &Rerecord_CommandBuffer_internal_user_data);
  Split_CommandBuffer_internal = (void (*)(void*, VkCommandBuffer, const uint64_t*, uint32_t))fn(
      user_data, "Split_CommandBuffer",
      &Rerecord_CommandBuffer_internal_user_data);
}

void Rerecord_CommandBuffer(VkCommandBuffer cb) {
  return (*Rerecord_CommandBuffer_internal)(
      Rerecord_CommandBuffer_internal_user_data, cb);
}

void Split_CommandBuffer(VkCommandBuffer cb, const uint64_t* indices, uint32_t num_indices) {
  return (*Split_CommandBuffer_internal)(
      Rerecord_CommandBuffer_internal_user_data, cb, indices, num_indices);
}

void LayerOptions_CaptureCommands(LayerOptions* o, VkCommandBuffer cb) {
  return (*LayerOptions_CaptureCommands_internal)(o, cb);
}

void LayerOptions_CaptureAllCommands(LayerOptions* o) {
  return (*LayerOptions_CaptureAllCommands_internal)(o);
}

const char* LayerOptions_GetUserConfig(LayerOptions* o) {
  return (*LayerOptions_GetUserConfig_internal)(o);
}

void SendJson(const char* json, size_t length) {
  SendJson_internal(SendJson_user_data, json, length);
}

void LogMessage(LogType log_level, const char* json, size_t length) {
  LogMessage_internal(LogMessage_user_data, static_cast<uint32_t>(log_level), json, length);
}

uint64_t GetCommandIndex() {
  return GetCommandIndex_internal(GetCommandIndex_user_data);
}