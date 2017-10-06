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
#include "chunk_writer.h"
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

namespace {

const int      VERSION_MAJOR       = 1;
const int      VERSION_MINOR       = 1;
const int64_t  TAG_FIRST_GROUP     = -1;
const int64_t  TAG_GROUP_FINALIZER = 0;
const int64_t  TAG_DECLARE_TYPE    = 1;
const int64_t  TAG_FIRST_OBJECT    = 2;
const uint64_t NO_ID               = 0xffffffffffffffff;

constexpr int TYPE_ID_CACHE_COUNT  = 4;

const char magic[] = "protopack";

// PackEncoderImpl implements the PackEncoder interface.
class PackEncoderImpl : public gapii::PackEncoder {
public:
    PackEncoderImpl(const std::shared_ptr<core::StringWriter>& writer);
    ~PackEncoderImpl();

    virtual void object(const Message* msg) override;
    virtual SPtr group(const Message* msg) override;
    virtual void flush() override;

private:
    struct TypeIDCache {
        std::mutex mutex;
        std::unordered_map<const Descriptor*, uint32_t> type_ids;
    };

    struct Shared {
        Shared(const std::shared_ptr<core::StringWriter>& writer);

        std::mutex mutex;
        std::shared_ptr<core::StringWriter> writer;
        std::unordered_map<const Descriptor*, uint32_t> type_ids;
        TypeIDCache type_id_caches[TYPE_ID_CACHE_COUNT];
        uint64_t group_id;
    };

    PackEncoderImpl(const std::shared_ptr<Shared>& shared, uint64_t group_id);

    void writeGroupID(std::string& buffer);
    uint64_t writeTypeIfNew(const Descriptor* desc);
    uint64_t writeTypeIfNewBlocking(const Descriptor* desc);
    void writeString(std::string& buffer, const std::string& str);
    void writeVarint32(std::string& buffer, uint32_t value);
    void writeZigzag(std::string& buffer, int64_t value);
    void writeVarint(std::string& buffer, uint64_t value);
    void flushBuffer(std::string& buffer);

