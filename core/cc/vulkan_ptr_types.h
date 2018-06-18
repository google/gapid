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

// Parts of this file are derived from vulkan_platform.h with this license.
/*
** Copyright (c) 2014-2015 The Khronos Group Inc.
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
*/

#ifndef CORE_VULKAN_EXTRAS_H_
#define CORE_VULKAN_EXTRAS_H_

#include "core/cc/target.h"

#if defined(TARGET_OS_WINDOWS)
// On Windows, Vulkan commands use the stdcall convention
#define VKAPI_ATTR
#define VKAPI_CALL __stdcall
#define VKAPI_PTR VKAPI_CALL
#elif defined(TARGET_OS_ANDROID) && defined(__ARM_ARCH_7A__)
// On Android/ARMv7a, Vulkan functions use the armeabi-v7a-hard calling
// convention, even if the application's native code is compiled with the
// armeabi-v7a calling convention.
#define VKAPI_ATTR __attribute__((pcs("aapcs-vfp")))
#define VKAPI_CALL
#define VKAPI_PTR VKAPI_ATTR
#else
// On other platforms, use the default calling convention
#define VKAPI_ATTR
#define VKAPI_CALL
#define VKAPI_PTR
#endif

#define VULKAN_API_ATTR VKAPI_ATTR
#define VULKAN_API_CALL VKAPI_CALL
#define VULKAN_API_PTR VKAPI_PTR

#endif
