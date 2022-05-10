#pragma once

/*
 * Copyright (C) 2022 Google Inc.
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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "noop_serializer.h"
#include "null_caller.h"
#include "spy.h"
#include "spy_serializer.h"
#include "transform.h"

namespace gapid2 {
class mec_controller : public transform_base {
 public:
  using super = transform_base;
  mec_controller() : null_caller_(&empty_), noop_serializer(&empty_) {}

  void initialize(spy_serializer* spy_serializer,
                  transform_base* passthrough_caller,
                  spy* spy) {
    spy_ = spy;
    spy_serializer_ = spy_serializer;
    noop_serializer.encoder = spy_serializer_;
    noop_serializer.state_block_ = state_block_;
    passthrough_caller_ = passthrough_caller;
  }

  VkResult vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
    auto ret = transform_base::vkQueuePresentKHR(queue, pPresentInfo);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    if (capture_frame > 0 && --capture_frame == 0) {
      start_capture();
    } else if (capture_frame == 0 && frames_to_capture > 0 && --frames_to_capture == 0) {
      end_capture();
    }
    return ret;
  }

  void start_capture();
  void end_capture();

  spy_serializer* spy_serializer_;
  transform_base* passthrough_caller_;
  spy* spy_;
  size_t capture_frame = 100;
  size_t frames_to_capture = 100;
  transform_base empty_;
  transform<null_caller> null_caller_;
  transform<noop_serializer> noop_serializer;
};
}  // namespace gapid2