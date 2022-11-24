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

#include "layer_base.h"
#include "layer_helper.h"
#include "layerer.h"
namespace gapid2 {
class passthrough_layer : public layer_base {
 public:
  passthrough_layer() : transform_base_(nullptr) {
    layer_base::initialize(&transform_base_);
    creation_tracker_ = std::make_unique<gapid2::transform<creation_tracker<VkCommandBuffer>>>(&transform_base_);
    layerer_ = std::make_unique<gapid2::transform<gapid2::layerer>>(&transform_base_);
    auto layers = gapid2::get_layers();
    auto user_config = gapid2::get_user_config();
    // dont inline these as they are order dependent
    layerer_->initializeLayers(layers, user_config);
  }

  gapid2::transform_base* get_top_level_functions() override {
    return &transform_base_;
  }

 private:
  std::unique_ptr<gapid2::transform<gapid2::creation_tracker<VkCommandBuffer>>> creation_tracker_;
  std::unique_ptr<gapid2::transform<gapid2::layerer>> layerer_;
  gapid2::transform<gapid2::transform_base> transform_base_;
};
}  // namespace gapid2