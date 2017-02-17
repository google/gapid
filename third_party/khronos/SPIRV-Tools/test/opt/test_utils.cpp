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

#include <gtest/gtest.h>

#include "pass_utils.h"

namespace {

using namespace spvtools;

TEST(JoinAllInsts, Cases) {
  EXPECT_EQ("", JoinAllInsts({}));
  EXPECT_EQ("a\n", JoinAllInsts({"a"}));
  EXPECT_EQ("a\nb\n", JoinAllInsts({"a", "b"}));
  EXPECT_EQ("a\nb\nc\n", JoinAllInsts({"a", "b", "c"}));
  EXPECT_EQ("hello,\nworld!\n\n\n", JoinAllInsts({"hello,", "world!", "\n"}));
}

TEST(JoinNonDebugInsts, Cases) {
  EXPECT_EQ("", JoinNonDebugInsts({}));
  EXPECT_EQ("a\n", JoinNonDebugInsts({"a"}));
  EXPECT_EQ("", JoinNonDebugInsts({"OpName"}));
  EXPECT_EQ("a\nb\n", JoinNonDebugInsts({"a", "b"}));
  EXPECT_EQ("", JoinNonDebugInsts({"OpName", "%1 = OpString \"42\""}));
  EXPECT_EQ("Opstring\n", JoinNonDebugInsts({"OpName", "Opstring"}));
  EXPECT_EQ("the only remaining string\n",
            JoinNonDebugInsts(
                {"OpSourceContinued", "OpSource", "OpSourceExtension",
                 "lgtm OpName", "hello OpMemberName", "this is a OpString",
                 "lonely OpLine", "happy OpNoLine", "OpModuleProcessed",
                 "the only remaining string"}));
}

namespace {
struct SubstringReplacementTestCase {
  const char* orig_str;
  const char* find_substr;
  const char* replace_substr;
  const char* expected_str;
  bool replace_should_succeed;
};
}
using FindAndReplaceTest =
    ::testing::TestWithParam<SubstringReplacementTestCase>;

TEST_P(FindAndReplaceTest, SubstringReplacement) {
  auto process = std::string(GetParam().orig_str);
  EXPECT_EQ(GetParam().replace_should_succeed,
            FindAndReplace(&process, GetParam().find_substr,
                           GetParam().replace_substr))
      << "Original string: " << GetParam().orig_str
      << " replace: " << GetParam().find_substr
      << " to: " << GetParam().replace_substr
      << " should returns: " << GetParam().replace_should_succeed;
  EXPECT_STREQ(GetParam().expected_str, process.c_str())
      << "Original string: " << GetParam().orig_str
      << " replace: " << GetParam().find_substr
      << " to: " << GetParam().replace_substr
      << " expected string: " << GetParam().expected_str;
}

INSTANTIATE_TEST_CASE_P(
    SubstringReplacement, FindAndReplaceTest,
    ::testing::ValuesIn(std::vector<SubstringReplacementTestCase>({
        // orig string, find substring, replace substring, expected string,
        // replacement happened
        {"", "", "", "", false},
        {"", "b", "", "", false},
        {"", "", "c", "", false},
        {"", "a", "b", "", false},

        {"a", "", "c", "a", false},
        {"a", "b", "c", "a", false},
        {"a", "b", "", "a", false},
        {"a", "a", "", "", true},
        {"a", "a", "b", "b", true},

        {"ab", "a", "b", "bb", true},
        {"ab", "a", "", "b", true},
        {"ab", "b", "", "a", true},
        {"ab", "ab", "", "", true},
        {"ab", "ab", "cd", "cd", true},
        {"bc", "abc", "efg", "bc", false},

        {"abc", "ab", "bc", "bcc", true},
        {"abc", "ab", "", "c", true},
        {"abc", "bc", "", "a", true},
        {"abc", "bc", "d", "ad", true},
        {"abc", "a", "123", "123bc", true},
        {"abc", "ab", "a", "ac", true},
        {"abc", "a", "aab", "aabbc", true},
        {"abc", "abcd", "efg", "abc", false},
    })));
}  // anonymous namespace
