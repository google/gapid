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

#ifndef REPLAY2_MEMORY_REMAPPER_TYPESAFE_ADDRESS_H
#define REPLAY2_MEMORY_REMAPPER_TYPESAFE_ADDRESS_H

#include <cstddef>

namespace agi {
namespace replay2 {

class TypesafeAddress {
   public:
    explicit TypesafeAddress(std::byte* address) : address_(address) {}
    std::byte* bytePtr() const { return address_; }

    bool operator==(const TypesafeAddress& rhs) const { return address_ == rhs.address_; }
    bool operator!=(const TypesafeAddress& rhs) const { return !(*this == rhs); }

   protected:
    std::byte* address_;
};

}  // namespace replay2
}  // namespace agi

#endif
