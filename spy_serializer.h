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

#include <fstream>
#include <mutex>
#include <thread>

#include "command_serializer.h"
namespace gapid2 {
class spy_serializer : public command_serializer {
 public:
  using super = transform_base;
  spy_serializer() : out_file("D:\\src\\file.trace", std::ios::out | std::ios::binary),
                     enabled_(false) {
    encoder_tls_key = TlsAlloc();
  }

  encoder_handle get_locked_encoder(uintptr_t key) override;
  encoder_handle get_encoder(uintptr_t key) override;
  void enable_with_mec();
  void enable();
  void disable();

 private:
  std::recursive_mutex call_mutex;
  DWORD encoder_tls_key;
  std::fstream out_file;
  std::atomic<bool> enabled_;
  std::atomic<std::thread::id> tid_ = std::thread::id();
};
}  // namespace gapid2