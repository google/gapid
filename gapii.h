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

#include "command_buffer_invalidator.h"
#include "command_buffer_recorder.h"
#include "creation_tracker.h"
#include "layer_base.h"
#include "mec_controller.h"
#include "spy.h"
#include "spy_serializer.h"
#include "state_tracker.h"

namespace gapid2 {
class gapii : public layer_base {
 public:
  gapii() : transform_base_(nullptr) {
    layer_base::initialize(&transform_base_);
    serializer_ = std::make_unique<gapid2::transform<gapid2::spy_serializer>>(&transform_base_);
    spy_ = std::make_unique<gapid2::transform<gapid2::spy>>(&transform_base_);
#if defined(MEC)
    state_block_ = std::make_unique<gapid2::transform<gapid2::state_block>>(&transform_base_);
    creation_tracker_ = std::make_unique<gapid2::transform<gapid2::minimal_creation_state_tracker>>(&transform_base_);
    command_buffer_recorder_ = std::make_unique<gapid2::transform<gapid2::command_buffer_recorder>>(&transform_base_);
    command_buffer_invalidator_ = std::make_unique<gapid2::transform<gapid2::command_buffer_invalidator>>(&transform_base_);
    state_tracker_ = std::make_unique<gapid2::transform<gapid2::state_tracker>>(&transform_base_);
    mec_controller_ = std::make_unique<gapid2::transform<gapid2::mec_controller>>(&transform_base_);
    spy_->initialize(serializer_.get(), minimal_state_tracker_.get());
    mec_controller_->initialize(serializer_.get(), minimal_state_tracker_.get(), spy_.get(), command_buffer_recorder_.get());
    //serializer_->enable();
#else
    spy_->initialize(serializer_.get(), minimal_state_tracker_.get());
    serializer_->enable();
#endif
  }

  ~gapii() {
  }

  gapid2::transform_base* get_top_level_functions() override {
    return &transform_base_;
  }

 private:
  std::unique_ptr<gapid2::transform<gapid2::state_tracker>> state_tracker_;
  std::unique_ptr<gapid2::transform<gapid2::command_buffer_invalidator>> command_buffer_invalidator_;
  std::unique_ptr<gapid2::transform<gapid2::command_buffer_recorder>> command_buffer_recorder_;
  std::unique_ptr<gapid2::transform<gapid2::state_block>> state_block_;
  std::unique_ptr<gapid2::transform<gapid2::minimal_creation_state_tracker>> creation_tracker_;
  std::unique_ptr<gapid2::transform<gapid2::spy_serializer>> serializer_;
  std::unique_ptr<gapid2::transform<gapid2::spy>> spy_;
  std::unique_ptr<gapid2::transform<gapid2::mec_controller>> mec_controller_;
  gapid2::transform<gapid2::transform_base> transform_base_;
};
}  // namespace gapid2
