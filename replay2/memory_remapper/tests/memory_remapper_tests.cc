// Copyright (C) 2022 Google Inc.
//
// This file is generated code. It was created by the AGI code generator.
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

#include <gtest/gtest.h>
#include "replay2/memory_remapper/memory_remapper.h"

using namespace agi::replay2;

class ModResourceGenerator : public ResourceGenerator {
public:
	ModResourceGenerator(size_t length) : length_(length) {}
	virtual ~ModResourceGenerator() {}

	size_t length() const override { return length_; }

	void generate(ReplayAddress replayAddress) override {
		for(size_t i = 0; i < length_; ++i) {
			replayAddress.bytePtr()[i] = std::byte(i % 256);
		}
	}
private:
	size_t length_;
};

class ConstResourceGenerator : public ResourceGenerator {
public:
	ConstResourceGenerator(std::byte value, size_t length) : value_(value), length_(length) {}
	virtual ~ConstResourceGenerator() {}

	size_t length() const override { return length_; }

	void generate(ReplayAddress replayAddress) override {
		for(size_t i = 0; i < length_; ++i) {
			replayAddress.bytePtr()[i] = value_;
		}
	}
private:
	std::byte value_;
	size_t length_;
};

void ASSERT_MOD_REPLAY_ADDRESS(const MemoryRemapper& remapper, const CaptureAddress& captureAddress, const ReplayAddress& replayAddress, size_t length) {
	for(size_t i = 0; i < length; ++i) {
		ReplayAddress replayAddress = remapper.RemapCaptureAddress(captureAddress.offsetByBytes(i));
		ASSERT_EQ(replayAddress.bytePtr()[0], std::byte(i % 256));
	}
}

void ASSERT_CONST_REPLAY_ADDRESS(const MemoryRemapper& remapper, const CaptureAddress& captureAddress, const ReplayAddress& replayAddress, std::byte value, size_t length) {
	for(size_t i = 0; i < length; ++i) {
		ReplayAddress replayAddress = remapper.RemapCaptureAddress(captureAddress.offsetByBytes(i));
		ASSERT_EQ(replayAddress.bytePtr()[0], value);
	}
}

TEST(MemoryRemapperTests, SimpleMapping) {

	const size_t size = 128;

	std::byte* rawCapturePtr = new std::byte[size];
	CaptureAddress captureAddress(rawCapturePtr);

	MemoryRemapper remapper;
	const MemoryObservation captureObservation(captureAddress, std::make_shared<ModResourceGenerator>(size));

	const ReplayAddress replayAddress = remapper.AddMapping(captureObservation);
	EXPECT_NE(replayAddress.bytePtr(), nullptr);
	ASSERT_MOD_REPLAY_ADDRESS(remapper, captureAddress, replayAddress, size);

	const ReplayAddress replayAddress2 = remapper.RemapCaptureAddress(captureAddress);
	EXPECT_EQ(replayAddress, replayAddress2);

	EXPECT_NO_THROW(remapper.RemoveMapping(captureAddress));
	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddress), MemoryRemapper::AddressNotMappedException);
}

TEST(MemoryRemapperTests, UnknownMapping) {

	const size_t size = 128;

	std::byte* rawCapturePtr = new std::byte[size];
	CaptureAddress captureAddress(rawCapturePtr);

	MemoryRemapper remapper;

	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddress), MemoryRemapper::AddressNotMappedException);

	EXPECT_THROW(remapper.RemoveMapping(captureAddress), MemoryRemapper::AddressNotMappedException);
	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddress), MemoryRemapper::AddressNotMappedException);
}

TEST(MemoryRemapperTests, ZeroLengthMapping) {

	const size_t size = 0;

	std::byte* rawCapturePtr = new std::byte[size];
	CaptureAddress captureAddress(rawCapturePtr);

	MemoryRemapper remapper;
	const MemoryObservation captureObservation(captureAddress, std::make_shared<ModResourceGenerator>(size));

	EXPECT_THROW(remapper.AddMapping(captureObservation), MemoryRemapper::CannotMapZeroLengthAddressRange);
	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddress), MemoryRemapper::AddressNotMappedException);
	EXPECT_THROW(remapper.RemoveMapping(captureAddress), MemoryRemapper::AddressNotMappedException);
}

