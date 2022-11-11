#include <iostream>
#include <shared_mutex>
#include <unordered_set>

#include "command_buffer_recorder.h"
#include "command_buffer_splitter.h"
#include "common.h"
#include "json.hpp"
#include "layerer.h"
#include "transform.h"

namespace gapid2 {
struct LayerOptions {
  void CaptureCommands(VkCommandBuffer cb) {
    if (captureAll) {
      std::cerr << "Not adding " << cb
                << " to the list of command buffers to track because all are "
                   "being tracked"
                << std::endl;
    }
    std::cerr << "Adding " << cb << " to the list of command buffers to track"
              << std::endl;
    buffersToCheck.insert(cb);
  };

  void CaptureAllCommands() {
    std::cerr << "Tracking all command buffers for layer " << std::endl;
    captureAll = true;
    buffersToCheck.clear();
  }

  const char* GetUserConfig() {
    return userConfig.c_str();
  }

  static void CaptureCommandsForward(LayerOptions* opts, VkCommandBuffer cb) {
    return opts->CaptureCommands(cb);
  }

  static void CaptureAllCommandsForward(LayerOptions* opts) {
    return opts->CaptureAllCommands();
  }

  static const char* GetUserConfigForward(LayerOptions* opts) {
    return opts->GetUserConfig();
  }

  bool captureAll = false;
  std::unordered_set<VkCommandBuffer> buffersToCheck;
  std::string userConfig;
};

void SendJson(void* user_data, const char* message, size_t length) {
  send_layer_data(message, length, static_cast<uint64_t>(reinterpret_cast<uintptr_t>(user_data)));
}

void LogMessage(void* user_data, uint32_t level, const char* message, size_t length) {
  send_layer_log(static_cast<message_type>(level), message, length, static_cast<uint64_t>(reinterpret_cast<uintptr_t>(user_data)));
}

uint64_t GetCommandIndex(void* user_data) {
  return reinterpret_cast<layerer*>(user_data)->get_current_command_index();
}

inline void* layerer::ResolveHelperFunction(uint64_t layer_idx,
                                            const char* name,
                                            void** fout) {
  if (!strcmp(name, "LayerOptions_CaptureCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureCommandsForward);
  }
  if (!strcmp(name, "LayerOptions_CaptureAllCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureAllCommandsForward);
  }
  if (!strcmp(name, "LayerOptions_GetUserConfig")) {
    return reinterpret_cast<void*>(&LayerOptions::GetUserConfigForward);
  }
  if (!strcmp(name, "SendJson")) {
    *fout = reinterpret_cast<void*>(static_cast<uintptr_t>(layer_idx));
    return reinterpret_cast<void*>(&SendJson);
  }
  if (!strcmp(name, "LogMessage")) {
    *fout = reinterpret_cast<void*>(static_cast<uintptr_t>(layer_idx));
    return reinterpret_cast<void*>(&LogMessage);
  }
  if (!strcmp(name, "GetCommandIndex")) {
    *fout = reinterpret_cast<void*>(this);
    return reinterpret_cast<void*>(&GetCommandIndex);
  }

  return nullptr;
}

struct fs {
  transform<command_buffer_recorder>* cbr;
  transform<command_buffer_splitter>* cbs;
  layerer* layerer;
};

void call_rerecord(void* data, VkCommandBuffer cb) {
  auto* f = reinterpret_cast<fs*>(data);
  return f->cbr->RerecordCommandBuffer(cb, f->layerer);
}

void call_split(void* data, VkCommandBuffer cb, uint64_t* indices, uint32_t index) {
  auto* f = reinterpret_cast<fs*>(data);
  return f->cbs->SplitCommandBuffer(cb, f->layerer, indices, index);
}

inline void layerer::RunUserSetup(const std::string& layer_name, HMODULE module) {
  auto setup = (void* (*)(LayerOptions*))GetProcAddress(module, "SetupLayer");
  LayerOptions lo;
  lo.userConfig = "";
  if (!user_config_.empty()) {
    auto setup = nlohmann::json::parse(user_config_, nullptr, false);
    if (setup.contains(layer_name)) {
      lo.userConfig = setup[layer_name].dump();
    }
  }

  if (setup) {
    OutputDebugStringA("Running user setup for layer\n");
    setup(&lo);
  } else {
    OutputDebugStringA("No user setup found for layer\n");
  }

  transform<command_buffer_recorder>* cbr = nullptr;
  transform<command_buffer_splitter>* cbs = nullptr;

  if (lo.captureAll || lo.buffersToCheck.empty()) {
    auto cb = std::make_unique<command_buffer_splitter_layers>(this);

    OutputDebugStringA("Setting up command buffer recorder for layer\n");
    cbr = cb->command_buffer_recorder_.get();
    cbs = cb->command_buffer_splitter_.get();
    auto setup = (void (*)(VkCommandBuffer))GetProcAddress(module, "OnCommandBufferSplit");
    cbs->on_command_buffer_split_ = setup;
    splitters.push_back(std::move(cb));
  }

  auto post_setup =
      (void (*)(void*, void* (*)(void*, const char*, void**)))GetProcAddress(
          module, "PostSetupInternalPointers");
  if (!post_setup) {
    OutputDebugStringA(
        "Unknown layer data, missing PostSetupInternalPointers\n");
    return;
  }

  post_setup(new fs{cbr, cbs, this}, [](void* cb, const char* fn_name, void** user_data) -> void* {
    if (!strcmp(fn_name, "Rerecord_CommandBuffer")) {
      *user_data = cb;
      if (!cb) {
        return nullptr;
      }
      return reinterpret_cast<void*>(&call_rerecord);
    }
    if (!strcmp(fn_name, "Split_CommandBuffer")) {
      *user_data = cb;
      if (!cb) {
        return nullptr;
      }
      return reinterpret_cast<void*>(&call_split);
    }
    OutputDebugStringA("Invalid setup call\n");
    return nullptr;
  });
}

inline void layerer::RunUserShutdown(HMODULE module) {
  auto setup = (void* (*)())GetProcAddress(module, "ShutdownLayer");
  if (setup) {
    OutputDebugStringA("Running user shutdown for layer\n");
    setup();
  } else {
    OutputDebugStringA("No user shutdown found for layer\n");
  }
}

}  // namespace gapid2