/*
 * Copyright (C) 2019 Google Inc.
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
 *
 */

#include "core/cc/target.h"
#include "core/cc/log.h"
#include <vulkan/vk_layer.h>

#include <cstring>

extern "C"
{
    __attribute__((constructor)) void _layer_dummy_func__();
}
#if (TARGET_OS == GAPID_OS_WINDOWS) || (TARGET_OS == GAPID_OS_OSX)
class dummy_struct
{
};
#else
#include <dlfcn.h>
#include <cstdio>
#include <cstdint>
class dummy_struct
{
public:
    dummy_struct();
};

dummy_struct::dummy_struct()
{
    GAPID_ERROR("Loading dummy struct");
    Dl_info info;
    if (dladdr((void *)&_layer_dummy_func__, &info))
    {
        dlopen(info.dli_fname, RTLD_NODELETE);
    }
}
#endif

extern "C"
{
    void _layer_dummy_func__()
    {
        dummy_struct d;
        (void)d;
    }
}
namespace api_timing
{
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceExtensionProperties(PFN_vkEnumerateInstanceExtensionProperties, const char *pLayerName, uint32_t *pPropertyCount,
                                       VkExtensionProperties *pProperties)
{
    if (!pProperties)
    {
        *pPropertyCount = 1;
        return VK_SUCCESS;
    }
    if (*pPropertyCount < 1)
    {
        return VK_INCOMPLETE;
    }
    strcpy(pProperties[0].extensionName, "GAPID_Enabled");
    return VK_SUCCESS;
}
} // namespace api_timing