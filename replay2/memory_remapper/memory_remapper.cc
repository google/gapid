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

#include "memory_remapper.h"

namespace agi {
namespace replay2 {

	void MarkDeadAddressRange(const ReplayAddressRange& replayAddressRange) {
		const std::byte dead[2] = { std::byte(0xDE), std::byte(0xAD) };
		for(int i = 0; i < replayAddressRange.length(); ++i) {
			replayAddressRange.baseAddress().bytePtr()[i] = dead[i %2];
		}
	}

	ReplayAddress MemoryRemapper::AddMapping(const MemoryObservation& observation) {

		const CaptureAddress& captureAddress = observation.captureAddress();

		auto replayAddressRangeAndOffset = findReplayAddressRangeAndOffset(captureAddress);
		if(replayAddressRangeAndOffset.first != nullptr) {
			throw AddressAlreadyMappedException();
		}

		const size_t resourceLength = observation.resourceGenerator()->length();
		if(resourceLength == 0) {
			throw CannotMapZeroLengthAddressRange();
		}

		std::byte* replayAllocation = new std::byte [resourceLength];
		const ReplayAddress replayAddress(replayAllocation);

		observation.resourceGenerator()->generate(replayAddress);

		const CaptureAddressRange captureAddressRange(captureAddress, resourceLength);
		const ReplayAddressRange replayAddressRange(replayAddress, resourceLength);

		captureAddressRanges_.emplace(captureAddressRange, replayAddressRange);

		return replayAddress;
	}

	void MemoryRemapper::RemoveMapping(const CaptureAddress& captureAddress) {

		auto iter = captureAddressRanges_.upper_bound(CaptureAddressRange(captureAddress, 0));
		if(iter == captureAddressRanges_.begin()) {
			throw AddressNotMappedException();
		}

		--iter;
		const CaptureAddressRange& captureAddressRange = iter->first;
		const ReplayAddressRange& replayAddressRange = iter->second;

		const intptr_t offset = captureAddress.bytePtr() -captureAddressRange.baseAddress().bytePtr();

		if(offset >= captureAddressRange.length()) {
			throw AddressNotMappedException();
		}

		if(offset > 0) {
			throw RemoveMappingOffsetAddressException();
		}

#ifndef NDEBUG
		// In debug we'll splat all released memory with 0xDEAD before releasing it to help with debugging.
		MarkDeadAddressRange(replayAddressRange);
#endif

		delete [] replayAddressRange.baseAddress().bytePtr();
		captureAddressRanges_.erase(iter);
	}

	ReplayAddress MemoryRemapper::RemapCaptureAddress(const CaptureAddress& captureAddress) const {

		auto replayAddressRangeAndOffset = findReplayAddressRangeAndOffset(captureAddress);

		const ReplayAddressRange *replayAddressRange = replayAddressRangeAndOffset.first;
		const intptr_t offset = replayAddressRangeAndOffset.second;

		if(replayAddressRange == nullptr) {
			throw AddressNotMappedException();
		}

		const ReplayAddress replayAddress(replayAddressRange->baseAddress().bytePtr() +offset);
		return replayAddress;
	}

	const std::pair<const ReplayAddressRange*, intptr_t>
	MemoryRemapper::findReplayAddressRangeAndOffset(const CaptureAddress& captureAddress) const {

		// Get an iterator to the first mapped address range after the address we're interested in.
		auto iter = captureAddressRanges_.upper_bound(CaptureAddressRange(captureAddress, 0));
		if(iter == captureAddressRanges_.begin()) {
			// If we got an iterator to the start, then the address is in unmapped memory.
			return std::pair<const ReplayAddressRange*, intptr_t>(nullptr, 0);
		}

		// Walk the iterator back 1 place, so we have the last address range starting before
		// the address we're interested in.
		--iter;
		const CaptureAddressRange& captureAddressRange = iter->first;
		const ReplayAddressRange& replayAddressRange = iter->second;

		// Compute the offset from the start of the address range to the address we're remapping
		// and use this to do a bounds check to see if we're inside the mapped range.
		const intptr_t offset = captureAddress.bytePtr() -captureAddressRange.baseAddress().bytePtr();
		if(offset >= captureAddressRange.length()) {
			return std::pair<const ReplayAddressRange*, intptr_t>(nullptr, 0);
		}

		return std::make_pair(&replayAddressRange, offset);
	}

}  // namespace replay2
}  // namespace agi
