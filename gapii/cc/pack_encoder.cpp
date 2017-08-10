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

#include <mutex>

using ::google::protobuf::Message;
using ::google::protobuf::Descriptor;
using ::google::protobuf::DescriptorProto;
using ::google::protobuf::io::CodedOutputStream;
using ::google::protobuf::io::StringOutputStream;

namespace {

const int      VERSION_MAJOR       = 1;
const int      VERSION_MINOR       = 1;
const int64_t  TAG_FIRST_GROUP     = -1;
const int64_t  TAG_GROUP_FINALIZER = 0;
const int64_t  TAG_DECLARE_TYPE    = 1;
const int64_t  TAG_FIRST_OBJECT    = 2;
const uint64_t NO_ID               = 0xffffffffffffffff;

const char magic[] = "protopack";

// PackEncoderImpl implements the PackEncoder interface.
class PackEncoderImpl : public gapii::PackEncoder {
public:
    PackEncoderImpl(const std::shared_ptr<core::StreamWriter>& output);
    ~PackEncoderImpl();

    virtual void object(const Message* msg) override;
    virtual SPtr group(const Message* msg) override;

private:
    struct Shared {
        Shared(const std::shared_ptr<core::StreamWriter>& writer);

        std::mutex mutex;
        std::shared_ptr<core::StreamWriter> writer;
        std::unordered_map<const Descriptor*, uint32_t> type_ids;
        uint64_t group_id;
    };

    PackEncoderImpl(const std::shared_ptr<Shared>& shared, uint64_t group_id);

    void writeGroupID();
    uint64_t writeTypeIfNew(const Descriptor* desc);
    void writeString(const std::string& str);
    void writeVarint32(uint32_t value);
    void writeZigzag(int64_t value);
    void writeVarint(uint64_t value);
    void writeVarintDirect(uint64_t value);
    void flushChunk();

    std::shared_ptr<Shared> mShared;
    uint64_t mGroupId;

    std::string mBuffer; // Flushes to mShared->mWriter
};

PackEncoderImpl::Shared::Shared(const std::shared_ptr<core::StreamWriter>& writer)
        : group_id(0)
        , writer(writer) {}

PackEncoderImpl::PackEncoderImpl(const std::shared_ptr<core::StreamWriter>& writer)
        : mShared(new Shared(writer))
        , mGroupId(NO_ID) {
    writer->write(magic, sizeof(magic) - 1);
    pack::Header header;
    header.mutable_version()->set_major(VERSION_MAJOR);
    header.mutable_version()->set_minor(VERSION_MINOR);

    header.SerializeToString(&mBuffer);
    flushChunk();
}

PackEncoderImpl::PackEncoderImpl(const std::shared_ptr<Shared>& shared, uint64_t group_id)
        : mShared(shared)
        , mGroupId(group_id) {
}

PackEncoderImpl::~PackEncoderImpl() {
    if (mGroupId != NO_ID) {
        std::lock_guard<std::mutex> lock(mShared->mutex);
        writeZigzag(TAG_GROUP_FINALIZER);
        writeGroupID();
        flushChunk();
    }
}

void PackEncoderImpl::object(const Message* msg) {
    std::lock_guard<std::mutex> lock(mShared->mutex);

    auto type_id = writeTypeIfNew(msg->GetDescriptor());
    writeZigzag(TAG_FIRST_OBJECT + (int64_t)type_id);
    writeGroupID();
    msg->AppendToString(&mBuffer);
    flushChunk();
}

gapii::PackEncoder::SPtr PackEncoderImpl::group(const Message* msg) {
    std::lock_guard<std::mutex> lock(mShared->mutex);

    auto type_id = writeTypeIfNew(msg->GetDescriptor());
    writeZigzag(TAG_FIRST_GROUP - (int64_t)type_id);
    writeGroupID();
    msg->AppendToString(&mBuffer);
    flushChunk();

    auto out = PackEncoder::SPtr(new PackEncoderImpl(mShared, mShared->group_id));

    mShared->group_id++;

    return out;
}

void PackEncoderImpl::writeGroupID() {
    if (mGroupId == NO_ID) {
        writeVarint(0);
    } else {
        writeVarint(mShared->group_id - mGroupId);
    }
}

uint64_t PackEncoderImpl::writeTypeIfNew(const Descriptor* desc) {
    for (int i = 0; i < desc->field_count(); i++) {
        if (auto fieldDesc = desc->field(i)->message_type()) {
            writeTypeIfNew(fieldDesc);
        }
    }
    for (int i = 0; i < desc->nested_type_count(); i++) {
        writeTypeIfNew(desc->nested_type(i));
    }
    auto insert = mShared->type_ids.insert(std::make_pair(desc, mShared->type_ids.size()));
    if (insert.second) {
        DescriptorProto descMsg;
        desc->CopyTo(&descMsg);
        writeZigzag(TAG_DECLARE_TYPE);
        writeString(desc->full_name());
        descMsg.AppendToString(&mBuffer);
        flushChunk();
    }
    return insert.first->second;
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

void PackEncoderImpl::writeZigzag(int64_t n) {
    writeVarint(uint64_t((n << 1) ^ (n >> 63)));
}

void PackEncoderImpl::writeVarint(uint64_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
    mBuffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

void PackEncoderImpl::writeVarintDirect(uint64_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
    mShared->writer->write(&buf[0], count);
}

void PackEncoderImpl::flushChunk() {
    writeVarintDirect(mBuffer.size());
    mShared->writer->write(mBuffer.data(), mBuffer.size());
    mBuffer.clear();
}

// PackEncoderNoop is a no-op implementation of the PackEncoder interface.
class PackEncoderNoop : public gapii::PackEncoder {
public:
    static SPtr instance;

    virtual void object(const ::google::protobuf::Message* msg) override {}
    virtual SPtr group(const ::google::protobuf::Message* msg) override {
        return instance;
    }
};

gapii::PackEncoder::SPtr PackEncoderNoop::instance = gapii::PackEncoder::SPtr(new PackEncoderNoop);

} // anonymous namespace

namespace gapii {

// create returns a PackEncoder::SPtr that writes to output.
PackEncoder::SPtr PackEncoder::create(std::shared_ptr<core::StreamWriter> output) {
    return PackEncoder::SPtr(new PackEncoderImpl(output));
}

// noop returns a PackEncoder::SPtr that does nothing.
PackEncoder::SPtr PackEncoder::noop() {
    return PackEncoderNoop::instance;
}

} // namespace gapii
