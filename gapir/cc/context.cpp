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

#include "context.h"
#include "gapir/cc/gles_gfx_api.h"
#include "gapir/cc/vulkan_gfx_api.h"
#include "gles_renderer.h"
#include "interpreter.h"
#include "memory_manager.h"
#include "post_buffer.h"
#include "replay_request.h"
#include "replay_service.h"
#include "resource_cache.h"
#include "resource_loader.h"
#include "stack.h"
#include "vulkan_renderer.h"

#include "core/cc/gl/formats.h"
#include "core/cc/log.h"
#include "core/cc/target.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#include <cstdlib>
#include <sstream>
#include <string>

namespace gapir {

std::unique_ptr<Context> Context::create(ReplayService* srv,
                                         core::CrashHandler& crash_handler,
                                         ResourceLoader* resource_loader,
                                         MemoryManager* memory_manager) {
  std::unique_ptr<Context> context(
      new Context(srv, crash_handler, resource_loader, memory_manager));
  return context;
}

// TODO: Make the PostBuffer size dynamic? It currently holds 2MB of data.
Context::Context(ReplayService* srv, core::CrashHandler& crash_handler,
                 ResourceLoader* resource_loader, MemoryManager* memory_manager)
    : mSrv(srv),
      mCrashHandler(crash_handler),
      mResourceLoader(resource_loader),
      mMemoryManager(memory_manager),
      mVulkanRenderer(nullptr),
      mPostBuffer(new PostBuffer(
          POST_BUFFER_SIZE,
          [this](std::unique_ptr<ReplayService::Posts> posts) -> bool {
            if (mSrv != nullptr) {
              return mSrv->sendPosts(std::move(posts));
            }
            return false;
          })),
      mNumSentDebugMessages(0) {}

Context::~Context() {
  for (auto it = mGlesRenderers.begin(); it != mGlesRenderers.end(); it++) {
    delete it->second;
  }
  delete mVulkanRenderer;
}

bool Context::cleanup() {
  for (auto it = mGlesRenderers.begin(); it != mGlesRenderers.end(); it++) {
    delete it->second;
  }
  delete mVulkanRenderer;
  mGlesRenderers.clear();
  mVulkanRenderer = nullptr;
  return true;
}

bool Context::initialize(const std::string& id) {
  mReplayRequest = ReplayRequest::create(mSrv, id, mMemoryManager);
  mPostBuffer->resetCount();
  if (mReplayRequest == nullptr) {
    GAPID_ERROR("Replay request creation failed");
    return false;
  }

  GAPID_DEBUG("ReplayRequest created successfully");
  if (!mMemoryManager->setVolatileMemory(
          mReplayRequest->getVolatileMemorySize())) {
    GAPID_WARNING("Setting the volatile memory size failed (size: %u)",
                  mReplayRequest->getVolatileMemorySize());
    return false;
  }

  // Constant memory already set with ReplayData
  return true;
}

void Context::prefetch(ResourceCache* cache) const {
  auto resources = mReplayRequest->getResources();
  if (resources.size() == 0) {
    return;
  }

  auto tempLoader = PassThroughResourceLoader::create(mSrv);
  cache->setPrefetch(resources, std::move(tempLoader));
}

bool Context::interpret(bool cleanup, bool isPrewarm) {
  Interpreter::ApiRequestCallback callback = [this](Interpreter* interpreter,
                                                    uint8_t api_index) -> bool {
    if (api_index == gapir::Vulkan::INDEX) {
      // There is only one vulkan "renderer" so we create it when requested.
      mVulkanRenderer = VulkanRenderer::create();
      if (mVulkanRenderer->isValid()) {
        mVulkanRenderer->setListener(this);
        Api* api = mVulkanRenderer->api();
        interpreter->setRendererFunctions(api->index(), &api->mFunctions);
        GAPID_INFO("Bound Vulkan renderer");
        return true;
      }
    }
    return false;
  };
  Interpreter::CheckReplayStatusCallback replayStatusCallback =
      [this, isPrewarm](uint64_t label, uint32_t total_instrs,
                        uint32_t finished_instrs) {
        // Don't send replay status updates when it's prewarm replay.
        if (isPrewarm) {
          return;
        }
        // Send notification to GAPIS about every 1% of instructions done.
        // Worth noting it's not precisely every 1%, because GAPIR check
        // progress at each command call rather than each instruction. Also
        // extra check is given here to make sure to send notification when
        // approaching the last cmd.
        if (total_instrs < 100 ||
            (finished_instrs % (total_instrs / 100) == 0) ||
            (total_instrs - finished_instrs) <= 3) {
          mSrv->sendReplayStatus(label, total_instrs, finished_instrs);
        }
      };
  if (mInterpreter == nullptr) {
    mInterpreter.reset(new Interpreter(mCrashHandler, mMemoryManager,
                                       mReplayRequest->getStackSize()));
    registerCallbacks(mInterpreter.get());
  }
  mInterpreter->setApiRequestCallback(std::move(callback));
  mInterpreter->setCheckReplayStatusCallback(std::move(replayStatusCallback));

  auto instAndCount = mReplayRequest->getInstructionList();
  auto res = mInterpreter->run(instAndCount.first, instAndCount.second) &&
             mPostBuffer->flush();
  if (cleanup) {
    mInterpreter.reset(nullptr);
  } else {
    mInterpreter->resetInstructions();
  }
  return res;
}

// onDebugMessage implements the Renderer::Listener interface.
void Context::onDebugMessage(uint32_t severity, uint8_t api_index,
                             const char* msg) {
  auto label = mInterpreter->getLabel();
  // Remove tailing new-line from the message (if any)
  std::string str_msg;
  if (msg != nullptr) {
    auto len = strlen(msg);
    while (len >= 1 && msg[len - 1] == '\n') {
      len--;
    }
    str_msg = std::string(msg, len);
    msg = str_msg.data();
  }
  GAPID_DEBUG("[%d]renderer: %s", label, msg);
  mSrv->sendErrorMsg(mNumSentDebugMessages++, severity, api_index, label,
                     str_msg, nullptr, 0);
}

void Context::registerCallbacks(Interpreter* interpreter) {
  // Custom function for posting and fetching resources to and from the server
  interpreter->registerBuiltin(Interpreter::GLOBAL_INDEX,
                               Interpreter::POST_FUNCTION_ID,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->postData(stack);
                               });
  interpreter->registerBuiltin(Interpreter::GLOBAL_INDEX,
                               Interpreter::NOTIFICATION_FUNCTION_ID,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->sendNotificationData(stack);
                               });
  interpreter->registerBuiltin(Interpreter::GLOBAL_INDEX,
                               Interpreter::RESOURCE_FUNCTION_ID,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->loadResource(stack);
                               });
  interpreter->registerBuiltin(Interpreter::GLOBAL_INDEX,
                               Interpreter::WAIT_FUNCTION_ID,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->waitForFence(stack);
                               });

  // Registering custom synthetic functions
  interpreter->registerBuiltin(Gles::INDEX, Builtins::StartTimer,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->startTimer(stack);
                               });
  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::StopTimer,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        return this->stopTimer(stack, pushReturn);
      });
  interpreter->registerBuiltin(Gles::INDEX, Builtins::FlushPostBuffer,
                               [this](uint32_t label, Stack* stack, bool) {
                                 return this->flushPostBuffer(stack);
                               });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayFrameDelimiter,
      [this](uint32_t label, Stack* stack, bool) {
        uint32_t id = stack->pop<uint32_t>();
        if (stack->isValid()) {
          GAPID_INFO("[%u]replayFrameDelimiter(%u)", label, id);
          auto found = mGlesRenderers.find(id);
          if (found == mGlesRenderers.end()) {
            GAPID_ERROR("replayFrameDelimiter has no renderer at ID: %u", id);
            return false;
          }
          found->second->frameDelimiter();
          return true;
        } else {
          GAPID_WARNING(
              "[%u]Error during calling function replayFrameDelimiter", label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayCreateRenderer,
      [this](uint32_t label, Stack* stack, bool) {
        uint32_t id = stack->pop<uint32_t>();
        if (stack->isValid()) {
          GAPID_INFO("[%u]replayCreateRenderer(%u)", label, id);
          auto existing = mGlesRenderers.find(id);
          if (existing != mGlesRenderers.end()) {
            delete existing->second;
          }
          // Share objects with the root GLES context.
          // This will essentially make all objects shared between all contexts.
          // It is ok since correct replay will only reference what it is
          // supposed to.
          if (!mRootGlesRenderer) {
            mRootGlesRenderer.reset(GlesRenderer::create(nullptr));
            if (!mRootGlesRenderer) {
              GAPID_ERROR("Could not create GLES renderer on this device");
              return false;
            }
            mRootGlesRenderer->setBackbuffer(GlesRenderer::Backbuffer(
                8, 8, core::gl::GL_RGBA8, core::gl::GL_DEPTH24_STENCIL8,
                core::gl::GL_DEPTH24_STENCIL8));
          }
          auto renderer = GlesRenderer::create(mRootGlesRenderer.get());
          if (!renderer) {
            GAPID_ERROR("Could not create GLES renderer on this device");
            return false;
          }
          renderer->setListener(this);
          mGlesRenderers[id] = renderer;
          return true;
        } else {
          GAPID_WARNING(
              "[%u]Error during calling function replayCreateRenderer", label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayBindRenderer,
      [this, interpreter](uint32_t label, Stack* stack, bool) {
        bool resetViewportScissor = stack->pop<bool>();
        uint32_t id = stack->pop<uint32_t>();
        if (stack->isValid()) {
          GAPID_INFO("[%u]replayBindRenderer(%u, %s)", label, id,
                     resetViewportScissor ? "true" : "false");
          auto renderer = mGlesRenderers[id];
          renderer->bind(resetViewportScissor);
          Api* api = renderer->api();
          interpreter->setRendererFunctions(api->index(), &api->mFunctions);
          GAPID_DEBUG("[%u]Bound renderer %u: %s - %s", label, id,
                      renderer->name(), renderer->version());
          return true;
        } else {
          GAPID_WARNING("[%u]Error during calling function replayBindRenderer",
                        label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayUnbindRenderer,
      [this](uint32_t label, Stack* stack, bool) {
        uint32_t id = stack->pop<uint32_t>();
        if (stack->isValid()) {
          GAPID_DEBUG("[%u]replayUnbindRenderer(%" PRIu32 ")", label, id);
          auto renderer = mGlesRenderers[id];
          renderer->unbind();
          // TODO: Unbind renderer functions with the interpreter?
          // Api* api = renderer->api();
          // interpreter->setRendererFunctions(api->index(), nullptr);
          GAPID_DEBUG("[%u]Unbound renderer %" PRIu32, label, id);
          return true;
        } else {
          GAPID_WARNING(
              "[%u]Error during calling function replayUnbindRenderer", label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayChangeBackbuffer,
      [this](uint32_t label, Stack* stack, bool) {
        GlesRenderer::Backbuffer backbuffer;

        backbuffer.format.stencil = stack->pop<uint32_t>();
        backbuffer.format.depth = stack->pop<uint32_t>();
        backbuffer.format.color = stack->pop<uint32_t>();
        backbuffer.height = stack->pop<int32_t>();
        backbuffer.width = stack->pop<int32_t>();
        uint32_t id = stack->pop<uint32_t>();

        if (!stack->isValid()) {
          GAPID_WARNING(
              "[%u]Error during calling function replayChangeBackbuffer",
              label);
          return false;
        }

        if (stack->isValid()) {
          GAPID_INFO("[%u]replayChangeBackbuffer(%d, %d, 0x%x, 0x%x, 0x%x)",
                     label, backbuffer.width, backbuffer.height,
                     backbuffer.format.color, backbuffer.format.depth,
                     backbuffer.format.stencil);
          auto renderer = mGlesRenderers[id];
          if (renderer == nullptr) {
            GAPID_WARNING(
                "[%u]replayChangeBackbuffer called with unknown renderer "
                "%" PRIu32,
                label, id);
            return false;
          }
          renderer->setBackbuffer(backbuffer);
          return true;
        } else {
          GAPID_WARNING(
              "[%u]Error during calling function replayChangeBackbuffer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Gles::INDEX, Builtins::ReplayCreateExternalImage,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        uint32_t texId = stack->pop<uint32_t>();
        uint32_t ctxId = stack->pop<uint32_t>();

        if (!stack->isValid()) {
          GAPID_WARNING(
              "[%u]Error during calling function replayCreateExternalImage",
              label);
          return false;
        }

        auto renderer = mGlesRenderers[ctxId];
        if (renderer == nullptr) {
          GAPID_WARNING(
              "[%u]replayCreateExternalImage called with unknown renderer "
              "%" PRIu32,
              label, ctxId);
          return false;
        }

        GAPID_INFO("[%u]replayCreateExternalImage(%d, %d)", label, ctxId,
                   texId);
        auto result = renderer->createExternalImage(texId);
        if (pushReturn) {
          stack->push(result);
        }
        return true;
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayCreateVkInstance,
      [this, interpreter](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayCreateVkInstance()", label);

        if (mVulkanRenderer != nullptr ||
            interpreter->registerApi(Vulkan::INDEX)) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          auto* pInstance = stack->pop<Vulkan::VkInstance*>();
          auto* pAllocator = stack->pop<Vulkan::VkAllocationCallbacks*>();
          auto* pCreateInfo = stack->pop<Vulkan::VkInstanceCreateInfo*>();
          if (!stack->isValid()) {
            GAPID_ERROR("Error during calling funtion ReplayCreateVkInstance");
            return false;
          }
          uint32_t result = Vulkan::VkResult::VK_SUCCESS;

          if (api->replayCreateVkInstanceImpl(stack, pCreateInfo, pAllocator,
                                              pInstance, false, &result)) {
            if (result == Vulkan::VkResult::VK_SUCCESS) {
              if (pushReturn) {
                stack->push(result);
              }
              return true;
            }
          }
          // If validation layers and debug report extension are enabled,
          // drop them and try to create VkInstance again.
          if (Vulkan::hasValidationLayers(pCreateInfo->ppEnabledLayerNames,
                                          pCreateInfo->enabledLayerCount) ||
              Vulkan::hasDebugReportExtension(
                  pCreateInfo->ppEnabledExtensionNames,
                  pCreateInfo->enabledExtensionCount)) {
            onDebugMessage(
                LOG_LEVEL_WARNING, Vulkan::INDEX,
                "Failed to create VkInstance with validation layers and "
                "debug report extension, drop them and try again");
            if (api->replayCreateVkInstanceImpl(stack, pCreateInfo, pAllocator,
                                                pInstance, true, &result)) {
              if (pushReturn) {
                stack->push(result);
              }
              return true;
            }
          }
          onDebugMessage(LOG_LEVEL_FATAL, Vulkan::INDEX,
                         "Failed to create 'VkInstance'");
          return false;
        } else {
          GAPID_WARNING(
              "[%u]replayCreateVkInstance called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayCreateVkDevice,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayCreateVkDevice()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          auto pDevice = stack->pop<Vulkan::VkDevice*>();
          auto pAllocator = stack->pop<Vulkan::VkAllocationCallbacks*>();
          auto pCreateInfo = stack->pop<Vulkan::VkDeviceCreateInfo*>();
          auto physicalDevice = static_cast<size_val>(stack->pop<size_val>());
          if (!stack->isValid()) {
            GAPID_ERROR("Error during calling funtion ReplayCreateVkDevice");
            return false;
          }
          uint32_t result = Vulkan::VkResult::VK_SUCCESS;

          if (api->replayCreateVkDeviceImpl(stack, physicalDevice, pCreateInfo,
                                            pAllocator, pDevice, false,
                                            &result)) {
            if (result == Vulkan::VkResult::VK_SUCCESS) {
              if (pushReturn) {
                stack->push(result);
              }
              return true;
            }
          }
          // If validation layers are enabled, drop them and try to create
          // VkInstance again.
          if (Vulkan::hasValidationLayers(pCreateInfo->ppEnabledLayerNames,
                                          pCreateInfo->enabledLayerCount)) {
            onDebugMessage(LOG_LEVEL_WARNING, Vulkan::INDEX,
                           "Failed to create VkDevice with validation layers, "
                           "drop them and try again");
            if (api->replayCreateVkDeviceImpl(stack, physicalDevice,
                                              pCreateInfo, pAllocator, pDevice,
                                              true, &result)) {
              if (pushReturn) {
                stack->push(result);
              }
              return true;
            }
          }
          onDebugMessage(LOG_LEVEL_FATAL, Vulkan::INDEX,
                         "Failed to create 'VkDevice'");
          return false;
        } else {
          GAPID_WARNING(
              "[%u]replayCreateVkDevice called without a bound Vulkan renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayRegisterVkInstance,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayRegisterVkInstance()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayRegisterVkInstance(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayRegisterVkInstance called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayDestroyVkInstance,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayDestroyVkInstance()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayDestroyVkInstance(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayDestroyVkInstance called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayUnregisterVkInstance,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayUnregisterVkInstance()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayUnregisterVkInstance(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayUnregisterVkInstance called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayRegisterVkDevice,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayRegisterVkDevice()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayRegisterVkDevice(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayRegisterVkDevice called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayUnregisterVkDevice,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayUnregisterVkDevice()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayUnregisterVkDevice(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayUnregisterVkDevice called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayRegisterVkCommandBuffers,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayRegisterVkCommandBuffers()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayRegisterVkCommandBuffers(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayRegisterVkCommandBuffers called without a bound "
              "Vulkan renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayUnregisterVkCommandBuffers,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]replayUnregisterVkCommandBuffers()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayUnregisterVkCommandBuffers(stack);
        } else {
          GAPID_WARNING(
              "[%u]replayUnregisterVkCommandBuffers called without a bound "
              "Vulkan renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayCreateSwapchain,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayCreateSwapchain()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayCreateSwapchain(stack, pushReturn);
        } else {
          GAPID_WARNING(
              "[%u]replayCreateSwapchain called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayAllocateImageMemory,
      [this](uint32_t label, Stack* stack, bool push_return) {
        GAPID_DEBUG("[%u]replayAllocateImageMemory()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayAllocateImageMemory(stack, push_return);
        } else {
          GAPID_WARNING(
              "[%u]replayAllocateImageMemory called without a "
              "bound Vulkan renderer",
              label);
          return false;
        }
      });
  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayEnumeratePhysicalDevices,
      [this](uint32_t label, Stack* stack, bool push_return) {
        GAPID_DEBUG("[%u]replayEnumeratePhysicalDevices()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayEnumeratePhysicalDevices(stack, push_return);
        } else {
          GAPID_WARNING(
              "[%u]replayEnumeratePhysicalDevices called without a "
              "bound Vulkan renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayGetFenceStatus,
      [this](uint32_t label, Stack* stack, bool push_return) {
        GAPID_DEBUG("[%u]ReplayGetFenceStatus()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayGetFenceStatus(stack, push_return);
        } else {
          GAPID_WARNING(
              "ReplayGetFenceStatus called without a bound Vulkan renderer");
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayGetEventStatus,
      [this](uint32_t label, Stack* stack, bool push_return) {
        GAPID_DEBUG("[%u]ReplayGetEventStatus()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayGetEventStatus(stack, push_return);
        } else {
          GAPID_WARNING(
              "ReplayGetEventStatus called without a bound Vulkan renderer");
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayCreateVkDebugReportCallback,
      [this](uint32_t label, Stack* stack, bool push_return) {
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          auto* handle = stack->pop<Vulkan::VkDebugReportCallbackEXT*>();
          auto* create_info =
              stack->pop<Vulkan::VkDebugReportCallbackCreateInfoEXT*>();
          if (!stack->isValid()) {
            GAPID_ERROR(
                "Error during calling funtion "
                "ReplayCreateVkDebugReportCallback");
            return false;
          }

          // Populate the create info
          create_info->pfnCallback =
              reinterpret_cast<void*>(Vulkan::replayDebugReportCallback);
          create_info->pUserData = this;

          stack->push(create_info);
          stack->push(handle);
          if (api->replayCreateVkDebugReportCallback(stack, true)) {
            uint32_t result = stack->pop<uint32_t>();
            if (result == Vulkan::VkResult::VK_SUCCESS) {
              GAPID_INFO("GAPID Debug report callback created");
            } else {
              onDebugMessage(
                  LOG_LEVEL_WARNING, Vulkan::INDEX,
                  "Failed to create debug report callback, "
                  "VK_EXT_debug_report extenion may be not supported on "
                  "this replay device");
            }
            if (push_return) {
              stack->push(result);
            }
          }
          return true;
        } else {
          GAPID_WARNING(
              "ReplayCreateVkDebugReportCallback called without a bound "
              "Vulkan renderer");
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayDestroyVkDebugReportCallback,
      [this](uint32_t label, Stack* stack, bool) {
        GAPID_DEBUG("[%u]ReplayDestroyVkDebugReportCallback()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayDestroyVkDebugReportCallback(stack);
        } else {
          GAPID_WARNING(
              "ReplayDestroyVkDebugReportCallback called without a bound "
              "Vulkan renderer");
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayGetSemaphoreCounterValue,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayGetSemaphoreCounterValue()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayGetSemaphoreCounterValue(stack, pushReturn);
        } else {
          GAPID_WARNING(
              "[%u]replayGetSemaphoreCounterValue called without a bound "
              "Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayGetSemaphoreCounterValueKHR,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayGetSemaphoreCounterValueKHR()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayGetSemaphoreCounterValueKHR(stack, pushReturn);
        } else {
          GAPID_WARNING(
              "[%u]replayGetSemaphoreCounterValueKHR called without a bound "
              "Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayWaitSemaphores,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayWaitSemaphores()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayWaitSemaphores(stack, pushReturn);
        } else {
          GAPID_WARNING(
              "[%u]replayWaitSemaphores called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });

  interpreter->registerBuiltin(
      Vulkan::INDEX, Builtins::ReplayWaitSemaphoresKHR,
      [this](uint32_t label, Stack* stack, bool pushReturn) {
        GAPID_DEBUG("[%u]replayWaitSemaphoresKHR()", label);
        if (mVulkanRenderer != nullptr) {
          auto* api = mVulkanRenderer->getApi<Vulkan>();
          return api->replayWaitSemaphoresKHR(stack, pushReturn);
        } else {
          GAPID_WARNING(
              "[%u]replayWaitSemaphoresKHR called without a bound Vulkan "
              "renderer",
              label);
          return false;
        }
      });
}

bool Context::loadResource(Stack* stack) {
  uint32_t resourceId = stack->pop<uint32_t>();
  void* address = stack->pop<void*>();

  if (!stack->isValid()) {
    GAPID_WARNING("Error during loadResource");
    return false;
  }

  const auto& resource = mReplayRequest->getResources()[resourceId];

  if (!mResourceLoader->load(&resource, 1, address, resource.getSize())) {
    GAPID_WARNING("Can't load resource: %s", resource.getID().c_str());
    return false;
  }

  return true;
}

bool Context::postData(Stack* stack) {
  const uint32_t count = stack->pop<uint32_t>();
  const void* address = stack->pop<const void*>();

  if (!stack->isValid()) {
    GAPID_WARNING("Error during postData");
    return false;
  }

  return mPostBuffer->push(address, count);
}

bool Context::flushPostBuffer(Stack* stack) {
  if (!stack->isValid()) {
    GAPID_WARNING("Error during flushPostBuffer");
    return false;
  }

  return mPostBuffer->flush();
}

bool Context::startTimer(Stack* stack) {
  size_t index = static_cast<size_t>(stack->pop<uint8_t>());
  if (stack->isValid()) {
    if (index < MAX_TIMERS) {
      GAPID_INFO("startTimer(%zu)", index);
      mTimers[index].Start();
      return true;
    } else {
      GAPID_WARNING("StartTimer called with invalid index %zu", index);
    }
  } else {
    GAPID_WARNING("Error while calling function StartTimer");
  }
  return false;
}

bool Context::stopTimer(Stack* stack, bool pushReturn) {
  size_t index = static_cast<size_t>(stack->pop<uint8_t>());
  if (stack->isValid()) {
    if (index < MAX_TIMERS) {
      GAPID_INFO("stopTimer(%zu)", index);
      uint64_t ns = mTimers[index].Stop();
      if (pushReturn) {
        stack->push(ns);
      }
      return true;
    } else {
      GAPID_WARNING("StopTimer called with invalid index %zu", index);
    }
  } else {
    GAPID_WARNING("Error while calling function StopTimer");
  }
  return false;
}

bool Context::sendNotificationData(Stack* stack) {
  const uint32_t count = stack->pop<uint32_t>();
  const uint32_t id = stack->pop<uint32_t>();
  const void* address = stack->pop<const void*>();
  auto label = mInterpreter->getLabel();

  if (!stack->isValid()) {
    GAPID_WARNING("Stack is invalid during sendNotificationData");
    return false;
  }

  return mSrv->sendNotificationData(id, label, address, count);
}

bool Context::waitForFence(Stack* stack) {
  const uint32_t id = stack->pop<uint32_t>();
  if (!stack->isValid()) {
    GAPID_WARNING("Stack is invalid during waitForFence");
    return false;
  }
  auto fr = mSrv->getFenceReady(id);
  if (fr == nullptr) {
    GAPID_WARNING("FenceReady is invalid during waitForFence");
    return false;
  }
  if (fr->id() != id) {
    GAPID_WARNING("Fence ID is invalid during waitForFence");
    return false;
  }
  return true;
}

}  // namespace gapir
