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

#ifndef CORE_GET_GLES_PROC_ADDRESS_H
#define CORE_GET_GLES_PROC_ADDRESS_H

namespace core {

typedef void*(GetGlesProcAddressFunc)(const char* name);

// GetGlesProcAddress returns the GLES function pointer to the function with
// the given name, or nullptr if the function was not found.
extern GetGlesProcAddressFunc* GetGlesProcAddress;

bool hasGLorGLES();

}  // namespace core

#endif  // CORE_GET_GLES_PROC_ADDRESS_H