    std::shared_ptr<Shared> mShared;
    uint64_t mGroupId;
};

PackEncoderImpl::Shared::Shared(const std::shared_ptr<core::StringWriter>& writer)
        : writer(writer)
        , group_id(0) {}

PackEncoderImpl::PackEncoderImpl(const std::shared_ptr<core::StringWriter>& writer)
        : mShared(new Shared(writer))
        , mGroupId(NO_ID) {
    pack::Header header;
    header.mutable_version()->set_major(VERSION_MAJOR);
    header.mutable_version()->set_minor(VERSION_MINOR);

    std::string buffer;
    header.SerializeToString(&buffer);
    flushBuffer(buffer);
}

PackEncoderImpl::PackEncoderImpl(const std::shared_ptr<Shared>& shared, uint64_t group_id)
        : mShared(shared)
        , mGroupId(group_id) {
}

PackEncoderImpl::~PackEncoderImpl() {
    if (mGroupId != NO_ID) {
        std::string buffer;
        writeZigzag(buffer, TAG_GROUP_FINALIZER);

        std::lock_guard<std::mutex> lock(mShared->mutex);
        writeGroupID(buffer);
        flushBuffer(buffer);
    }
}

void PackEncoderImpl::flush() {
    mShared->writer->flush();
}

void PackEncoderImpl::object(const Message* msg) {
    std::string buffer;
    auto type_id = writeTypeIfNew(msg->GetDescriptor());
    writeZigzag(buffer, TAG_FIRST_OBJECT + (int64_t)type_id);

    std::lock_guard<std::mutex> lock(mShared->mutex);
    writeGroupID(buffer);
    msg->AppendToString(&buffer);
    flushBuffer(buffer);
}

gapii::PackEncoder::SPtr PackEncoderImpl::group(const Message* msg) {
    std::string buffer;
    auto type_id = writeTypeIfNew(msg->GetDescriptor());
    writeZigzag(buffer, TAG_FIRST_GROUP - (int64_t)type_id);

    std::lock_guard<std::mutex> lock(mShared->mutex);
    writeGroupID(buffer);
    msg->AppendToString(&buffer);
    flushBuffer(buffer);

    auto out = PackEncoder::SPtr(new PackEncoderImpl(mShared, mShared->group_id));

    mShared->group_id++;

    return out;
}

void PackEncoderImpl::writeGroupID(std::string& buffer) {
    if (mGroupId == NO_ID) {
        writeVarint(buffer, 0);
    } else {
        writeVarint(buffer, mShared->group_id - mGroupId);
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

    TypeIDCache* type_id_cache = nullptr;
    std::unique_lock<std::mutex> cache_lock;

    for(int index = 0; !type_id_cache && index < TYPE_ID_CACHE_COUNT; ++index) {
        if (mShared->type_id_caches[index].mutex.try_lock()) {
            type_id_cache = &mShared->type_id_caches[index];
            cache_lock = std::unique_lock<std::mutex>(type_id_cache->mutex, std::adopt_lock);
        }
    }

    if (!type_id_cache) {
        return writeTypeIfNewBlocking(desc);
    }

    uint64_t type_id;

    auto iter_id = type_id_cache->type_ids.find(desc);
    if (iter_id == end(type_id_cache->type_ids)) {
        type_id = writeTypeIfNewBlocking(desc);
        type_id_cache->type_ids.insert(std::make_pair(desc, type_id));
    } else {
        type_id = iter_id->second;
    }

    return type_id;
}

uint64_t PackEncoderImpl::writeTypeIfNewBlocking(const Descriptor* desc) {
    std::lock_guard<std::mutex> lock(mShared->mutex);

    auto insert = mShared->type_ids.insert(std::make_pair(desc, mShared->type_ids.size()));
    if (insert.second) {
        std::string buffer;
        DescriptorProto descMsg;
        desc->CopyTo(&descMsg);
        writeZigzag(buffer, TAG_DECLARE_TYPE);
        writeString(buffer, desc->full_name());
        descMsg.AppendToString(&buffer);
        flushBuffer(buffer);
    }
    return insert.first->second;
}

void PackEncoderImpl::writeString(std::string& buffer, const std::string& str) {
    writeVarint(buffer, str.length());
    buffer.append(str);
}

void PackEncoderImpl::writeVarint32(std::string& buffer, uint32_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint32ToArray(value, &buf[0]) - &buf[0];
    buffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

void PackEncoderImpl::writeZigzag(std::string& buffer, int64_t n) {
    writeVarint(buffer, uint64_t((n << 1) ^ (n >> 63)));
}

void PackEncoderImpl::writeVarint(std::string& buffer, uint64_t value) {
    uint8_t buf[16];
    auto count = CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
    buffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

void PackEncoderImpl::flushBuffer(std::string& buffer) {
    mShared->writer->write(buffer);
    buffer.clear();
}

// PackEncoderNoop is a no-op implementation of the PackEncoder interface.
class PackEncoderNoop : public gapii::PackEncoder {
public:
    static SPtr instance;

    virtual void object(const ::google::protobuf::Message* msg) override {}
    virtual SPtr group(const ::google::protobuf::Message* msg) override {
        return instance;
    }
    virtual void flush() override{}
};

gapii::PackEncoder::SPtr PackEncoderNoop::instance = gapii::PackEncoder::SPtr(new PackEncoderNoop);

} // anonymous namespace

namespace gapii {

// create returns a PackEncoder::SPtr that writes to output.
PackEncoder::SPtr PackEncoder::create(std::shared_ptr<core::StreamWriter> stream) {
    stream->write(magic, sizeof(magic) - 1);
    auto writer = ChunkWriter::create(stream);
    return PackEncoder::SPtr(new PackEncoderImpl(writer));
}

// noop returns a PackEncoder::SPtr that does nothing.
PackEncoder::SPtr PackEncoder::noop() {
    return PackEncoderNoop::instance;
}

} // namespace gapii
