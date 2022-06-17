#pragma once

/*
 * Copyright (C) 2022 Google Inc.
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

#include "command_serializer.h"
#include <cstring>

namespace gapid2 {

void command_serializer::insert_annotation(const char* data) {
  auto len = strlen(data);
  auto enc = get_encoder(0);
  enc->encode<uint64_t>(1);
  enc->encode<uint64_t>(get_flags());
  enc->encode<uint64_t>(len+1);
  enc->encode_primitive_array(data, len + 1);
}
}