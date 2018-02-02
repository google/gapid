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

#ifndef __GAPIL_RUNTIME_STRING_H__
#define __GAPIL_RUNTIME_STRING_H__

#include "runtime.h"

#include <functional>

namespace core {
class Arena;
}  // namespace core

namespace gapil {

class String {
public:
    String();
    String(const String&);
    String(String&&);
    String(core::Arena* arena, const char*);
    String(core::Arena* arena, std::initializer_list<char>);
    String(core::Arena* arena, const char* start, const char* end);
    String(core::Arena* arena, const char* start, size_t len);

    ~String();

    String& operator = (const String&);
    String& operator += (const String&);

    bool operator == (const String&) const;
    bool operator != (const String&) const;
    bool operator < (const String&) const;
    bool operator <= (const String&) const;
    bool operator > (const String&) const;
    bool operator >= (const String&) const;

    size_t length() const;

    const char* c_str() const;

    void clear();

private:
    static string_t EMPTY;

    String(string_t*);

    void reference();
    void release();

    string_t* ptr;
};

}  // namespace gapil


namespace std {
template <> struct hash<gapil::String> {
    typedef gapil::String argument_type;
    typedef std::size_t result_type;
    result_type operator()(const argument_type& s) const noexcept {
        auto len = s.length();
        result_type hash = 0x32980321;
        if (auto p = s.c_str()) {
            for (size_t i = 0; i < len; i++) {
                hash = hash * 33 ^ p[i];
            }
        }
        return hash;
    }
};
} // namespace std

#endif  // __GAPIL_RUNTIME_STRING_H__