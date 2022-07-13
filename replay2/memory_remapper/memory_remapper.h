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

#ifndef REPLAY2_MEMORY_REMAPPER_MEMORY_REMAPPER_H
#define REPLAY2_MEMORY_REMAPPER_MEMORY_REMAPPER_H

#include "address_range.h"
#include "capture_address.h"
#include "memory_observation.h"
#include "replay_address.h"

#include <assert.h>
#include <map>

#include "replay2/core_utils/non_copyable.h"

namespace agi {
namespace replay2 {

class MemoryRemapper : public NonCopyable {
   public:
    ReplayAddress AddMapping(const MemoryObservation& observation);
    void RemoveMapping(const CaptureAddress& captureAddress);

    ReplayAddress RemapCaptureAddress(const CaptureAddress& captureAddress) const;

    class AddressNotMappedException : public std::exception {};
    class AddressAlreadyMappedException : public std::exception {};
    class RemoveMappingOffsetAddressException : public std::exception {};
    class CannotMapZeroLengthAddressRange : public std::exception {};

   private:
    const std::pair<const ReplayAddressRange*, intptr_t> findReplayAddressRangeAndOffset(
        const CaptureAddress& captureAddress) const;

    std::map<CaptureAddressRange, ReplayAddressRange> captureAddressRanges_;
};

}  // namespace replay2
}  // namespace agi

#endif
