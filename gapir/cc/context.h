/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef GAPIR_CONTEXT_H
#define GAPIR_CONTEXT_H

#include "core/cc/crash_handler.h"
#include "core/cc/target.h"
#include "core/cc/timer.h"

#include "gapir/cc/renderer.h"
#include "gapir/cc/replay_service.h"

#include <memory>
#include <string>
#include <unordered_map>

namespace gapir {

class Interpreter;
class MemoryManager;
class PostBuffer;
class ReplayRequest;
class ResourceCache;
class ResourceLoader;
class Stack;
class VulkanRenderer;

// Context object for the replay containing the Gl context, the memory manager
// and the replay specific functions to handle network communication of the
// interpreter
class Context : private Renderer::Listener {
 public:
  // Creates a new Context object and initialize it with loading the replay
  // request, setting up the memory manager, setting up the caches and
  // prefetching the resources
  static std::unique_ptr<Context> create(ReplayService* srv,
                                         core::CrashHandler& crash_handler,
                                         ResourceLoader* resource_loader,
                                         MemoryManager* memory_manager);

  virtual ~Context();

  void prefetch(ResourceCache* cache) const;

  // Run the interpreter over the opcode stream of the replay request and
  // returns true if the interpretation was successful false otherwise
  // If cleanup is false, then the next time this context is used,
  // It will continue from where it was.
  bool interpret(bool cleanup = true, bool isPrewarm = false);

  // Renderer::Listener compliance
  virtual void onDebugMessage(uint32_t severity, uint8_t api_index,
                              const char* msg) override;

  // Initialize the context object with loading the replay request, setting up
  // the memory manager, setting up the caches and prefetching the resources
  bool initialize(const std::string& id);

  // Clean up the context for the next replay.
  bool cleanup();

 private:
  enum {
    MAX_TIMERS = 256,
    POST_BUFFER_SIZE = 2 * 1024 * 1024,
  };

  Context(ReplayService* srv, core::CrashHandler& crash_handler,
          ResourceLoader* resource_loader, MemoryManager* memory_manager);

  // Register the callbacks for the interpreter (Gl functions, load resource,
  // post resource)
  void registerCallbacks(Interpreter* interpreter);

  // Post a chunk of data where the number of bytes is on the top of the stack
  // (uint32_t) and the address for the data is the second element on the stack
  // (void*)
  bool postData(Stack* stack);

  // Load a resource from the resource loader where the index of the resource
  // is at the top of the stack (uint32_t) and the target address is at the
  // second element of the stack (void*)
  bool loadResource(Stack* stack);

  // Tell the server that we're waiting for it to respond
  bool waitForFence(Stack* stack);

  // Starts the timer identified by u8 index.
  bool startTimer(Stack* stack);

  // Stops the timer identified by u8 index and returns u64 elapsed nanoseconds
  // since its start.
  bool stopTimer(Stack* stack, bool pushReturn);

  // Flushes any pending post data buffered from calling postData.
  bool flushPostBuffer(Stack* stack);

  // Send a chunk of notification data where the number of bytes is on the top
  // of the stack (uint32_t) and the address for the data is the second element
  // on the stack (void*)
  bool sendNotificationData(Stack* stack);

  // Server object to fetch and post resources back to.
  ReplayService* mSrv;

  // The crash handler used for catching and reporting crashes.
  core::CrashHandler& mCrashHandler;

  // Resource loader (possibly with caching) to fetch the resources required
  // by the replay. It is owned by the creator of the Context object.
  ResourceLoader* mResourceLoader;

  // Memory manager to manage the memory used by the replay and by the
  // interpreter. Is is owned by the creator of the Context object.
  MemoryManager* mMemoryManager;

  // The data of the request for this context belongs to
  std::unique_ptr<ReplayRequest> mReplayRequest;

  // An array of timers.
  core::Timer mTimers[MAX_TIMERS];

  // The lazily-built Vulkan renderer.
  VulkanRenderer* mVulkanRenderer;

  // A buffer for data to be sent back to the server.
  std::unique_ptr<PostBuffer> mPostBuffer;

  // The currently running interpreter.
  // Only valid for the duration of interpret()
  std::unique_ptr<Interpreter> mInterpreter;

  // The total number of debug messages sent to GAPIS.
  uint64_t mNumSentDebugMessages;
};

}  // namespace gapir

#endif  // GAPIR_CONTEXT_H
