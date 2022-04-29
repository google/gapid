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

#include "base_caller.h"
#include "minimal_state_tracker.h"
#include "state_block.h"
#include "transform.h"

namespace gapid2 {
class layer_base {
 public:
  void initialize(transform_base* next) {
    base_caller_ = std::make_unique<gapid2::transform<gapid2::base_caller>>(next);
    state_block_ = std::make_unique<gapid2::transform<gapid2::state_block>>(next);
    minimal_state_tracker_ = std::make_unique<gapid2::transform<gapid2::minimal_state_tracker>>(next);
  }

  void set_nexts(PFN_vkCreateInstance create_instance, PFN_vkGetInstanceProcAddr get_instance_proc_addr) {
    base_caller_->vkCreateInstance_ = create_instance;
    base_caller_->vkGetInstanceProcAddr_ = get_instance_proc_addr;
  }

  virtual gapid2::transform_base* get_top_level_functions() = 0;

  std::unique_ptr<gapid2::transform<gapid2::base_caller>> base_caller_;
  std::unique_ptr<gapid2::transform<gapid2::state_block>> state_block_;
  std::unique_ptr<gapid2::transform<gapid2::minimal_state_tracker>> minimal_state_tracker_;

 private:
};
}  // namespace gapid2