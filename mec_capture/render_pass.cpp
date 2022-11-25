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

#include "render_pass.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_render_passes(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecRenderPasses");
  for (auto& it : state_block->VkRenderPasss) {
    auto rp = it.second.second;
    VkRenderPass render_pass = it.first;
    if (rp->get_create_info2_khr()) {
      serializer->vkCreateRenderPass2KHR(rp->device,
                                         rp->get_create_info2_khr(), nullptr, &render_pass);
    } else if (rp->get_create_info2()) {
      serializer->vkCreateRenderPass2(rp->device,
                                      rp->get_create_info2(), nullptr, &render_pass);
    } else {
      serializer->vkCreateRenderPass(rp->device,
                                     rp->get_create_info(), nullptr, &render_pass);
    }
  }
}

}  // namespace gapid2