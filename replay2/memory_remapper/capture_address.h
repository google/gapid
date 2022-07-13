// Copyright (C) 2022 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef REPLAY2_MEMORY_REMAPPER_CAPTURE_ADDRESS_H
#define REPLAY2_MEMORY_REMAPPER_CAPTURE_ADDRESS_H

#include "typesafe_address.h"

#include <cstddef>

namespace agi {
namespace replay2 {

class CaptureAddress : public TypesafeAddress {
   public:
    explicit CaptureAddress(std::byte* address) : TypesafeAddress(address) {}
    CaptureAddress offsetByBytes(ptrdiff_t offset) const { return CaptureAddress(address_ + offset); }
};

}  // namespace replay2
}  // namespace agi

#endif
