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

#include "command_serializer.h"
#include "enums.h"
namespace gapid2 {

class noop_serializer : public command_serializer {
 public:
  command_serializer* encoder;
  encoder_handle get_locked_encoder(uintptr_t key) override {
    return encoder->get_locked_encoder(key);
  }
  encoder_handle get_encoder(uintptr_t key) override {
    return encoder->get_encoder(key);
  }

  uint64_t get_flags() const override {
    return static_cast<uint64_t>(flags_);
  }

  void set_flags(flags flag) {
    flags_ = flag;
  }

 private:
  flags flags_ = flags::NONE;
};

}  // namespace gapid2