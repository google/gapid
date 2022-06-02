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

#include <gmock/gmock.h>
#include <google/protobuf/descriptor.pb.h>
#include <gtest/gtest.h>

#include <iostream>
#include <limits>

#include "gapil/encoder/test/api.pb.h"
#include "gapil/encoder/test/encoder_types.h"
#include "gapil/runtime/cc/encoder.h"
#include "gapil/runtime/cc/runtime.h"

using testing::_;
using testing::DoAll;
using testing::Invoke;
using testing::Mock;
using testing::Return;
using testing::StrEq;
using testing::StrictMock;
using testing::WithArgs;

namespace gapii {

namespace {

class MockEncoder : public gapil::Encoder {
 public:
  MOCK_METHOD(int64_t, encodeBackref, (const void* object), (override));
  MOCK_METHOD(void*, encodeObject,
              (uint8_t is_group, uint32_t type, uint32_t data_size, void* data),
              (override));
  MOCK_METHOD(int64_t, encodeType,
              (const char* name, uint32_t desc_size, const void* desc),
              (override));
  MOCK_METHOD(void, sliceEncoded, (const pool_t* pool), (override));
  MOCK_METHOD(core::Arena*, arena, (), (const, override));
};

class EncoderTest : public testing::Test {
 protected:
  virtual void SetUp() {
    arena = new core::Arena();
    EXPECT_CALL(encoder, arena()).WillRepeatedly(Return(arena));
  }

  virtual void TearDown() {
    Mock::VerifyAndClear(&encoder);
    delete arena;
  }

  gapil::String makeString(const char* str) {
    return gapil::String(arena, str);
  }

  core::Arena* arena;
  StrictMock<MockEncoder> encoder;
};

// A very basic comparison, just does some spot checks.
void compare_descriptors(const google::protobuf::Descriptor* expected,
                         const void* data, uint32_t size) {
  google::protobuf::DescriptorProto actual;
  ASSERT_TRUE(actual.ParseFromArray(data, size));
  ASSERT_EQ(expected->name(), actual.name());
  ASSERT_EQ(expected->field_count(), actual.field_size());
  for (int i = 0; i < expected->field_count(); i++) {
    ASSERT_EQ(expected->field(i)->name(), actual.field(i).name())
        << "Field " << i;
    ASSERT_EQ(expected->field(i)->type(), actual.field(i).type())
        << "Field " << i;
  }
}

// Compares the values in the proto slice to the expected gapil::Slice.
template <typename T>
void compare_slice(const gapil::Slice<T>* expected,
                   const memory::Slice& actual) {
  ASSERT_EQ(expected->root(), actual.root());
  ASSERT_EQ(expected->base(), actual.base());
  ASSERT_EQ(expected->count(), actual.count());
  ASSERT_EQ(expected->size(), actual.size());
  ASSERT_EQ(expected->pool_id(), actual.pool());
}

}  // namespace

TEST_F(EncoderTest, TestCmdInts) {
  gapii::cmd::cmd_ints cmd{
      0x12345678,  // thread
      std::numeric_limits<uint8_t>::max(),
      std::numeric_limits<int8_t>::min(),
      std::numeric_limits<uint16_t>::max(),
      std::numeric_limits<int16_t>::min(),
      std::numeric_limits<uint32_t>::max(),
      std::numeric_limits<int32_t>::min(),
      std::numeric_limits<uint64_t>::max(),
      std::numeric_limits<int64_t>::min(),
  };
  gapii::cmd::cmd_intsCall call{
      0x80,
  };
  void* resultPtr1 = reinterpret_cast<void*>(0xF00D);
  void* resultPtr2 = reinterpret_cast<void*>(0xCAFE);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_ints"), _, _))
      .WillOnce(
          DoAll(WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
                  compare_descriptors(test::cmd_ints::descriptor(), desc, size);
                })),
                Return(42)));
  EXPECT_CALL(encoder, encodeObject(1, 42, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&cmd](uint32_t size, const void* data) {
                  test::cmd_ints actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(cmd.thread, actual.thread());
                  ASSERT_EQ(cmd.a, actual.a());
                  ASSERT_EQ(cmd.b, actual.b());
                  ASSERT_EQ(cmd.c, actual.c());
                  ASSERT_EQ(cmd.d, actual.d());
                  ASSERT_EQ(cmd.e, actual.e());
                  ASSERT_EQ(cmd.f, actual.f());
                  ASSERT_EQ(cmd.g, actual.g());
                  ASSERT_EQ(cmd.h, actual.h());
                })),
                Return(resultPtr1)));
  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_intsCall"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::cmd_intsCall::descriptor(), desc, size);
          })),
          Return(21)));
  EXPECT_CALL(encoder, encodeObject(1, 21, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&call](uint32_t size, const void* data) {
                  test::cmd_intsCall actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(call.result, actual.result());
                })),
                Return(resultPtr2)));

  EXPECT_EQ(resultPtr1, cmd.encode(&encoder, true));
  EXPECT_EQ(resultPtr2, call.encode(&encoder, true));
}

