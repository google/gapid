/*
 * Copyright (C) 2019 Google Inc.
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

/*
 * GAPII-GAPIS Protocol
 *
 * Messages sent between GAPII and GAPIS consists of a fixed-size header
 * followed by the message data, which can be empty.
 *
 * The header starts with one byte describing the message type, as defined
 * below in the enum MessageType. It is followed by the data size, expressed
 * as a 40bit unsigned integer, sent as 5 little-endian bytes.
 */

// This protocol is mirrored in gapii/client/protocol.go

#ifndef GAPII_PROTOCOL_H
#define GAPII_PROTOCOL_H

#include <string>

namespace gapii {
namespace protocol {

// Header size is one byte for the type and 5 bytes for the data size
const uint8_t kHeaderSize = 1u + 5u;

enum class MessageType : uint8_t {
  kData = 0x00u,
  kStartTrace = 0x01u,
  kEndTrace = 0x02u,
  kError = 0x03u
};

// Write header into given buffer. Buffer size must be at least kHeaderSize
inline void writeHeader(uint8_t* buffer, MessageType msg_type,
                        uint64_t data_size = 0u) {
  buffer[0] = static_cast<uint8_t>(msg_type);
  buffer[1] = static_cast<uint8_t>(data_size);
  buffer[2] = static_cast<uint8_t>(data_size >> 8u);
  buffer[3] = static_cast<uint8_t>(data_size >> 16u);
  buffer[4] = static_cast<uint8_t>(data_size >> 24u);
  buffer[5] = static_cast<uint8_t>(data_size >> 32u);
}

inline std::string createHeader(MessageType msg_type, uint64_t msg_size = 0u) {
  uint8_t buf[kHeaderSize];
  writeHeader(buf, msg_type, msg_size);
  return std::string(reinterpret_cast<char*>(&buf[0]), kHeaderSize);
}

inline std::string createError(const std::string& error) {
  return createHeader(MessageType::kError, error.size()) + error;
}

}  // namespace protocol
}  // namespace gapii

#endif  // GAPII_PROTOCOL_H
