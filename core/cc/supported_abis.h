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

#ifndef CORE_SUPPORTED_ABIS_H
#define CORE_SUPPORTED_ABIS_H

namespace core {

// supportedABIs returns a whitespace delimited set of ABIs supported by this
// host machine, starting with the preferred ABI.
// Must match the definitions in core/os/device/abi.go.
inline const char* supportedABIs() {
#ifdef __x86_64
  return "x86-64";
#elif defined __i386
  return "x86";
#elif defined __ARM_ARCH_7A__
  return "armeabi-v7a";
#elif defined __aarch64__
  return "arm64-v8a armeabi-v7a";
#else
#error "Unrecognised target architecture"
#endif
}

}  // namespace core

#endif  // CORE_SUPPORTED_ABIS_H