TEST_F(EncoderTest, TestCmdFloats) {
  gapii::cmd::cmd_floats cmd{0x10,  // thread
                             1234.5678f, 123456789.987654321};
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_floats"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::cmd_floats::descriptor(), desc, size);
          })),
          Return(22)));
  EXPECT_CALL(encoder, encodeObject(1, 22, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&cmd](uint32_t size, const void* data) {
                  test::cmd_floats actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(cmd.thread, actual.thread());
                  ASSERT_EQ(cmd.a, actual.a());
                  ASSERT_EQ(cmd.b, actual.b());
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, cmd.encode(&encoder, true));
}

TEST_F(EncoderTest, TestCmdEnums) {
  gapii::cmd::cmd_enums cmd{
      0x23,  // thread
      100,
      std::numeric_limits<int64_t>::min(),
  };
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_enums"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::cmd_enums::descriptor(), desc, size);
          })),
          Return(11)));
  EXPECT_CALL(encoder, encodeObject(1, 11, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&cmd](uint32_t size, const void* data) {
                  test::cmd_enums actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(cmd.thread, actual.thread());
                  ASSERT_EQ(cmd.e, actual.e());
                  ASSERT_EQ(cmd.e_s64, actual.e_s64());
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, cmd.encode(&encoder, true));
}

TEST_F(EncoderTest, TestCmdArrays) {
  gapii::cmd::cmd_arrays cmd{
      0x88,  // thread
      {1},
      {1, 2},
      {1, 2, 3},
  };
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_arrays"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::cmd_arrays::descriptor(), desc, size);
          })),
          Return(77)));
  EXPECT_CALL(encoder, encodeObject(1, 77, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&cmd](uint32_t size, const void* data) {
                  test::cmd_arrays actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(cmd.thread, actual.thread());
                  ASSERT_EQ(1, actual.a_size());
                  ASSERT_EQ(1, actual.a(0));
                  ASSERT_EQ(2, actual.b_size());
                  ASSERT_EQ(1, actual.b(0));
                  ASSERT_EQ(2, actual.b(1));
                  ASSERT_EQ(3, actual.c_size());
                  ASSERT_EQ(1, actual.c(0));
                  ASSERT_EQ(2, actual.c(1));
                  ASSERT_EQ(3, actual.c(2));
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, cmd.encode(&encoder, true));
}

TEST_F(EncoderTest, TestCmdPointers) {
  gapii::cmd::cmd_pointers cmd{
      0xaa,  // thread
      reinterpret_cast<uint8_t*>(0x12345678),
      reinterpret_cast<int32_t*>(0xabcdef42),
      reinterpret_cast<float*>(0x0123456789abcdef),
  };
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.cmd_pointers"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::cmd_pointers::descriptor(), desc, size);
          })),
          Return(33)));
  EXPECT_CALL(encoder, encodeObject(1, 33, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&cmd](uint32_t size, const void* data) {
                  test::cmd_pointers actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(cmd.thread, actual.thread());
                  ASSERT_EQ(reinterpret_cast<int64_t>(cmd.a), actual.a());
                  ASSERT_EQ(reinterpret_cast<int64_t>(cmd.b), actual.b());
                  ASSERT_EQ(reinterpret_cast<int64_t>(cmd.c), actual.c());
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, cmd.encode(&encoder, true));
}

