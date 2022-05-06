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

#include "creation_tracker.h"
#include "layer_base.h"
#include "spy.h"
#include "state_tracker.h"

namespace gapid2 {
class gapii : public layer_base {
 public:
  gapii() : transform_base_(nullptr) {
    layer_base::initialize(&transform_base_);
    spy_ = std::make_unique<gapid2::transform<gapid2::spy>>(&transform_base_);
#ifdef MEC
    state_block_ = std::make_unique<gapid2::transform<gapid2::state_block>>(&transform_base_);
    state_tracker_ = std::make_unique<gapid2::transform<gapid2::state_tracker>>(&transform_base_);
#endif

    spy_->initialize();
  }

  ~gapii() {
  }

  gapid2::transform_base* get_top_level_functions() override {
    return &transform_base_;
  }

 private:
  std::unique_ptr<gapid2::transform<gapid2::state_tracker>> state_tracker_;
  std::unique_ptr<gapid2::transform<gapid2::state_block>> state_block_;

  std::unique_ptr<gapid2::transform<gapid2::spy>> spy_;
  gapid2::transform<gapid2::transform_base> transform_base_;
};
}  // namespace gapid2
