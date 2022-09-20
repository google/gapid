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

#pragma once
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <cassert>

#include "buffer.h"
#include "buffer_view.h"
#include "command_buffer.h"
#include "command_pool.h"
#include "common.h"
#include "descriptor_pool.h"
#include "descriptor_set.h"
#include "descriptor_set_layout.h"
#include "descriptor_update_template.h"
#include "device.h"
#include "device_memory.h"
#include "event.h"
#include "fence.h"
#include "framebuffer.h"
#include "handles.h"
#include "image.h"
#include "image_view.h"
#include "instance.h"
#include "physical_device.h"
#include "pipeline.h"
#include "pipeline_cache.h"
#include "pipeline_layout.h"
#include "query_pool.h"
#include "queue.h"
#include "render_pass.h"
#include "sampler.h"
#include "sampler_ycbcr_conversion.h"
#include "semaphore.h"
#include "shader_module.h"
#include "surface.h"
#include "swapchain.h"
#include "temporary_allocator.h"

namespace gapid2 {

}  // namespace gapid2