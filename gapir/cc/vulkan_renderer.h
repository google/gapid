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

#ifndef GAPIR_VULKAN_RENDERER_H
#define GAPIR_VULKAN_RENDERER_H

#include "renderer.h"

namespace gapir {

// The Vulkan renderer implementation.
// TODO(antiagainst): Right now this is just a skeleton. Flesh out it.
class VulkanRenderer : public Renderer {
 public:
  // Construct and return an offscreen renderer.
  static VulkanRenderer* create();

  // Returns the renderer's API.
  virtual Api* api() = 0;

  // Return true if this is a valid api for this system.
  virtual bool isValid() = 0;
};

}  // namespace gapir

#endif  // GAPIR_VULKAN_RENDERER_H
