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
#include "schema.h"
#include "stream_writer.h"

#include <cstring>

namespace core {

Encoder::Encoder(std::shared_ptr<StreamWriter> output) : mOutput(output), mLastObjectId(0) {
    mEntities.insert(std::make_pair(nullptr, 0));
}

void Encoder::Bool(bool v) {
    uint8_t b = v ? 1 : 0;
    mOutput->write(&b, 1);
}

void Encoder::Int8(int8_t v) {
    mOutput->write(&v, 1);
}

void Encoder::Uint8(uint8_t v) {
    mOutput->write(&v, 1);
}

void Encoder::Uint16(uint16_t v) {
    uint8_t buf[9];
    uint16_t space = 0x7f;
    uint8_t tag = 0;
    for (int o = 8; true; o--) {
        if (v <= space) {
            buf[o] = uint8_t(v) | uint8_t(tag);
            mOutput->write(buf + o, 9 - o);
            return;
        }
        buf[o] = uint8_t(v);
        v = v >> 8;
        space >>= 1;
        tag = (tag >> 1) | 0x80;
    }
}

void Encoder::Int16(int16_t v) {
    uint16_t uv = uint16_t(v) << 1;
    Uint16((v < 0) ? ~uv : uv);
}

void Encoder::Float32(float v) {
    uint32_t bits = *reinterpret_cast<uint32_t*>(&v);
    uint32_t shuffled =
            ((bits & 0x000000ff) << 24) |
            ((bits & 0x0000ff00) << 8)  |
            ((bits & 0x00ff0000) >> 8)  |
            ((bits & 0xff000000) >> 24);
    return Uint32(shuffled);
}

void Encoder::Uint32(uint32_t v) {
    uint8_t buf[9];
    uint32_t space = 0x7f;
    uint8_t tag = 0;
    for (int o = 8; true; o--) {
        if (v <= space) {
            buf[o] = uint8_t(v) | uint8_t(tag);
            mOutput->write(buf + o, 9 - o);
            return;
        }
        buf[o] = uint8_t(v);
        v = v >> 8;
        space >>= 1;
        tag = (tag >> 1) | 0x80;
    }
}

void Encoder::Int32(int32_t v) {
    uint32_t uv = uint32_t(v) << 1;
    Uint32((v < 0) ? ~uv : uv);
}

void Encoder::Float64(double v) {
    uint64_t bits = *reinterpret_cast<uint64_t*>(&v);
    uint64_t shuffled =
            ((bits & 0x00000000000000ffULL) << 56) |
            ((bits & 0x000000000000ff00ULL) << 40) |
            ((bits & 0x0000000000ff0000ULL) << 24) |
            ((bits & 0x00000000ff000000ULL) << 8)  |
            ((bits & 0x000000ff00000000ULL) >> 8)  |
            ((bits & 0x0000ff0000000000ULL) >> 24) |
            ((bits & 0x00ff000000000000ULL) >> 40) |
            ((bits & 0xff00000000000000ULL) >> 56);
    return Uint64(shuffled);
}

void Encoder::Uint64(uint64_t v) {
    uint8_t buf[9];
    uint64_t space = 0x7f;
    uint8_t tag = 0;
    for (int o = 8; true; o--) {
        if (v <= space) {
            buf[o] = uint8_t(v) | uint8_t(tag);
            mOutput->write(buf + o, 9 - o);
            return;
        }
        buf[o] = uint8_t(v);
        v = v >> 8;
        space >>= 1;
        tag = (tag >> 1) | 0x80;
    }
}

void Encoder::Int64(int64_t v) {
    uint64_t uv = uint64_t(v) << 1;
    Uint64((v < 0) ? ~uv : uv);
}

void Encoder::Pointer(const void* p) {
    Uint64(reinterpret_cast<uintptr_t>(p));
    Uint32(0); // Pool-id
}

void Encoder::String(const char* v) {
    uint32_t len = v != nullptr ? static_cast<uint32_t>(strlen(v)) : 0;
    Uint32(len);
    mOutput->write(v, len);
}

void Encoder::Data(const void* ptr, int32_t size) {
    mOutput->write(ptr, size);
}

void Encoder::Entity(const schema::Entity* entity) {
    auto ret = mEntities.insert(
        std::make_pair(entity, mEntities.size()));
    uint32_t sid = ret.first->second;
    if (!ret.second) {
        Uint32(sid << 1);
    } else {
        Uint32((sid << 1) | 1);
        entity->encode(*this);
    }
}

void Encoder::Struct(const Encodable& obj) {
    obj.Encode(this);
}

void Encoder::Variant(const Encodable* obj) {
    if (obj == nullptr) {
        Entity(nullptr);
        return;
    }
    Entity(obj->Schema());
    obj->Encode(this);
}

void Encoder::Object(const Encodable* obj) {
    if (obj == nullptr) {
        Uint32(0);
        return;
    }
    uint32_t sid = ++mLastObjectId;
    Uint32((sid << 1) | 1);
    Variant(obj);
}

}  // namespace core
