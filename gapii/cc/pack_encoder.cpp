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

#include "core/cc/stream_writer.h"

#include <google/protobuf/descriptor.pb.h>
#include <google/protobuf/io/coded_stream.h>
#include <google/protobuf/message.h>

#include <mutex>

using ::google::protobuf::Descriptor;
using ::google::protobuf::DescriptorProto;
using ::google::protobuf::Message;
using ::google::protobuf::io::CodedOutputStream;

namespace {

const uint64_t NO_ID = 0xffffffffffffffff;

constexpr int TYPE_ID_CACHE_COUNT = 4;

const char header[] = "ProtoPack\r\n2.0\n";

// PackEncoderImpl implements the PackEncoder interface.
class PackEncoderImpl : public gapii::PackEncoder {
 public:
  PackEncoderImpl(const std::shared_ptr<core::StringWriter>& writer);
  ~PackEncoderImpl();

  virtual TypeIDAndIsNew type(const char* name, size_t size,
                              const void* data) override;
  virtual void object(const Message* msg) override;
  virtual void object(TypeID type, size_t size, const void* data) override;
  virtual SPtr group(const Message* msg) override;
  virtual PackEncoder* group(TypeID type, size_t size,
                             const void* data) override;
  virtual void flush() override;

 private:
  struct TypeIDCache {
    std::mutex mutex;
    std::unordered_map<const void*, TypeID> type_ids;
  };

  struct Shared {
    Shared(const std::shared_ptr<core::StringWriter>& writer);

    std::recursive_mutex mutex;
    std::shared_ptr<core::StringWriter> writer;
    std::unordered_map<const void*, TypeID> type_ids;
    TypeIDCache type_id_caches[TYPE_ID_CACHE_COUNT];
    uint64_t mCurrentChunkId;
  };

  PackEncoderImpl(const std::shared_ptr<Shared>& shared,
                  uint64_t parentChunkId);

  void writeParentID(std::string& buffer);
  TypeIDAndIsNew writeTypeIfNew(const Descriptor* desc);
  TypeIDAndIsNew writeTypeIfNew(const char* name, size_t size,
                                const void* data);
  TypeIDAndIsNew writeTypeIfNewBlocking(const Descriptor* desc);
  TypeIDAndIsNew writeTypeIfNewBlocking(const char* name, size_t size,
                                        const void* data);
  void writeString(std::string& buffer, const std::string& str);
  void writeZigzag(std::string& buffer, int64_t value);
  void writeVarint(std::string& buffer, uint64_t value);
  uint64_t flushChunk(std::string& buffer, bool isTypeDefChunk);