TEST_F(EncoderTest, TestBasicTypes) {
  gapii::basic_types val{
      10,
      20,
      30,
      40,
      50,
      60,
      70,
      80,
      90,
      100,
      1,
      0x10,
      reinterpret_cast<uint32_t*>(0x1234),
      makeString("meow"),
  };
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.basic_types"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::basic_types::descriptor(), desc, size);
          })),
          Return(100)));
  EXPECT_CALL(encoder, encodeObject(1, 100, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&val](uint32_t size, const void* data) {
                  test::basic_types actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  ASSERT_EQ(val.ma, actual.a());
                  ASSERT_EQ(val.mb, actual.b());
                  ASSERT_EQ(val.mc, actual.c());
                  ASSERT_EQ(val.md, actual.d());
                  ASSERT_EQ(val.me, actual.e());
                  ASSERT_EQ(val.mf, actual.f());
                  ASSERT_EQ(val.mg, actual.g());
                  ASSERT_EQ(val.mh, actual.h());
                  ASSERT_EQ(val.mi, actual.i());
                  ASSERT_EQ(val.mj, actual.j());
                  ASSERT_EQ(val.mk, actual.k());
                  ASSERT_EQ(val.ml, actual.l());
                  ASSERT_EQ(reinterpret_cast<int64_t>(val.mm), actual.m());
                  EXPECT_STREQ(val.mn.c_str(), actual.n().c_str());
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, val.encode(&encoder, true));
}

TEST_F(EncoderTest, TestNestedClasses) {
  gapii::basic_types basic{
      10, 0, 0, 0, 50, 60, 0, 80, 0, 0, 1, 0, nullptr, makeString("woof"),
  };
  gapii::inner_class inner{
      basic,
  };
  gapii::nested_classes nested{
      inner,
  };
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.nested_classes"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::nested_classes::descriptor(), desc, size);
          })),
          Return(100)));
  EXPECT_CALL(encoder, encodeType(StrEq("encoder.inner_class"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::inner_class::descriptor(), desc, size);
          })),
          Return(-101)));
  EXPECT_CALL(encoder, encodeObject(1, 100, _, _))
      .WillOnce(DoAll(
          WithArgs<2, 3>(Invoke([&basic](uint32_t size, const void* data) {
            test::nested_classes actual;
            ASSERT_TRUE(actual.ParseFromArray(data, size));
            ASSERT_TRUE(actual.has_a());
            ASSERT_TRUE(actual.a().has_a());
            ASSERT_EQ(basic.ma, actual.a().a().a());
            ASSERT_EQ(basic.mb, actual.a().a().b());
            ASSERT_EQ(basic.mc, actual.a().a().c());
            ASSERT_EQ(basic.md, actual.a().a().d());
            ASSERT_EQ(basic.me, actual.a().a().e());
            ASSERT_EQ(basic.mf, actual.a().a().f());
            ASSERT_EQ(basic.mg, actual.a().a().g());
            ASSERT_EQ(basic.mh, actual.a().a().h());
            ASSERT_EQ(basic.mi, actual.a().a().i());
            ASSERT_EQ(basic.mj, actual.a().a().j());
            ASSERT_EQ(basic.mk, actual.a().a().k());
            ASSERT_EQ(basic.ml, actual.a().a().l());
            ASSERT_EQ(reinterpret_cast<int64_t>(basic.mm), actual.a().a().m());
            EXPECT_STREQ(basic.mn.c_str(), actual.a().a().n().c_str());
          })),
          Return(resultPtr)));

  EXPECT_EQ(resultPtr, nested.encode(&encoder, true));
}

