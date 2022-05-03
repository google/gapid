
def version():
    return 1.2


def exts(plat):
    e = [
        "VK_KHR_get_physical_device_properties2",
        "VK_KHR_multiview",
        "VK_KHR_maintenance2",
        "VK_AMD_memory_overallocation_behavior",
        "VK_KHR_swapchain",
        "VK_KHR_surface",
        "VK_KHR_image_format_list",
        "VK_AMD_shader_fragment_mask",
        "VK_EXT_4444_formats",
        "VK_EXT_conservative_rasterization",
        "VK_EXT_custom_border_color",
        "VK_EXT_depth_clip_enable",
        "VK_EXT_extended_dynamic_state",
        "VK_EXT_host_query_reset",
        "VK_EXT_memory_budget",
        "VK_EXT_memory_priority",
        "VK_EXT_robustness2",
        "VK_EXT_shader_demote_to_helper_invocation",
        "VK_EXT_shader_stencil_export",
        "VK_EXT_shader_viewport_index_layer",
        "VK_EXT_transform_feedback",
        "VK_EXT_vertex_attribute_divisor",
        "VK_KHR_buffer_device_address",
        "VK_KHR_create_renderpass2",
        "VK_KHR_depth_stencil_resolve",
        "VK_KHR_draw_indirect_count",
        "VK_KHR_driver_properties",
        "VK_KHR_image_format_list",
        "VK_KHR_sampler_mirror_clamp_to_edge",
        "VK_KHR_shader_float_controls",
        "VK_KHR_swapchain",
        "VK_KHR_get_surface_capabilities2",

        # MORE!
        "VK_KHR_maintenance3",
        "VK_KHR_storage_buffer_storage_class",
        "VK_KHR_shader_draw_parameters",
        "VK_KHR_16bit_storage",
        "VK_KHR_shader_atomic_int64",
        "VK_KHR_shader_float16_int8",
        "VK_KHR_timeline_semaphore",
        "VK_EXT_depth_range_unrestricted",
        "VK_EXT_descriptor_indexing",
        "VK_EXT_fragment_shader_interlock",
        "VK_EXT_shader_image_atomic_int64",
        "VK_EXT_scalar_block_layout",
        "VK_AMD_shader_core_properties",
        "VK_EXT_sample_locations",
        "VK_NV_shader_sm_builtins",

        #Helpers
        "VK_KHR_external_memory_capabilities",
        "VK_KHR_external_memory",
        "VK_EXT_external_memory_host"
    ]
    if plat.system() == "Windows":
        e.extend(
            ["VK_EXT_full_screen_exclusive",
             "VK_KHR_win32_surface",
             ])
        print("Platform detected: Win32")
    elif plat.system() == "Linux":
        e.extend(["VK_KHR_xcb_surface"])
        print("Platform detected: X11/XCB")
    return e


COPYRIGHT = '''
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
 '''

HEADER = f'''#pragma once
{COPYRIGHT}

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>
'''

CPPHEADER = f'''{COPYRIGHT}

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>
'''
