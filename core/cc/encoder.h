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

#ifndef CORE_ENCODER_H
#define CORE_ENCODER_H

#include "id.h"
#include "assert.h"

#include <memory>
#include <stdint.h>
#include <string>
#include <unordered_map>
#include <vector>

namespace core {

namespace schema {
class Entity;
}

class StreamWriter;
class Encodable;
class Encoder;

class Encodable {
public:
    virtual void Encode(Encoder* to) const = 0;
    virtual const schema::Entity* Schema() const = 0;
};

// Encoder provides methods for encoding values to the provided StreamWriter
// using variable-length-encoding.
class Encoder {
public:
    Encoder(std::shared_ptr<StreamWriter> output);

    void Bool(bool);
    void Int8(int8_t);
    void Uint8(uint8_t);
    void Uint16(uint16_t);
    void Int16(int16_t);
    void Float32(float);
    void Uint32(uint32_t);
    void Int32(int32_t);
    void Float64(double);
    void Uint64(uint64_t);
    void Int64(int64_t);
    void Pointer(const void*);
    void String(const char*);
    void String(const std::string& str) { String(str.c_str()); }
    void Data(const void* ptr, int32_t size);
    void Id(const core::Id&);
    void Entity(const schema::Entity*);

    void Encode(bool b) { Bool(b); }
    void Encode(int8_t v) { Int8(v); }
    void Encode(uint8_t v) { Uint8(v); }
    void Encode(uint16_t v) { Uint16(v); }
    void Encode(float v) { Float32(v); }
    void Encode(uint32_t v) { Uint32(v); }
    void Encode(int32_t v) { Int32(v); }
    void Encode(double v) { Float64(v); }
    void Encode(uint64_t v) { Uint64(v); }
    void Encode(const char* v) { String(v); }
    void Encode(const std::string& v) { String(v); }
    void Encode(const core::Id& id) { Id(id); }

    void Struct(const Encodable& obj);
    void Variant(const Encodable* obj);
    void Object(const Encodable* obj);

private:
    std::unordered_map<core::Id, uint32_t> mIds;
    std::unordered_map<const schema::Entity*, uint32_t> mEntities;
    std::shared_ptr<StreamWriter> mOutput;
    uint32_t mLastObjectId;
};


}  // namespace core

#endif  // CORE_ENCODER_H
