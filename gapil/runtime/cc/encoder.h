// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#ifndef __GAPIL_RUNTIME_ENCODER_H__
#define __GAPIL_RUNTIME_ENCODER_H__

#include <stdint.h>

typedef struct pool_t pool;

namespace core {
class Arena;
}  // namespace core

namespace gapil {

// Encoder is an interface for the generated struct encoders to serialize the
// encoded bytes to a stream.
class Encoder {
 public:
  // encodeType returns a new positive unique reference identifer if
  // the type has not been encoded before in this scope, otherwise it returns
  // the negated ID of the previously encoded type identifier.
  virtual int64_t encodeType(const char* name, uint32_t desc_size,
                             const void* desc) = 0;

  // encodeObject encodes the object.
  // If is_group is true, a new encoder will be returned for encoding
  // sub-objects. If is_group is false then encodeObject will return null.
  virtual void* encodeObject(uint8_t is_group, uint32_t type,
                             uint32_t data_size, void* data) = 0;

  // encodeBackref returns a new positive unique reference identifer if
  // object has not been encoded before in this scope, otherwise it returns the
  // negated ID of the previously encoded object identifier.
  virtual int64_t encodeBackref(const void* object) = 0;

  // sliceEncoded is called whenever a slice is encoded. This callback
  // can be used to write the slice's data into the encoder's stream.
  virtual void sliceEncoded(const pool_t* pool) = 0;

  virtual core::Arena* arena() const = 0;
};

}  // namespace gapil

#endif  // __GAPIL_RUNTIME_ENCODER_H__