TEST_F(EncoderTest, TestMapTypes) {
  gapii::map_types val(arena);
  val.ma[10] = 200;
  val.ma[20] = 100;
  val.ma[30] = 300;
  val.mb[makeString("snake")] = makeString("hiss");
  val.mb[makeString("cat")] = makeString("meow");
  val.mb[makeString("dog")] = makeString("woof");
  val.mb[makeString("fox")] = makeString("???");
  val.mc = val.ma;
  val.md = val.mb;

  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.map_types"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::map_types::descriptor(), desc, size);
          })),
          Return(100)));
  EXPECT_CALL(encoder, encodeType(StrEq("encoder.sint64_to_sint64_map"), _, _))
      .WillOnce(
          DoAll(WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
                  compare_descriptors(test::sint64_to_sint64_map::descriptor(),
                                      desc, size);
                })),
                Return(101)));
  EXPECT_CALL(encoder, encodeType(StrEq("encoder.string_to_string_map"), _, _))
      .WillOnce(
          DoAll(WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
                  compare_descriptors(test::string_to_string_map::descriptor(),
                                      desc, size);
                })),
                Return(102)));
  EXPECT_CALL(encoder, encodeBackref(val.ma.instance_ptr()))
      .WillOnce(Return(200))
      .WillOnce(Return(-200));
  EXPECT_CALL(encoder, encodeBackref(val.mb.instance_ptr()))
      .WillOnce(Return(201))
      .WillOnce(Return(-201));
  EXPECT_CALL(encoder, encodeObject(1, 100, _, _))
      .WillOnce(DoAll(
          WithArgs<2, 3>(Invoke([this, &val](uint32_t size, const void* data) {
            test::map_types actual;
            ASSERT_TRUE(actual.ParseFromArray(data, size));
            ASSERT_EQ(200, actual.a().referenceid());
            ASSERT_EQ(val.ma.count(), actual.a().keys_size());
            ASSERT_EQ(val.ma.count(), actual.a().values_size());
            for (uint64_t i = 0; i < val.ma.count(); i++) {
              ASSERT_TRUE(val.ma.contains(actual.a().keys(i)));
              ASSERT_EQ(val.ma[actual.a().keys(i)], actual.a().values(i));
            }
            ASSERT_EQ(201, actual.b().referenceid());
            ASSERT_EQ(val.mb.count(), actual.b().keys_size());
            ASSERT_EQ(val.mb.count(), actual.b().values_size());
            for (uint64_t i = 0; i < val.mb.count(); i++) {
              gapil::String key = makeString(actual.b().keys(i).c_str());
              ASSERT_TRUE(val.mb.contains(key)) << "key " << actual.b().keys(i);
              EXPECT_STREQ(val.mb[key].c_str(), actual.b().values(i).c_str());
            }
            ASSERT_EQ(200, actual.c().referenceid());
            ASSERT_EQ(0, actual.c().keys_size());
            ASSERT_EQ(0, actual.c().values_size());
            ASSERT_EQ(201, actual.d().referenceid());
            ASSERT_EQ(0, actual.d().keys_size());
            ASSERT_EQ(0, actual.d().values_size());
          })),
          Return(resultPtr)));

  EXPECT_EQ(resultPtr, val.encode(&encoder, true));
}