TEST(MemoryRemapperTests, MultipleMappings) {

	std::vector<ReplayAddress> replayAddresses;
	std::vector<CaptureAddress> captureAddresses;

	MemoryRemapper remapper;

	for(int i = 0; i < 64; ++i) {

		const size_t size = (i +1) * 2;

		std::byte* rawCapturePtr = new std::byte[size];
		CaptureAddress captureAddress(rawCapturePtr);
		captureAddresses.push_back(captureAddress);

		const MemoryObservation captureObservation(captureAddress, std::make_shared<ConstResourceGenerator>(std::byte(i), size));

		const ReplayAddress replayAddress = remapper.AddMapping(captureObservation);
		replayAddresses.push_back(replayAddress);

		const ReplayAddress replayAddress2 = remapper.RemapCaptureAddress(captureAddress);
		EXPECT_EQ(replayAddress, replayAddress2);
	}

	for(int i = 0; i < 64; ++i) {
		const size_t size = i * 2;
		EXPECT_NE(replayAddresses[i].bytePtr(), nullptr);
		ASSERT_CONST_REPLAY_ADDRESS(remapper, captureAddresses[i], replayAddresses[i], std::byte(i), size);
		EXPECT_NO_THROW(remapper.RemoveMapping(captureAddresses[i]));
		EXPECT_THROW(remapper.RemapCaptureAddress(captureAddresses[i]), MemoryRemapper::AddressNotMappedException);
	}
}

TEST(MemoryRemapperTests, MappingCollision) {

	const size_t offset = 31;

	const size_t sizeA = 128;
	std::byte* rawCapturePtrA = new std::byte[sizeA];
	CaptureAddress captureAddressA(rawCapturePtrA);

	const size_t sizeB = sizeA -offset;
	std::byte* rawCapturePtrB = rawCapturePtrA +offset;
	CaptureAddress captureAddressB(rawCapturePtrB);

	MemoryRemapper remapper;
	const MemoryObservation captureObservationA(captureAddressA, std::make_shared<ConstResourceGenerator>(std::byte(0), sizeA));
	const MemoryObservation captureObservationB(captureAddressB, std::make_shared<ConstResourceGenerator>(std::byte(1), sizeB));

	ReplayAddress replayAddressA = remapper.AddMapping(captureObservationA);
	EXPECT_THROW(remapper.AddMapping(captureObservationB), MemoryRemapper::AddressAlreadyMappedException);

	EXPECT_NE(replayAddressA.bytePtr(), nullptr);
	ASSERT_CONST_REPLAY_ADDRESS(remapper, captureAddressA, replayAddressA, std::byte(0), sizeA);
	EXPECT_NO_THROW(remapper.RemoveMapping(captureAddressA));

	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddressA), MemoryRemapper::AddressNotMappedException);
	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddressB), MemoryRemapper::AddressNotMappedException);
}

TEST(MemoryRemapperTests, RemoveMappingOffsetAddressException) {

	const size_t size = 128;

	std::byte* rawCapturePtr = new std::byte[size];
	CaptureAddress captureAddress(rawCapturePtr);

	MemoryRemapper remapper;
	const MemoryObservation captureObservation(captureAddress, std::make_shared<ModResourceGenerator>(size));

	const ReplayAddress replayAddress = remapper.AddMapping(captureObservation);
	EXPECT_NE(replayAddress.bytePtr(), nullptr);
	ASSERT_MOD_REPLAY_ADDRESS(remapper, captureAddress, replayAddress, size);

	CaptureAddress offsetCaptureAddress(captureAddress.bytePtr() +13);
	EXPECT_THROW(remapper.RemoveMapping(offsetCaptureAddress), MemoryRemapper::RemoveMappingOffsetAddressException);
	EXPECT_NO_THROW(remapper.RemapCaptureAddress(captureAddress));

	EXPECT_NO_THROW(remapper.RemoveMapping(captureAddress));
	EXPECT_THROW(remapper.RemapCaptureAddress(captureAddress), MemoryRemapper::AddressNotMappedException);
}

int main(int argc, char **argv) {
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
