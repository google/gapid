// Copyright (c) 2016 Google Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

#ifndef LIBSPIRV_TEST_OPT_PASS_UTILS_H_
#define LIBSPIRV_TEST_OPT_PASS_UTILS_H_

#include <functional>
#include <string>
#include <vector>

namespace spvtools {

// In-place substring replacement. Finds the |find_str| in the |process_str|
// and replaces the found substring with |replace_str|. Returns true if at
// least one replacement is done successfully, returns false otherwise. The
// replaced substring won't be processed again, which means: If the
// |replace_str| has |find_str| as its substring, that newly replaced part of
// |process_str| won't be processed again.
bool FindAndReplace(std::string* process_str, const std::string find_str,
                    const std::string replace_str);

// Returns true if the given string contains any debug opcode substring.
bool ContainsDebugOpcode(const char* inst);

// Returns the concatenated string from a vector of |strings|, with postfixing
// each string with the given |delimiter|. if the |skip_dictator| returns true
// for an original string, that string will be omitted.
std::string SelectiveJoin(const std::vector<const char*>& strings,
                          const std::function<bool(const char*)>& skip_dictator,
                          char delimiter = '\n');

// Concatenates a vector of strings into one string. Each string is postfixed
// with '\n'.
std::string JoinAllInsts(const std::vector<const char*>& insts);

// Concatenates a vector of strings into one string. Each string is postfixed
// with '\n'. If a string contains opcode for debug instruction, that string
// will be ignored.
std::string JoinNonDebugInsts(const std::vector<const char*>& insts);

}  // namespace spvtools

#endif  // LIBSPIRV_TEST_OPT_PASS_UTILS_H_
