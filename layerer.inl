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

 private:
  bool captureAll = false;
  std::unordered_set<VkCommandBuffer> buffersToCheck;
};

template <typename T>
void* Layerer<T>::ResolveHelperFunction(const char* name, void** fout) {
  if (!strcmp(name, "LayerOptions_CaptureCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureCommandsForward);
  }
  if (!strcmp(name, "LayerOptions_CaptureAllCommands")) {
    return reinterpret_cast<void*>(&LayerOptions::CaptureAllCommandsForward);
  }
  return nullptr;
}

template <typename T>
void Layerer<T>::RunUserSetup(HMODULE module) {
  auto setup = (void* (*)(LayerOptions*))GetProcAddress(module, "SetupLayer");
  LayerOptions lo;
  if (setup) {
    OutputDebugStringA("Running user setup for layer");
    setup(&lo);
  } else {
    OutputDebugStringA("No user setup found for layer");
  }
}

}  // namespace gapid2