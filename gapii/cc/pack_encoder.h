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

#ifndef GAPII_PACK_ENCODER_H
#define GAPII_PACK_ENCODER_H

#include <google/protobuf/io/coded_stream.h>
#include <google/protobuf/io/zero_copy_stream_impl_lite.h>

#include <memory>
#include <unordered_map>
#include <utility>

namespace google {
namespace protobuf {
class Descriptor;
class Message;
}  // namespace protobuf
}  // namespace google

namespace core {
class StreamWriter;
}  // namespace core

namespace gapii {

// PackEncoder provides methods for encoding protobuf messages to the provided
// StreamWriter using the pack-stream format.
class PackEncoder {
 public:
  typedef std::shared_ptr<PackEncoder> SPtr;

  typedef uint32_t TypeID;
  typedef std::pair<TypeID, bool> TypeIDAndIsNew;

  virtual ~PackEncoder() = default;

  // type encodes the given type descriptor if it hasn't been already,
  // returning the type identifier and a boolean indicating whether the type
  // was encoded this call.
  // type assumes the data pointer is stable between calls of the same type.
  virtual TypeIDAndIsNew type(const char* name, size_t size,
                              const void* data) = 0;

  // object encodes the leaf protobuf message.
  virtual void object(const ::google::protobuf::Message* msg) = 0;

  // object encodes the leaf object from an already encoded protobuf message.
  virtual void object(TypeID type, size_t size, const void* data) = 0;

  // group encodes the protobuf message as a group that can contain other
  // objects and groups.
  virtual SPtr group(const ::google::protobuf::Message* msg) = 0;

  // object encodes the group object from an already encoded protobuf message.
  // The returned PackEncoder can be used to encode objects into this group,
  // and must be deleted by the caller.
  virtual PackEncoder* group(TypeID type, size_t size, const void* data) = 0;

  // flush flushes out all of the pending in the encoder
  virtual void flush() = 0;

  // create returns a PackEncoder::SPtr that writes to output.
  // If no_buffer is true, thn the output will be flushed after every write.
  static SPtr create(std::shared_ptr<core::StreamWriter> output,
                     bool no_buffer);

  // noop returns a PackEncoder::SPtr that does nothing.
  static SPtr noop();
};

}  // namespace gapii

#endif  // GAPII_PACK_ENCODER_H
