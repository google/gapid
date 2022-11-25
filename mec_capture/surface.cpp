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

#include "surface.h"

#include "mid_execution_generator.h"
#include "state_block.h"

namespace gapid2 {

void mid_execution_generator::capture_surfaces(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecSurfaces");
  for (auto& it : state_block->VkSurfaceKHRs) {
    auto surf = it.second.second;
    VkSurfaceKHR surface = it.first;

#if defined(VK_USE_PLATFORM_WIN32_KHR)
    serializer->vkCreateWin32SurfaceKHR(
        surf->instance,
        surf->get_create_info(),
        nullptr,
        &surface);
#elif defined(VK_USE_PLATFORM_XCB_KHR)
    serializer->vkCreateXcbSurfaceKHR(
        surf->instance,
        surf->get_create_info(),
        nullptr,
        &surface);
#endif
  }
}

}  // namespace gapid2