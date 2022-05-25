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

#include "runtime.h"

#ifdef __cplusplus
extern "C" {
#endif  // __cplusplus

////////////////////////////////////////////////////////////////////////////////
// Serialization callbacks                                                    //
////////////////////////////////////////////////////////////////////////////////

#ifndef DECL_GAPIL_ENCODER_CB
#define DECL_GAPIL_ENCODER_CB(RETURN, NAME, ...) RETURN NAME(__VA_ARGS__)
#endif

// gapil_encode_type returns a new positive unique reference identifer if
// the type has not been encoded before in this scope, otherwise it returns the
// negated ID of the previously encoded type identifier.
DECL_GAPIL_ENCODER_CB(int64_t, gapil_encode_type, context* ctx,
                      const char* name, uint32_t desc_size, const void* desc);

// gapil_encode_object encodes the object.
// If is_group is true, a new encoder will be returned for encoding sub-objects.
// If is_group is false then gapil_encode_object will return null.
DECL_GAPIL_ENCODER_CB(void*, gapil_encode_object, context* ctx,
                      uint8_t is_group, uint32_t type, uint32_t data_size,
                      void* data);

// gapil_encode_backref returns a new positive unique reference identifer if
// object has not been encoded before in this scope, otherwise it returns the
// negated ID of the previously encoded object identifier.
DECL_GAPIL_ENCODER_CB(int64_t, gapil_encode_backref, context* ctx,
                      const void* object);

// gapil_slice_encoded is called whenever a slice is encoded. This callback
// can be used to write the slice's data into the encoder's stream.
DECL_GAPIL_ENCODER_CB(void, gapil_slice_encoded, context* ctx,
                      const void* slice);

#undef DECL_GAPIL_ENCODER_CB

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  // __GAPIL_RUNTIME_ENCODER_H__