TEST_F(EncoderTest, TestRefTypes) {
  gapil::Ref<gapii::basic_types> basic1 =
      gapil::Ref<gapii::basic_types>::create(arena, 10, 0, 0, 0, 50, 60, 0, 80,
                                             0, 0, 1, 0, nullptr,
                                             makeString("slurp"));
  gapil::Ref<gapii::inner_class> inner = gapil::Ref<gapii::inner_class>::create(
      arena, basic_types(20, 0, 0, 0, 40, 70, 0, 60, 0, 0, 2, 0, nullptr,
                         makeString("crunch")));
  gapii::ref_types val(basic1, inner, basic1, inner);
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.ref_types"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::ref_types::descriptor(), desc, size);
          })),
          Return(-100)));
  EXPECT_CALL(encoder, encodeBackref(val.ma.get()))
      .WillOnce(Return(200))
      .WillOnce(Return(-200));
  EXPECT_CALL(encoder, encodeBackref(val.mb.get()))
      .WillOnce(Return(201))
      .WillOnce(Return(-201));
  EXPECT_CALL(encoder, encodeObject(1, 100, _, _))
      .WillOnce(DoAll(
          WithArgs<2, 3>(Invoke([&val](uint32_t size, const void* data) {
            test::ref_types actual;
            ASSERT_TRUE(actual.ParseFromArray(data, size));
            ASSERT_EQ(200, actual.a().referenceid());
            ASSERT_EQ(val.ma->ma, actual.a().value().a());
            ASSERT_EQ(val.ma->mb, actual.a().value().b());
            ASSERT_EQ(val.ma->mc, actual.a().value().c());
            ASSERT_EQ(val.ma->md, actual.a().value().d());
            ASSERT_EQ(val.ma->me, actual.a().value().e());
            ASSERT_EQ(val.ma->mf, actual.a().value().f());
            ASSERT_EQ(val.ma->mg, actual.a().value().g());
            ASSERT_EQ(val.ma->mh, actual.a().value().h());
            ASSERT_EQ(val.ma->mi, actual.a().value().i());
            ASSERT_EQ(val.ma->mj, actual.a().value().j());
            ASSERT_EQ(val.ma->mk, actual.a().value().k());
            ASSERT_EQ(val.ma->ml, actual.a().value().l());
            ASSERT_EQ(reinterpret_cast<int64_t>(val.ma->mm),
                      actual.a().value().m());
            EXPECT_STREQ(val.ma->mn.c_str(), actual.a().value().n().c_str());
            ASSERT_EQ(201, actual.b().referenceid());
            ASSERT_EQ(val.mb->ma.ma, actual.b().value().a().a());
            ASSERT_EQ(val.mb->ma.mb, actual.b().value().a().b());
            ASSERT_EQ(val.mb->ma.mc, actual.b().value().a().c());
            ASSERT_EQ(val.mb->ma.md, actual.b().value().a().d());
            ASSERT_EQ(val.mb->ma.me, actual.b().value().a().e());
            ASSERT_EQ(val.mb->ma.mf, actual.b().value().a().f());
            ASSERT_EQ(val.mb->ma.mg, actual.b().value().a().g());
            ASSERT_EQ(val.mb->ma.mh, actual.b().value().a().h());
            ASSERT_EQ(val.mb->ma.mi, actual.b().value().a().i());
            ASSERT_EQ(val.mb->ma.mj, actual.b().value().a().j());
            ASSERT_EQ(val.mb->ma.mk, actual.b().value().a().k());
            ASSERT_EQ(val.mb->ma.ml, actual.b().value().a().l());
            ASSERT_EQ(reinterpret_cast<int64_t>(val.mb->ma.mm),
                      actual.b().value().a().m());
            EXPECT_STREQ(val.mb->ma.mn.c_str(),
                         actual.b().value().a().n().c_str());
            ASSERT_EQ(200, actual.c().referenceid());
            ASSERT_FALSE(actual.c().has_value());
            ASSERT_EQ(201, actual.d().referenceid());
            ASSERT_FALSE(actual.d().has_value());
            ASSERT_TRUE(&val == &val);
          })),
          Return(resultPtr)));

  EXPECT_EQ(resultPtr, val.encode(&encoder, true));
}

TEST_F(EncoderTest, TestSliceTypes) {
  pool pool1{1, 0x11}, pool2{1, 0x12};
  gapii::slice_types val(
      gapil::Slice<uint8_t>(nullptr, 0x1000, 0x2000, 0x10, 0x10),
      gapil::Slice<float>(&pool1, 0x2000, 0x3000, 0x80, 0x20),
      gapil::Slice<int_types>(&pool2, 0x3000, 0x4000, 0xc0, 0x30));
  void* resultPtr = reinterpret_cast<void*>(0xF00D);

  EXPECT_CALL(encoder, encodeType(StrEq("encoder.slice_types"), _, _))
      .WillOnce(DoAll(
          WithArgs<1, 2>(Invoke([](uint32_t size, const void* desc) {
            compare_descriptors(test::slice_types::descriptor(), desc, size);
          })),
          Return(-100)));
  EXPECT_CALL(encoder, sliceEncoded(nullptr)).Times(1);
  EXPECT_CALL(encoder, sliceEncoded(&pool1)).Times(1);
  EXPECT_CALL(encoder, sliceEncoded(&pool2)).Times(1);
  EXPECT_CALL(encoder, encodeObject(1, 100, _, _))
      .WillOnce(
          DoAll(WithArgs<2, 3>(Invoke([&val](uint32_t size, const void* data) {
                  test::slice_types actual;
                  ASSERT_TRUE(actual.ParseFromArray(data, size));
                  compare_slice(&val.ma, actual.a());
                  compare_slice(&val.mb, actual.b());
                  compare_slice(&val.mc, actual.c());
                })),
                Return(resultPtr)));

  EXPECT_EQ(resultPtr, val.encode(&encoder, true));
}

}  // namespace gapii