  std::shared_ptr<Shared> mShared;
  uint64_t mParentChunkId;
};

PackEncoderImpl::Shared::Shared(
    const std::shared_ptr<core::StringWriter>& writer)
    : writer(writer), type_ids{{nullptr, 0}}, mCurrentChunkId(0) {}

PackEncoderImpl::PackEncoderImpl(
    const std::shared_ptr<core::StringWriter>& writer)
    : mShared(new Shared(writer)), mParentChunkId(NO_ID) {}

PackEncoderImpl::PackEncoderImpl(const std::shared_ptr<Shared>& shared,
                                 uint64_t parentChunkId)
    : mShared(shared), mParentChunkId(parentChunkId) {}

PackEncoderImpl::~PackEncoderImpl() {
  if (mParentChunkId != NO_ID) {
    std::string buffer;
    std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
    writeParentID(buffer);
    flushChunk(buffer, false);
  }
}

void PackEncoderImpl::flush() { mShared->writer->flush(); }

gapii::PackEncoder::TypeIDAndIsNew PackEncoderImpl::type(const char* name,
                                                         size_t size,
                                                         const void* data) {
  return writeTypeIfNew(name, size, data);
}

void PackEncoderImpl::object(const Message* msg) {
  std::string buffer;
  auto type_id = writeTypeIfNew(msg->GetDescriptor()).first;

  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  writeParentID(buffer);
  writeZigzag(buffer, type_id);
  msg->AppendToString(&buffer);
  flushChunk(buffer, false);
}

void PackEncoderImpl::object(TypeID type_id, size_t size, const void* data) {
  std::string buffer;
  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  writeParentID(buffer);
  writeZigzag(buffer, type_id);
  buffer.append(reinterpret_cast<const char*>(data), size);
  flushChunk(buffer, false);
}

gapii::PackEncoder::SPtr PackEncoderImpl::group(const Message* msg) {
  std::string buffer;
  auto type_id = writeTypeIfNew(msg->GetDescriptor()).first;

  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  writeParentID(buffer);
  writeZigzag(buffer, -(int64_t)type_id);
  msg->AppendToString(&buffer);
  auto chunkID = flushChunk(buffer, false);

  return PackEncoder::SPtr(new PackEncoderImpl(mShared, chunkID));
}

gapii::PackEncoder* PackEncoderImpl::group(TypeID type_id, size_t size,
                                           const void* data) {
  std::string buffer;
  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  writeParentID(buffer);
  writeZigzag(buffer, -(int64_t)type_id);
  buffer.append(reinterpret_cast<const char*>(data), size);
  auto chunkID = flushChunk(buffer, false);

  return new PackEncoderImpl(mShared, chunkID);
}

void PackEncoderImpl::writeParentID(std::string& buffer) {
  if (mParentChunkId == NO_ID) {
    writeZigzag(buffer, 0);
  } else {
    writeZigzag(buffer, mParentChunkId - mShared->mCurrentChunkId);
  }
}

// TODO: Refactor the body of this out into something that can be shared with
//       the two overloads.
gapii::PackEncoder::TypeIDAndIsNew PackEncoderImpl::writeTypeIfNew(
    const Descriptor* desc) {
  TypeIDCache* type_id_cache = nullptr;
  std::unique_lock<std::mutex> cache_lock;

  for (int index = 0; !type_id_cache && index < TYPE_ID_CACHE_COUNT; ++index) {
    if (mShared->type_id_caches[index].mutex.try_lock()) {
      type_id_cache = &mShared->type_id_caches[index];
      cache_lock =
          std::unique_lock<std::mutex>(type_id_cache->mutex, std::adopt_lock);
    }
  }

  if (!type_id_cache) {
    return writeTypeIfNewBlocking(desc);
  }

  TypeIDAndIsNew res;

  auto iter_id = type_id_cache->type_ids.find(desc);
  if (iter_id == end(type_id_cache->type_ids)) {
    res = writeTypeIfNewBlocking(desc);
    type_id_cache->type_ids.insert(std::make_pair(desc, res.first));
  } else {
    res = std::make_pair(iter_id->second, false);
  }

  return res;
}

gapii::PackEncoder::TypeIDAndIsNew PackEncoderImpl::writeTypeIfNew(
    const char* name, size_t size, const void* data) {
  TypeIDCache* type_id_cache = nullptr;
  std::unique_lock<std::mutex> cache_lock;

  for (int index = 0; !type_id_cache && index < TYPE_ID_CACHE_COUNT; ++index) {
    if (mShared->type_id_caches[index].mutex.try_lock()) {
      type_id_cache = &mShared->type_id_caches[index];
      cache_lock =
          std::unique_lock<std::mutex>(type_id_cache->mutex, std::adopt_lock);
    }
  }

  if (!type_id_cache) {
    return writeTypeIfNewBlocking(name, size, data);
  }

  TypeIDAndIsNew res;

  auto iter_id = type_id_cache->type_ids.find(data);
  if (iter_id == end(type_id_cache->type_ids)) {
    res = writeTypeIfNewBlocking(name, size, data);
    type_id_cache->type_ids.insert(std::make_pair(data, res.first));
  } else {
    res = std::make_pair(iter_id->second, false);
  }

  return res;
}

gapii::PackEncoder::TypeIDAndIsNew PackEncoderImpl::writeTypeIfNewBlocking(
    const char* name, size_t size, const void* data) {
  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  auto insert =
      mShared->type_ids.insert(std::make_pair(data, mShared->type_ids.size()));
  auto type_id = insert.first->second;
  if (!insert.second) {
    return std::make_pair(type_id, false);
  }

  std::string buffer;
  writeString(buffer, name);
  buffer.append(reinterpret_cast<const char*>(data), size);
  flushChunk(buffer, true);
  return std::make_pair(type_id, true);
}

gapii::PackEncoder::TypeIDAndIsNew PackEncoderImpl::writeTypeIfNewBlocking(
    const Descriptor* desc) {
  std::lock_guard<std::recursive_mutex> lock(mShared->mutex);
  auto insert =
      mShared->type_ids.insert(std::make_pair(desc, mShared->type_ids.size()));
  auto type_id = insert.first->second;
  if (!insert.second) {
    return std::make_pair(type_id, false);
  }

  std::string buffer;
  DescriptorProto descMsg;
  desc->CopyTo(&descMsg);
  writeString(buffer, desc->full_name());
  descMsg.AppendToString(&buffer);
  flushChunk(buffer, true);

  for (int i = 0; i < desc->field_count(); i++) {
    if (auto fieldDesc = desc->field(i)->message_type()) {
      writeTypeIfNewBlocking(fieldDesc);
    }
  }
  for (int i = 0; i < desc->nested_type_count(); i++) {
    writeTypeIfNewBlocking(desc->nested_type(i));
  }

  return std::make_pair(type_id, true);
}

void PackEncoderImpl::writeString(std::string& buffer, const std::string& str) {
  writeVarint(buffer, str.length());
  buffer.append(str);
}

void PackEncoderImpl::writeZigzag(std::string& buffer, int64_t n) {
  writeVarint(buffer, uint64_t((n << 1) ^ (n >> 63)));
}

void PackEncoderImpl::writeVarint(std::string& buffer, uint64_t value) {
  uint8_t buf[16];
  auto count =
      CodedOutputStream::WriteVarint64ToArray(value, &buf[0]) - &buf[0];
  buffer.append(reinterpret_cast<char*>(&buf[0]), count);
}

uint64_t PackEncoderImpl::flushChunk(std::string& buffer, bool isTypeDefChunk) {
  const int64_t size = buffer.size();
  std::string sizeBuffer;
  writeZigzag(sizeBuffer, isTypeDefChunk ? -size : size);
  mShared->writer->write({&sizeBuffer, &buffer});
  buffer.clear();
  return mShared->mCurrentChunkId++;
}

// PackEncoderNoop is a no-op implementation of the PackEncoder interface.
class PackEncoderNoop : public gapii::PackEncoder {
 public:
  static SPtr instance;

