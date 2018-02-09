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

// This file is intended to be included by vulkan_spy.h inside
// of the gapid namespace.

inline void VulkanSpy::notifyPendingCommandAdded(CallObserver*, VkQueue) {}

template<typename VkErrorType>
inline void VulkanSpy::onVkError(CallObserver* observer, VkErrorType err) {
  GAPID_WARNING("Unhandled Vulkan error");
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_INVALID_HANDLE> err) {
  GAPID_WARNING("Error: Invalid %s: %llu", err->mhandleType.c_str(), err->mhandle)
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_NULL_POINTER> err) {
  GAPID_WARNING("Error: Null pointer of %s", err->mpointerType.c_str())
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_UNRECOGNIZED_EXTENSION> err) {
  GAPID_WARNING("Error: Unrecognized extension: %s", err->mname.c_str())
}
