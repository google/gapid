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

#ifndef REPLAY2_MEMORY_REMAPPER_ADDRESS_RANGE_H
#define REPLAY2_MEMORY_REMAPPER_ADDRESS_RANGE_H

#include "capture_address.h"
#include "replay_address.h"
#include "typesafe_address.h"

#include <type_traits>

namespace agi {
namespace replay2 {

template <class AddressType>
class AddressRange {
    static_assert(std::is_base_of<TypesafeAddress, AddressType>::value,
                  "Cannot instanciate AddressRange<T> for T that does not inherit off AddressRange.");

   public:
    AddressRange(AddressType address, size_t length) : baseAddress_(address), length_(length) {}

    const AddressType& baseAddress() const { return baseAddress_; }
    size_t length() const { return length_; }

    bool operator<(const AddressRange& rhs) const { return baseAddress_.bytePtr() < rhs.baseAddress().bytePtr(); }

   private:
    AddressType baseAddress_;
    size_t length_;
};

typedef AddressRange<CaptureAddress> CaptureAddressRange;
typedef AddressRange<ReplayAddress> ReplayAddressRange;

}  // namespace replay2
}  // namespace agi

#endif
