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

#include "common.h"
#include <assert.h>

std::vector<uint32_t> makeVector(uint32_t word) {
  std::vector<uint32_t> result = {word};
  return result;
}

std::vector<uint32_t> makeVector(std::initializer_list<uint32_t> list) {
  std::vector<uint32_t> result(list);
  return result;
}

std::vector<uint32_t> makeVector(const char* str) {
  assert(str != nullptr && "makeVector: making vector from null pointer.");
  uint32_t word;
  std::vector<uint32_t> result;
  char* word_string = (char*)&word;
  char* word_ptr = word_string;
  int char_count = 0;
  char c;

  if (str != nullptr) {
    do {
      c = *(str++);
      *(word_ptr++) = c;
      ++char_count;
      if (char_count == 4) {
        result.push_back(word);
        word_ptr = word_string;
        char_count = 0;
      }
    } while (c != 0);

    // deal with partial last word
    if (char_count > 0) {
      // pad with 0s
      for (; char_count < 4; ++char_count) *(word_ptr++) = 0;
      result.push_back(word);
    }
  }
  return result;
}

/**
 * Return string represented by given binary.
 **/
std::string extractString(const std::vector<uint32_t>& words) {
  std::string ret;
  for (uint32_t i = 0; i < words.size(); i++) {
    uint32_t w = words[i];

    for (uint32_t j = 0; j < 4; j++, w >>= 8) {
      char c = w & 0xff;
      if (c == '\0') return ret;
      ret += c;
    }
  }
  assert(false &&
         "extractString: expected the vector to represent a null-terminated "
         "string");
  return ret;
}
