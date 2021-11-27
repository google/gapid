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

#pragma once

#include <fstream>
#include <iomanip>
#include <iostream>
#include <vector>

namespace gapid2 {
struct JsonPrinter {
  JsonPrinter() { os = &std::cout; }
  void set_file(const char* file) {
    fl = std::ofstream(file);
    os = &fl;
  }

  void begin_object(const char* name) {
    (*os) << depth;
    handle_comma();
    needs_comma.push_back(false);
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    (*os) << "{" << std::endl;
    depth = depth + "  ";
  }

  void end_object() {
    depth = depth.substr(2);
    (*os) << depth;
    needs_comma.pop_back();
    (*os) << "}" << std::endl;
  }

  void begin_array(const char* name) {
    (*os) << depth;
    handle_comma();
    needs_comma.push_back(false);
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    (*os) << "[" << std::endl;
    depth = depth + "  ";
  }

  void end_array() {
    depth = depth.substr(2);
    (*os) << depth;
    needs_comma.pop_back();
    (*os) << "]" << std::endl;
  }

  void handle_comma() {
    if (needs_comma.empty()) {
      (*os) << " ";
      return;
    }
    if (!needs_comma.back()) {
      (*os) << " ";
      needs_comma.back() = true;
      return;
    }
    (*os) << ",";
  }

  template <typename T>
  void print(const char* name, T val) {
    (*os) << depth;
    handle_comma();
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    (*os) << val << std::endl;
  }

  void print_null(const char* name) {
    (*os) << depth;
    handle_comma();
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    (*os) << "null" << std::endl;
  }

  void print_char_array(const char* name, const char* val, size_t size) {
    (*os) << depth;
    handle_comma();
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    (*os) << '"' << std::setw(size) << std::left << val << '"' << std::endl;
  }

  void print_string(const char* name, const char* str) {
    (*os) << depth;
    handle_comma();
    if (name[0]) {
      (*os) << '"' << name << "\" : ";
    }
    if (str) {
      (*os) << '"' << str << '"' << std::endl;
    } else {
      (*os) << "null" << std::endl;
    }
  }

  std::string depth;
  std::vector<bool> needs_comma;
  std::ostream* os;
  std::ofstream fl;
};
}  // namespace gapid2