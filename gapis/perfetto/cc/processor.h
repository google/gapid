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

#ifndef PROCESSOR_H_
#define PROCESSOR_H_

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void* processor;

typedef struct {
  size_t size;
  uint8_t* data;
} result;

processor new_processor();
bool parse_data(processor processor, const void* data, size_t size);
result execute_query(processor processor, const char* query);
void delete_processor(processor processor);

#ifdef __cplusplus
}
#endif

#endif  // PROCESSOR_H_
