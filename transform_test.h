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

#include "transform_base.h"

namespace gapid2 {
class transform_test : public transform_base {};
class transform_test2 : public transform_base {
 public:
  VkResult vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkInstance* pInstance) override final {
    return transform_base::vkCreateInstance(pCreateInfo, pAllocator, pInstance);
  }
};

}  // namespace gapid2