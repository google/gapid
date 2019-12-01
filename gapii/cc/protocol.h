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

#ifndef GAPII_PROTOCOL_H
#define GAPII_PROTOCOL_H

#include <string>

#include <google/protobuf/io/coded_stream.h>

namespace gapii {
namespace protocol {

// Maximum header size is maximum varint size plus message type size
const uint8_t kMaxHeaderSize = 10u + 1u;

enum class MessageType : uint8_t {
  kData = 0x00u,
  kStartTrace = 0x01u,
  kEndTrace = 0x02u,
  kError = 0x03u
};

std::string createHeader(MessageType msg_type, uint64_t msg_size = 0u) {
  msg_size <<= 1u;
  if (msg_type != MessageType::kData) {
    ++msg_size;
  }
  uint8_t buf[16];
  uint8_t* buf_end =
      google::protobuf::io::CodedOutputStream::WriteVarint64ToArray(msg_size,
                                                                    &buf[0]);
  if (msg_type != MessageType::kData) {
    *buf_end++ = static_cast<uint8_t>(msg_type);
  }
  auto count = buf_end - &buf[0];
  return std::string(reinterpret_cast<char*>(&buf[0]), count);
}

}  // namespace protocol
}  // namespace gapii

#endif  // GAPII_PROTOCOL_H
