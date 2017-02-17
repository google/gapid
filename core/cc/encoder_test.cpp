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

#include "encoder.h"
#include "stream_writer.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <vector>

using ::testing::ElementsAre;
using ::testing::ElementsAreArray;

namespace core {
namespace test {

class BufferWriter : public StreamWriter {
public:
    inline BufferWriter(std::vector<uint8_t>* buffer) : mBuffer(buffer) {}

    virtual inline uint64_t write(const void* data, uint64_t size) override {
        const uint8_t* p = reinterpret_cast<const uint8_t*>(data);
        for (uint64_t i = 0; i < size; i++) {
            mBuffer->push_back(p[i]);
        }
        return size;
    }

private:
    std::vector<uint8_t>* mBuffer;
};

class EncoderTest : public ::testing::Test {
protected:
    virtual void SetUp() {
        auto writer = std::shared_ptr<StreamWriter>(new BufferWriter(&mBuffer));
        mEncoder = new Encoder(writer);
    }

    Encoder* mEncoder;
    std::vector<uint8_t> mBuffer;
};

TEST_F(EncoderTest, Bool) {
    mEncoder->Bool(true);
    mEncoder->Bool(false);
    EXPECT_THAT(mBuffer, ElementsAre(1, 0));
}

TEST_F(EncoderTest, Int8) {
    mEncoder->Int8(0);
    mEncoder->Int8(127);
    mEncoder->Int8(-128);
    mEncoder->Int8(-1);
    EXPECT_THAT(mBuffer, ElementsAre(0x00, 0x7f, 0x80, 0xff));
}

TEST_F(EncoderTest, Uint8) {
    mEncoder->Uint8(0x00);
    mEncoder->Uint8(0x7f);
    mEncoder->Uint8(0x80);
    mEncoder->Uint8(0xff);
    EXPECT_THAT(mBuffer, ElementsAre(0x00, 0x7f, 0x80, 0xff));
}

TEST_F(EncoderTest, Int16) {
    mEncoder->Int16(0);
    mEncoder->Int16(32767);
    mEncoder->Int16(-32768);
    mEncoder->Int16(-1);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xc0, 0xff, 0xfe,
        0xc0, 0xff, 0xff,
        0x01,
    })));
}

TEST_F(EncoderTest, Uint16) {
    mEncoder->Uint16(0);
    mEncoder->Uint16(0xbeef);
    mEncoder->Uint16(0xc0de);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xc0, 0xbe, 0xef,
        0xc0, 0xc0, 0xde,
    })));
}

TEST_F(EncoderTest, Int32) {
    mEncoder->Int32(0);
    mEncoder->Int32(2147483647);
    mEncoder->Int32(-2147483648);
    mEncoder->Int32(-1);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xf0, 0xff, 0xff, 0xff, 0xfe,
        0xf0, 0xff, 0xff, 0xff, 0xff,
        0x01,
    })));
}

TEST_F(EncoderTest, Uint32) {
    mEncoder->Uint32(0);
    mEncoder->Uint32(0x01234567);
    mEncoder->Uint32(0x10abcdef);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xe1, 0x23, 0x45, 0x67,
        0xf0, 0x10, 0xab, 0xcd, 0xef,
    })));
}

TEST_F(EncoderTest, Int64) {
    mEncoder->Int64(0);
    mEncoder->Int64(9223372036854775807LL);
    mEncoder->Int64(-9223372036854775808ULL);
    mEncoder->Int64(-1);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
        0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
        0x01,
    })));
}

TEST_F(EncoderTest, Uint64) {
    mEncoder->Uint64(0);
    mEncoder->Uint64(0x0123456789abcdefULL);
    mEncoder->Uint64(0xfedcba9876543210ULL);
    mEncoder->Uint64(0xffffffffULL);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xff, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
        0xff, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
        0xf0, 0xff, 0xff, 0xff, 0xff,
    })));
}

TEST_F(EncoderTest, Float32) {
    mEncoder->Float32(0);
    mEncoder->Float32(1);
    mEncoder->Float32(64.5);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xc0, 0x80, 0x3f,
        0xc0, 0x81, 0x42,
    })));
}

TEST_F(EncoderTest, Float64) {
    mEncoder->Float64(0);
    mEncoder->Float64(1);
    mEncoder->Float64(64.5);
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00,
        0xc0, 0xf0, 0x3f,
        0xe0, 0x20, 0x50, 0x40,
    })));
}

TEST_F(EncoderTest, Pointer) {
    mEncoder->Pointer(reinterpret_cast<void*>(0x00000000));
    mEncoder->Pointer(reinterpret_cast<void*>(0x01234567));
    mEncoder->Pointer(reinterpret_cast<void*>(0x10abcdef));
    mEncoder->Pointer(reinterpret_cast<void*>(0xffffffff));
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x00, 0x00,
        0xe1, 0x23, 0x45, 0x67, 0x00,
        0xf0, 0x10, 0xab, 0xcd, 0xef, 0x00,
        0xf0, 0xff, 0xff, 0xff, 0xff, 0x00,
    })));
}

TEST_F(EncoderTest, String) {
    mEncoder->String("Hello");
    mEncoder->String("");
    mEncoder->String("World");
    EXPECT_THAT(mBuffer, ElementsAreArray(std::vector<uint8_t>({
        0x05, 'H', 'e', 'l', 'l', 'o',
        0x00,
        0x05, 'W', 'o', 'r', 'l', 'd',
    })));
}

} // namespace test
}  // namespace core
