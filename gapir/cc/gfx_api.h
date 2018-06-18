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

#ifndef GAPIR_GFX_API_H
#define GAPIR_GFX_API_H

#include "function_table.h"

#include <stdint.h>

namespace gapir {

class Stack;

// Api is the abstract base class to all graphics APIs.
class Api {
 public:
  // Returns the unique identifier of the graphics API.
  // The pointer is guaranteed to be constant for all instances of the API.
  virtual const char* id() const = 0;
  // Returns the index of the graphics API.
  virtual uint8_t index() const = 0;
  // The function table for the API.
  FunctionTable mFunctions;
};

}  // namespace gapir

#endif  // GAPIR_GFX_API_H
