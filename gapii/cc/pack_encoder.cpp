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


#include "pack_encoder.h"
#include "core/data/pack/pack.pb.h"

#include <core/cc/stream_writer.h>

#include <google/protobuf/descriptor.pb.h>
#include <google/protobuf/io/coded_stream.h>
#include <google/protobuf/message.h>

using ::google::protobuf::Message;
using ::google::protobuf::Descriptor;
using ::google::protobuf::DescriptorProto;
using ::google::protobuf::io::CodedOutputStream;
using ::google::protobuf::io::StringOutputStream;

namespace {

const uint64_t specialSection = 0;
const char magic[] = "protopack";

// PackEncoderImpl implements the PackEncoder interface.
class PackEncoderImpl : public gapii::PackEncoder {
public:
    PackEncoderImpl(std::shared_ptr<core::StreamWriter> output);

    void message(const ::google::protobuf::Message* msg) override;

private:
    void writeType(const ::google::protobuf::Descriptor* desc);
    void writeSection(uint64_t tag, const std::string& name, const ::google::protobuf::Message* msg);
    void flushChunk();
    void writeString(const std::string& str);
    void writeVarint32(uint32_t value);
    void writeVarint(uint64_t value);
    void writeVarintDirect(uint64_t value);

    std::unordered_map<const ::google::protobuf::Descriptor*, uint32_t> mIds;

    std::string mBuffer; // Flushes to mWriter
    std::shared_ptr<core::StreamWriter> mWriter;
};

PackEncoderImpl::PackEncoderImpl(std::shared_ptr<core::StreamWriter> writer)
        : mWriter(writer) {
    writer->write(magic, sizeof(magic) - 1);
    pack::Header header;
    header.mutable_version()->set_major(1);

    header.SerializeToString(&mBuffer);
    flushChunk();
}

void PackEncoderImpl::message(const Message* msg) {
    auto desc = msg->GetDescriptor();

    auto insert = mIds.insert(std::make_pair(desc, mIds.size() + 1));
    if (insert.second) {
        writeType(desc);
    }
    writeSection(insert.first->second, "", msg);
}

void PackEncoderImpl::writeType(const Descriptor* desc) {
    DescriptorProto msg;
    desc->CopyTo(&msg);
    writeSection(specialSection, desc->full_name(), &msg);
}

void PackEncoderImpl::writeSection(uint64_t tag, const std::string& name, const Message* msg) {
    writeVarint(tag);
    if (name.size() != 0) {
        writeString(name);
    }
    msg->AppendToString(&mBuffer);
    flushChunk();
}

void PackEncoderImpl::flushChunk() {
    writeVarintDirect(mBuffer.size());
    mWriter->write(mBuffer.data(), mBuffer.size());
    mBuffer.clear();
}

void PackEncoderImpl::writeString(const std::string& str) {
    writeVarint(str.length());
    mBuffer.append(str);
}

void PackEncoderImpl::writeVarint32(uint32_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint32ToArray(value, &buf[0]) - &buf[0];
    mBuffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

void PackEncoderImpl::writeVarint(uint64_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
    mBuffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

void PackEncoderImpl::writeVarintDirect(uint64_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
    mWriter->write(&buf[0], count);
}

// PackEncoderNoop is a no-op implementation of the PackEncoder interface.
class PackEncoderNoop : public gapii::PackEncoder {
public:
    void message(const ::google::protobuf::Message* msg) override {}
};

} // anonymous namespace

namespace gapii {

// create returns a PackEncoder::SPtr that writes to output.
PackEncoder::SPtr PackEncoder::create(std::shared_ptr<core::StreamWriter> output) {
    return PackEncoder::SPtr(new PackEncoderImpl(output));
}

// noop returns a PackEncoder::SPtr that does nothing.
PackEncoder::SPtr PackEncoder::noop() {
    return PackEncoder::SPtr(new PackEncoderNoop());
}

} // namespace gapii