  virtual TypeIDAndIsNew type(const char* name, size_t size,
                              const void* data) override {
    return std::make_pair(0, false);
  }
  virtual void object(const Message* msg) override {}
  virtual void object(TypeID type, size_t size, const void* data) override {}
  virtual SPtr group(const Message* msg) override { return instance; }
  virtual PackEncoder* group(TypeID type, size_t size,
                             const void* data) override {
    return new PackEncoderNoop();
  }
  virtual void flush() override {}
};

gapii::PackEncoder::SPtr PackEncoderNoop::instance =
    gapii::PackEncoder::SPtr(new PackEncoderNoop);

}  // anonymous namespace

namespace gapii {

// create returns a PackEncoder::SPtr that writes to output.
PackEncoder::SPtr PackEncoder::create(
    std::shared_ptr<core::StreamWriter> stream, bool no_buffer) {
  auto writer = ChunkWriter::create(stream, no_buffer);
  std::string header_chunk(header, sizeof(header));
  writer->write({&header_chunk});
  writer->flush();  // don't buffer header, otherwise client will time out
  return PackEncoder::SPtr(new PackEncoderImpl(writer));
}

// noop returns a PackEncoder::SPtr that does nothing.
PackEncoder::SPtr PackEncoder::noop() { return PackEncoderNoop::instance; }

}  // namespace gapii
