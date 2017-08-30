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

#ifndef GAPII_CONNECTION_HEADER_H
#define GAPII_CONNECTION_HEADER_H

#include <stddef.h>
#include <stdint.h>

namespace core {

class StreamReader;

}  // namespace core

namespace gapii {

// ConnectionHeader is the first packet of data sent from the tool controlling
// the capture to the interceptor.
// All fields are encoded little-endian with no compression, regardless of
// architecture.
class ConnectionHeader {
public:
    ConnectionHeader();

    static const size_t MAX_PATH = 512;

    // Fakes no support for PCS, forcing the app to share shader source.
    static const uint32_t FLAG_DISABLE_PRECOMPILED_SHADERS = 0x00000001;
    // Driver errors are queried after each call and stored as extras.
    static const uint32_t FLAG_RECORD_ERROR_STATE          = 0x10000000;
    // Defers the start frame until a message is receieved over the network.
    static const uint32_t FLAG_DEFER_START                 = 0x00000010;

    // read reads the ConnectionHeader from the provided stream, returning true
    // on success or false on error.
    bool read(core::StreamReader* reader);

    uint8_t  mMagic[4];                     // 's', 'p', 'y', '0'
    uint32_t mVersion;                      // 1
    uint32_t mObserveFrameFrequency;        // non-zero == enabled.
    uint32_t mObserveDrawFrequency;         // non-zero == enabled.
    uint32_t mStartFrame;                   // non-zero == Frame to start at.
    uint32_t mNumFrames;                    // non-zero == Number of frames to capture.
    uint32_t mAPIs;                         // Bitset of APIS to enable.
    uint32_t mFlags;                        // Combination of FLAG_XX bits.
    uint64_t mGvrHandle;                    // Handle of GVR library.
    char     mLibInterceptorPath[MAX_PATH]; // Path of libinterceptor.so.
};

} // namespace gapii

#endif // GAPII_CONNECTION_HEADER_H
