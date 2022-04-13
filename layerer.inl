#include <iostream>
#include <shared_mutex>
#include <unordered_set>
#include "layerer.h"

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

  static void CaptureCommandsForward(LayerOptions* opts, VkCommandBuffer cb) {
    return opts->CaptureCommands(cb);
  }

  static void CaptureAllCommandsForward(LayerOptions* opts) {
    return opts->CaptureAllCommands();
  }

  bool captureAll = false;
  std::unordered_set<VkCommandBuffer> buffersToCheck;
};

template <typename T, typename HandleUpdater>
void* Layerer<T, HandleUpdater>::ResolveHelperFunction(const char* name,
                                                       void** fout) {
  if (!strcmp(name, "LayerOptions_CaptureCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureCommandsForward);
  }
  if (!strcmp(name, "LayerOptions_CaptureAllCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureAllCommandsForward);
  }
  return nullptr;
}

template <typename HandleUpdater>
void call_rerecord(void* data, VkCommandBuffer cb) {
  auto* cbr = reinterpret_cast<CommandBufferRecorder<HandleUpdater>*>(data);
  return cbr->RerecordCommandBuffer(cb);
}

template <typename T, typename HandleUpdater>
void Layerer<T, HandleUpdater>::RunUserSetup(HMODULE module) {
  auto setup = (void* (*)(LayerOptions*))GetProcAddress(module, "SetupLayer");
  LayerOptions lo;
  if (setup) {
    OutputDebugStringA("Running user setup for layer\n");
    setup(&lo);
  } else {
    OutputDebugStringA("No user setup found for layer\n");
  }
  CommandBufferRecorder<HandleUpdater>* cbr = nullptr;

  if (lo.captureAll || lo.buffersToCheck.empty()) {
    auto cb = std::make_unique<CommandBufferRecorder<HandleUpdater>>(f);

    OutputDebugStringA("Setting up command buffer recorder for layer\n");

    cb->updater = updater;
    cbr = cb.get();
    recorders.push_back(std::move(cb));
  }

  auto post_setup =
      (void (*)(void*, void* (*)(void*, const char*, void**)))GetProcAddress(
          module, "PostSetupInternalPointers");
  if (!post_setup) {
    OutputDebugStringA(
        "Unknown layer data, missing PostSetupInternalPointers\n");
    return;
  }

  post_setup(cbr, [](void* cb, const char* fn_name, void** user_data) -> void* {
    if (!strcmp(fn_name, "Rerecord_CommandBuffer")) {
      *user_data = cb;
      if (!cb) {
        return nullptr;
      }
      return reinterpret_cast<void*>(&call_rerecord<HandleUpdater>);
    }
    OutputDebugStringA("Invalid setup call\n");
    return nullptr;
  });
}

}  // namespace gapid2